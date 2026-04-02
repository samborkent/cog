package component

import goast "go/ast"

var (
	ImportCmp  = &goast.Ident{Name: "go_cmp"}
	CmpOrdered = &goast.SelectorExpr{
		X:   ImportCmp,
		Sel: &goast.Ident{Name: "Ordered"},
	}

	Any        = &goast.Ident{Name: "any"}
	Comparable = &goast.Ident{Name: "comparable"}
)
