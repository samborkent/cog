package component

import goast "go/ast"

const newFunc = "new"

func ZeroValue(t goast.Expr) *goast.StarExpr {
	return &goast.StarExpr{
		X: &goast.CallExpr{
			Fun:  &goast.Ident{Name: newFunc},
			Args: []goast.Expr{t},
		},
	}
}
