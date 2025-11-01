package parser

import (
	"github.com/samborkent/cog/internal/ast"
	"github.com/samborkent/cog/internal/tokens"
	"github.com/samborkent/cog/internal/types"
)

type SymbolKind uint8

const (
	SymbolKindVariable SymbolKind = iota
	SymbolKindConstant
	SymbolKindType
	SymbolKindField
)

type Symbol struct {
	Identifier *ast.Identifier
	Scope      Scope
	Kind       SymbolKind
}

func (s Symbol) Type() types.Type {
	return s.Identifier.ValueType
}

var None = &ast.Identifier{
	Token: tokens.Token{
		Type:    tokens.Identifier,
		Literal: "_",
	},
	Name:      "_",
	ValueType: types.None,
}

type SymbolTable struct {
	Outer *SymbolTable

	table     map[string]Symbol
	goimports map[string]*ast.Identifier
	fields    map[string]map[string]Symbol
}

func NewSymbolTable() *SymbolTable {
	table := make(map[string]Symbol)

	table["_"] = Symbol{
		Identifier: None,
		Scope:      LocalScope,
		Kind:       SymbolKindVariable,
	}

	return &SymbolTable{
		table:     table,
		goimports: make(map[string]*ast.Identifier),
		fields:    make(map[string]map[string]Symbol),
	}
}

func NewEnclosedSymbolTable(outer *SymbolTable) *SymbolTable {
	s := NewSymbolTable()
	s.Outer = outer
	s.goimports = outer.goimports

	return s
}

func (s *SymbolTable) Define(ident *ast.Identifier, kind SymbolKind) {
	if ident.Name == "" {
		panic("empty identifier")
	}

	if ident.ValueType == nil {
		ident.ValueType = types.None
	}

	symbol := Symbol{
		Identifier: ident,
		Scope:      LocalScope,
		Kind:       kind,
	}

	if s.Outer == nil {
		symbol.Scope = GlobalScope
	}

	s.table[ident.Name] = symbol

	if ident.Name != "" && (kind == SymbolKindVariable || kind == SymbolKindConstant) {
		switch ident.ValueType.Underlying().Kind() {
		case types.StructKind:
			structType, ok := ident.ValueType.Underlying().(*types.Struct)
			if !ok {
				break
			}

			_, ok = s.fields[ident.Name]
			if ok {
				break
			}

			s.fields[ident.Name] = make(map[string]Symbol, len(structType.Fields))

			for _, field := range structType.Fields {
				s.fields[ident.Name][field.Name] = Symbol{
					Identifier: &ast.Identifier{
						Name:      field.Name,
						ValueType: field.Type,
						Exported:  field.Exported,
					},
					Scope: StructScope,
					Kind:  SymbolKindField,
				}
			}
		}
	}
}

func (s *SymbolTable) DefineEnumValue(selector string, field *ast.Identifier) {
	if field.Name == "" {
		panic("empty enum value identifier")
	}

	_, ok := s.fields[selector]
	if !ok {
		s.fields[selector] = make(map[string]Symbol)
	}

	s.fields[selector][field.Name] = Symbol{
		Identifier: field,
		Scope:      EnumScope,
		Kind:       SymbolKindField,
	}
}

func (s *SymbolTable) DefineGlobal(ident *ast.Identifier, kind SymbolKind) {
	if ident.Name == "" {
		panic("empty global identifier")
	}

	if ident.ValueType == nil {
		ident.ValueType = types.None
	}

	s.table[ident.Name] = Symbol{
		Identifier: ident,
		Scope:      ScanScope,
		Kind:       kind,
	}
}

func (s *SymbolTable) DefineGoImport(ident *ast.Identifier) {
	if ident.Name == "" {
		panic("empty go import")
	}

	if ident.ValueType == nil {
		ident.ValueType = types.None
	}

	s.goimports[ident.Name] = ident
}

func (s *SymbolTable) Resolve(name string) (Symbol, bool) {
	obj, ok := s.table[name]
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
	fields, ok := s.fields[typeName]
	if !ok && s.Outer != nil {
		symbol, ok := s.Outer.ResolveField(typeName, field)
		if !ok {
			return Symbol{}, false
		}

		return symbol, true
	}

	symbol, ok := fields[field]
	return symbol, ok
}

func (s *SymbolTable) ResolveGoImport(name string) (*ast.Identifier, bool) {
	ident, ok := s.goimports[name]
	return ident, ok
}

func (s *SymbolTable) Update(name string, t types.Type) {
	if symbol, ok := s.table[name]; ok {
		symbol.Identifier.ValueType = t
		s.table[name] = symbol
	}
}
