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

func ContextMain(ident *goast.Ident) *goast.DeclStmt {
	return &goast.DeclStmt{
		Decl: &goast.GenDecl{
			Tok: token.VAR,
			Specs: []goast.Spec{
				&goast.ValueSpec{
					Names: []*goast.Ident{ident},
					Type:  ContextType(),
					Values: []goast.Expr{
						&goast.CallExpr{
							Fun: &goast.SelectorExpr{
								X:   &goast.Ident{Name: "context"},
								Sel: &goast.Ident{Name: "Background"},
							},
						},
					},
				},
			},
		},
	}
}
