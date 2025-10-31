package comp

import (
	goast "go/ast"
	"go/token"
)

func ContextArg() *goast.Field {
	return &goast.Field{
		Names: []*goast.Ident{{Name: "ctx"}},
		Type:  ContextType(),
	}
}

func ContextType() *goast.SelectorExpr {
	return &goast.SelectorExpr{
		X:   &goast.Ident{Name: "context"},
		Sel: &goast.Ident{Name: "Context"},
	}
}

func ContextMain(ident *goast.Ident) *goast.AssignStmt {
	return &goast.AssignStmt{
		Lhs: []goast.Expr{ident},
		Tok: token.DEFINE,
		Rhs: []goast.Expr{
			&goast.CallExpr{
				Fun: &goast.SelectorExpr{
					X:   &goast.Ident{Name: "context"},
					Sel: &goast.Ident{Name: "Background"},
				},
			},
		},
	}
}
