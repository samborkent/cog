package parser

import (
	"fmt"
	"runtime"

	"github.com/samborkent/cog/internal/ast"
	isync "github.com/samborkent/cog/internal/sync"
	"github.com/samborkent/cog/internal/tokens"
	"github.com/samborkent/cog/internal/types"
)

type Symbol struct {
	Identifier *ast.Identifier
	Scope      Scope
}

func (s Symbol) Type() types.Type {
	return s.Identifier.ValueType
}

var None = &ast.Identifier{
	Token: tokens.Token{
		Type:    tokens.Identifier,
		Literal: "_",
	},
	ValueType: types.None,
	Qualifier: ast.QualifierVariable,
}

// CogImport represents an imported cog package.
type CogImport struct {
	Path    string            // import path (e.g. "geom")
	Name    string            // package name (last segment of path)
	Exports map[string]Symbol // exported symbols from the imported package
}

// checkState tracks which accesses are safe for a checked option/result variable.
type checkState uint8

const (
	checkValue checkState = 1 << iota // value access is safe (? check proven OK)
	checkError                        // error access is safe (? check proven error)
)

type SymbolTable struct {
	Outer *SymbolTable

	concurrent bool

	table      *isync.Map[string, Symbol]
	goimports  *isync.Map[string, *ast.Identifier] // key: package name
	cogimports *isync.Map[string, *CogImport]      // key: package name
	fields     *isync.Map[string, *isync.Map[string, Symbol]]
	checked    *isync.Map[string, checkState] // option/result variables verified in this scope
}

func NewSymbolTable() *SymbolTable {
	return NewSymbolTableWithConcurrency(false)
}

func NewSymbolTableWithConcurrency(concurrent bool) *SymbolTable {
	var opts []isync.Option
	if concurrent {
		opts = []isync.Option{isync.WithConcurrency()}
	}

	table := isync.NewMap[string, Symbol](opts...)

	table.Store("_", Symbol{
		Identifier: None,
		Scope:      LocalScope,
	})

	return &SymbolTable{
		concurrent: concurrent,
		table:      table,
		goimports:  isync.NewMap[string, *ast.Identifier](opts...),
		cogimports: isync.NewMap[string, *CogImport](opts...),
		fields:     isync.NewMap[string, *isync.Map[string, Symbol]](opts...),
		checked:    isync.NewMap[string, checkState](opts...),
	}
}

// opts returns the construction options matching the table's concurrency mode.
func (s *SymbolTable) opts() []isync.Option {
	if s.concurrent {
		return []isync.Option{isync.WithConcurrency()}
	}

	return nil
}

func NewSymbolTableAuto(fileCount, cutoff int) *SymbolTable {
	concurrent := fileCount >= cutoff && runtime.GOMAXPROCS(-1) > 1

	return NewSymbolTableWithConcurrency(concurrent)
}

func NewEnclosedSymbolTable(outer *SymbolTable) *SymbolTable {
	s := NewSymbolTableWithConcurrency(outer.concurrent)
	s.Outer = outer
	s.goimports = outer.goimports
	s.cogimports = outer.cogimports

	return s
}

func (s *SymbolTable) Define(ident *ast.Identifier) {
	if ident.Token.Literal == "" {
		panic("empty identifier")
	}

	if ident.ValueType == nil {
		ident.ValueType = types.None
	}

	symbol := Symbol{
		Identifier: ident,
		Scope:      LocalScope,
	}

	if s.Outer == nil {
		symbol.Scope = GlobalScope
	}

	s.table.Store(ident.Token.Literal, symbol)

	// TODO: investigate why this check was here
	// if ident.Qualifier != ast.QualifierType {
	switch ident.ValueType.Kind() {
	case types.StructKind:
		structType, ok := ident.ValueType.Underlying().(*types.Struct)
		if !ok {
			break
		}

		_, ok = s.fields.Load(ident.Token.Literal)
		if ok {
			break
		}

		var fields *isync.Map[string, Symbol]
		fields = isync.NewMap[string, Symbol](s.opts()...)
		s.fields.Store(ident.Token.Literal, fields)

		for _, field := range structType.Fields {
			fields.Store(field.Name, Symbol{
				Identifier: &ast.Identifier{
					Token: tokens.Token{
						Type:    tokens.Identifier,
						Literal: field.Name,
					},
					ValueType: field.Type,
					Exported:  field.Exported,
				},
				Scope: StructScope,
			})
		}
	}
	// }
}

func (s *SymbolTable) DefineMethod(receiver string, method *ast.Identifier) error {
	if method.Qualifier != ast.QualifierMethod {
		panic("DefineMethod may only be called for method identifiers")
	}

	fields, ok := s.fields.Load(receiver)
	if !ok {
		fields = isync.NewMap[string, Symbol](s.opts()...)
		s.fields.Store(receiver, fields)
	}

	_, ok = fields.Load(method.Token.Literal)
	if ok {
		return fmt.Errorf("method name conflict: field with name %q already exists for type %q", method.String(), receiver)
	}

	fields.Store(method.Token.Literal, Symbol{
		Identifier: method,
		Scope:      StructScope,
	})

	return nil
}

