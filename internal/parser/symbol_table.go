package parser

import (
	"iter"
	"runtime"

	"github.com/samborkent/cog/internal/ast"
	"github.com/samborkent/cog/internal/isync"
	"github.com/samborkent/cog/internal/tokens"
	"github.com/samborkent/cog/internal/types"
)

type Symbol struct {
	Identifier *ast.Identifier
	Scope      Scope
	Used       bool
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
	Path         string               // import path (e.g. "geom")
	Name         string               // package name (last segment of path)
	ExportValues map[string]Symbol    // exported identifiers from the imported package
	ExportTypes  map[string]*ast.Type // exported types from the imported package
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

	idents     *isync.Map[string, Symbol]          // Key: identifier
	types      *isync.Map[string, *ast.Type]       // Key: type name
	goimports  *isync.Map[string, *ast.Identifier] // key: package name
	cogimports *isync.Map[string, *CogImport]      // key: package name
	checked    *isync.Map[string, checkState]      // option/result variables verified in this scope

	ownership      *isync.Map[string, ownershipState]                     // walks up scope chains
	fieldOwnership *isync.Map[string, *isync.Map[string, ownershipState]] // per-field within structs
	borrowCounts   *isync.Map[string, int32]                              // per-scope only, does NOT walk up
}

func NewSymbolTable() *SymbolTable {
	return NewSymbolTableWithConcurrency(false)
}

