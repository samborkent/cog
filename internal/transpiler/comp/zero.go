package comp

import goast "go/ast"

func ZeroValue(t goast.Expr) *goast.StarExpr {
	return &goast.StarExpr{
		X: &goast.CallExpr{
			Fun:  &goast.Ident{Name: "new"},
			Args: []goast.Expr{t},
		},
	}
}