func (s *SymbolTable) DefineEnumValue(selector string, field *ast.Identifier) {
	if field.Token.Literal == "" {
		panic("empty enum value identifier")
	}

	fields, ok := s.fields.Load(selector)
	if !ok {
		fields = isync.NewMap[string, Symbol](s.opts()...)
		s.fields.Store(selector, fields)
	}

	fields.Store(field.Token.Literal, Symbol{
		Identifier: field,
		Scope:      EnumScope,
	})
}

func (s *SymbolTable) DefineGlobal(ident *ast.Identifier) {
	// If a forward value stub exists (created during globalsPass for
	// forward references), resolve it by copying the real declaration's
	// info into the existing stub. All expression nodes referencing the
	// stub share the same pointer, so mutating it updates all referents.
	existing, ok := s.table.Load(ident.Token.Literal)
	if ok && existing.Scope == ScanScope && types.IsNone(existing.Identifier.ValueType) {
		existing.Identifier.ValueType = ident.ValueType
		existing.Identifier.Qualifier = ident.Qualifier
		existing.Identifier.Exported = ident.Exported

		// Register struct fields so selectors (e.g. val.x) resolve.
		if ident.ValueType != nil && ident.ValueType.Kind() == types.StructKind {
			if st, ok := ident.ValueType.Underlying().(*types.Struct); ok {
				if _, exists := s.fields.Load(ident.Token.Literal); !exists {
					fields := isync.NewMap[string, Symbol](s.opts()...)
					s.fields.Store(ident.Token.Literal, fields)

					for _, field := range st.Fields {
						fields.Store(field.Name, Symbol{
							Identifier: &ast.Identifier{
								Token: tokens.Token{
									Type:    tokens.Identifier,
									Literal: field.Name,
								},
								ValueType: field.Type,
								Exported:  field.Exported,
							},
							Scope: StructScope,
						})
					}
				}
			}
		}

		return
	}

	s.Define(ident)

	symbol, ok := s.table.Load(ident.Token.Literal)
	if !ok {
		panic("symbol not found after definition")
	}

	symbol.Scope = ScanScope
	s.table.Store(ident.Token.Literal, symbol)
}

func (s *SymbolTable) DefineGoImport(ident *ast.Identifier) {
	if ident.Token.Literal == "" {
		panic("empty go import")
	}

	if ident.ValueType == nil {
		ident.ValueType = types.None
	}

	s.goimports.Store(ident.Token.Literal, ident)
}

func (s *SymbolTable) Resolve(name string) (Symbol, bool) {
	obj, ok := s.table.Load(name)
	if !ok && s.Outer != nil {
		obj, ok = s.Outer.Resolve(name)
		if !ok {
			return Symbol{}, false
		}

		return obj, true
	}

	return obj, ok
}

func (s *SymbolTable) ResolveField(typeName, field string) (Symbol, bool) {
	fields, ok := s.fields.Load(typeName)
	if !ok {
		if s.Outer != nil {
			return s.Outer.ResolveField(typeName, field)
		}

		return Symbol{}, false
	}

	return fields.Load(field)
}

func (s *SymbolTable) ResolveGoImport(name string) (*ast.Identifier, bool) {
	return s.goimports.Load(name)
}

// ForEachGlobal iterates over all symbols in the root (global) table.
func (s *SymbolTable) ForEachGlobal(fn func(name string, sym Symbol)) {
	root := s
	for root.Outer != nil {
		root = root.Outer
	}

	for name, sym := range root.table.All() {
		fn(name, sym)
	}
}

func (s *SymbolTable) Update(name string, t types.Type) {
	if symbol, ok := s.table.Load(name); ok {
		symbol.Identifier.ValueType = t
		// TODO: redundant because identifier is pointer?
		s.table.Store(name, symbol)
	}
}

func (s *SymbolTable) DefineCogImport(imp *CogImport) {
	s.cogimports.Store(imp.Name, imp)
}

func (s *SymbolTable) ResolveCogImport(name string) (*CogImport, bool) {
	return s.cogimports.Load(name)
}

// CogImports returns all registered cog imports.
func (s *SymbolTable) CogImports() *isync.Map[string, *CogImport] {
	return s.cogimports
}

// MarkChecked records that an option/result variable's value has been checked in this scope.
func (s *SymbolTable) MarkChecked(name string, state checkState) {
	s.checked.Store(name, state)
}

// ClearChecked removes must-check state for a name in the current scope.
func (s *SymbolTable) ClearChecked(name string) {
	s.checked.Delete(name)
}

// IsValueChecked reports whether the named variable's value is safe to access.
func (s *SymbolTable) IsValueChecked(name string) bool {
	if state, ok := s.checked.Load(name); ok && state&checkValue != 0 {
		return true
	}

	if s.Outer != nil {
		return s.Outer.IsValueChecked(name)
	}

	return false
}

// IsErrorChecked reports whether the named variable's error is safe to access.
func (s *SymbolTable) IsErrorChecked(name string) bool {
	if state, ok := s.checked.Load(name); ok && state&checkError != 0 {
		return true
	}

	if s.Outer != nil {
		return s.Outer.IsErrorChecked(name)
	}

	return false
}