func NewSymbolTableWithConcurrency(concurrent bool) *SymbolTable {
	var opts []isync.Option
	if concurrent {
		opts = []isync.Option{isync.WithConcurrency()}
	}

	idents := isync.NewMap[string, Symbol](opts...)

	idents.Store("_", Symbol{
		Identifier: None,
		Scope:      LocalScope,
	})

	return &SymbolTable{
		concurrent:     concurrent,
		idents:         idents,
		types:          isync.NewMap[string, *ast.Type](opts...),
		goimports:      isync.NewMap[string, *ast.Identifier](opts...),
		cogimports:     isync.NewMap[string, *CogImport](opts...),
		checked:        isync.NewMap[string, checkState](opts...),
		ownership:      isync.NewMap[string, ownershipState](opts...),
		fieldOwnership: isync.NewMap[string, *isync.Map[string, ownershipState]](opts...),
		borrowCounts:   isync.NewMap[string, int32](opts...),
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

// Global gets global symbol table.
func (s *SymbolTable) Global() *SymbolTable {
	if s.Outer == nil {
		return s
	}

	return s.Outer.Global()
}

// Symbols iterates over all symbols in this table and its parents.
func (s *SymbolTable) Symbols() iter.Seq[Symbol] {
	return func(yield func(Symbol) bool) {
		_ = s.everyIdent(yield)
	}
}

// Types iterates over all types in this table and its parents.
func (s *SymbolTable) Types() iter.Seq[*ast.Type] {
	return func(yield func(*ast.Type) bool) {
		_ = s.everyType(yield)
	}
}

// Efficiently yield every identifier of this symbol table and its parents.
func (s *SymbolTable) everyIdent(f func(Symbol) bool) bool {
	if s == nil || s.idents == nil {
		return true
	}

	// Yield every identifier of parent recursively if set.
	result := s.Outer == nil || s.Outer.everyIdent(f)

	// Yield every identifier of this symbol table.
	for _, ident := range s.idents.Map {
		result = f(ident) && result
	}

	return result
}

// Efficiently yield every type of this symbol table and its parents.
func (s *SymbolTable) everyType(f func(*ast.Type) bool) bool {
	if s == nil || s.types == nil {
		return true
	}

	// Yield every type of parent recursively if set.
	result := s.Outer == nil || s.Outer.everyType(f)

	// Yield every type of this symbol table.
	for _, typ := range s.types.Map {
		result = f(typ) && result
	}

	return result
}

func (s *SymbolTable) DefineIdent(ident *ast.Identifier) {
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

	s.idents.Store(ident.Token.Literal, symbol)
}

func (s *SymbolTable) DefineType(typ *ast.Type) {
	s.types.Store(typ.Alias.Name, typ)
}

func (s *SymbolTable) DefineGlobalIdent(ident *ast.Identifier) {
	// If a forward value stub exists (created during globalsPass for
	// forward references), resolve it by copying the real declaration's
	// info into the existing stub. All expression nodes referencing the
	// stub share the same pointer, so mutating it updates all referents.
	existing, ok := s.idents.Load(ident.Token.Literal)
	if ok && existing.Scope == ScanScope && types.IsNone(existing.Identifier.ValueType) {
		existing.Identifier.ValueType = ident.ValueType
		existing.Identifier.Qualifier = ident.Qualifier
		existing.Identifier.Exported = ident.Exported
		return
	}

	s.DefineIdent(ident)

	symbol, ok := s.idents.Load(ident.Token.Literal)
	if !ok {
		panic("symbol not found after definition")
	}

	symbol.Scope = ScanScope
	s.idents.Store(ident.Token.Literal, symbol)
}

func (s *SymbolTable) DefineGlobalType(typ *ast.Type) {
	// If a forward value stub exists (created during globalsPass for
	// forward references), resolve it by copying the real declaration's
	// info into the existing stub. All expression nodes referencing the
	// stub share the same pointer, so mutating it updates all referents.
	existing, ok := s.types.Load(typ.Alias.Name)
	if ok && types.IsNone(existing.Alias.Derived) {
		// Register methods of existing type on new type if any.
		typ.Alias.RegisterMethods(existing.Alias.Methods()...)
		s.types.Store(typ.Alias.Name, typ)
		return
	}

	if typ.Alias.Derived == nil {
		typ.Alias.Derived = types.None
	}

	s.DefineType(typ)
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

func (s *SymbolTable) ResolveIdent(name string) (Symbol, bool) {
	obj, ok := s.idents.Load(name)
	if !ok && s.Outer != nil {
		obj, ok = s.Outer.ResolveIdent(name)
	}

	return obj, ok
}

func (s *SymbolTable) ResolveType(name string) (*ast.Type, bool) {
	obj, ok := s.types.Load(name)
	if !ok && s.Outer != nil {
		obj, ok = s.Outer.ResolveType(name)
	}

	return obj, ok
}

// MarkUsed marks a symbol as used. It walks up scope chains to find the symbol.
func (s *SymbolTable) MarkUsed(name string) {
	sym, ok := s.idents.Load(name)
	if ok {
		sym.Used = true
		s.idents.Store(name, sym)
		return
	}

	if s.Outer != nil {
		s.Outer.MarkUsed(name)
	}
}

// CheckUnused returns symbols that are defined but never used.
// Global/package-scope symbols, exported variables, type declarations, blank
// identifiers, and dynamic variables are exempt from this check.
func (s *SymbolTable) CheckUnused() iter.Seq2[string, Symbol] {
	return func(yield func(string, Symbol) bool) {
		for name, sym := range s.idents.All() {
			// Skip variables whicha are already marked as used.
			if sym.Used {
				continue
			}

			// Skip omitted variables.
			if name == "_" {
				continue
			}

			// Global variables may be unused.
			if sym.Scope == GlobalScope || sym.Scope == ScanScope {
				continue
			}

			// Exported variables may be unused.
			if sym.Identifier.Exported {
				continue
			}

			// Dynamic variables may be unused.
			if sym.Identifier.Qualifier == ast.QualifierDynamic {
				continue
			}

			if !yield(name, sym) {
				return
			}
		}
	}
}

func (s *SymbolTable) ResolveGoImport(name string) (*ast.Identifier, bool) {
	return s.goimports.Load(name)
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

func (s *SymbolTable) FillExports(imp *CogImport) {
	for name, symbol := range s.idents.All() {
		if symbol.Identifier.Exported {
			imp.ExportValues[name] = symbol
		}
	}

	for name, typ := range s.types.All() {
		if typ.Alias.Exported {
			imp.ExportTypes[name] = typ
		}
	}
}
