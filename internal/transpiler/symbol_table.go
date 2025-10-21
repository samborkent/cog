package transpiler

import (
	"fmt"
	goast "go/ast"
	"maps"
	"slices"
)

type SymbolTable struct {
	Outer *SymbolTable

	table map[string]*goast.Ident
}

func NewSymbolTable() *SymbolTable {
	table := make(map[string]*goast.Ident)

	return &SymbolTable{
		table: table,
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

func (s *SymbolTable) collect() []string {
	idents := []string{}

	if len(s.table) > 0 {
		idents = append(idents, slices.Collect(maps.Keys(s.table))...)
	}

	if s.Outer != nil {
		idents = append(idents, s.Outer.collect()...)
	}

	return idents
}
