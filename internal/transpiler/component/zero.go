package component

import goast "go/ast"

var newIdent = &goast.Ident{Name: "new"}

func ZeroValue(t goast.Expr) *goast.StarExpr {
	return &goast.StarExpr{
		X: &goast.CallExpr{
			Fun:  newIdent,
			Args: []goast.Expr{t},
		},
	}
}
