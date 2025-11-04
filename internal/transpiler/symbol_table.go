package transpiler

import (
	"fmt"
	goast "go/ast"

	"github.com/samborkent/cog/internal/ast"
)

type SymbolTable struct {
	Outer *SymbolTable

	table    map[string]*goast.Ident
	dynamics map[string]*ast.Identifier
}

func NewSymbolTable() *SymbolTable {
	table := make(map[string]*goast.Ident)

	return &SymbolTable{
		table:    table,
		dynamics: make(map[string]*ast.Identifier),
	}
}

func NewEnclosedSymbolTable(outer *SymbolTable) *SymbolTable {
	s := NewSymbolTable()
	s.Outer = outer

	return s
}

func (s *SymbolTable) Define(name string) *goast.Ident {
	ident, ok := s.table[name]
	if ok {
		return ident
	}

	s.table[name] = &goast.Ident{Name: "_"}

	return s.table[name]
}

func (s *SymbolTable) DefineDynamic(ident *ast.Identifier) {
	if s.Outer != nil {
		panic("cannot define dynamically scoped variables outside of package scope")
	}

	name := convertExport(ident.Name, ident.Exported)
	s.dynamics[name] = ident
	s.Define(joinStr(name, "Key"))
	s.Define(joinStr(name, "Default"))
}

func (s *SymbolTable) MarkUsed(name string) {
	ident, ok := s.table[name]
	if !ok {
		if s.Outer != nil {
			s.Outer.MarkUsed(name)
			return
		}

		panic(fmt.Sprintf("identifier %q is not defined", name))
	}

	ident.Name = name
}

func (s *SymbolTable) Resolve(name string) (*goast.Ident, bool) {
	ident, ok := s.table[name]
	if !ok && s.Outer != nil {
		ident, ok = s.Outer.Resolve(name)
		if !ok {
			return nil, false
		}

		return ident, true
	}

	return ident, ok
}

func (s *SymbolTable) ResolveDynamic(name string) (*ast.Identifier, bool) {
	if s.Outer != nil {
		return s.Outer.ResolveDynamic(name)
	}

	ident, ok := s.dynamics[name]
	return ident, ok
}
