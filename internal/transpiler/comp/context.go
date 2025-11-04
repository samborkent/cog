package comp

import (
	goast "go/ast"
	"go/token"
)

var (
	ContextArg = &goast.Field{
		Names: []*goast.Ident{ContextVar},
		Type:  ContextType,
	}
	ContextPackage = &goast.Ident{Name: "context"}
	ContextType    = &goast.SelectorExpr{
		X:   ContextPackage,
		Sel: &goast.Ident{Name: "Context"},
	}
	ContextVar = &goast.Ident{Name: "ctx"}
)

func ContextMain(ident *goast.Ident) *goast.DeclStmt {
	return &goast.DeclStmt{
		Decl: &goast.GenDecl{
			Tok: token.VAR,
			Specs: []goast.Spec{
				&goast.ValueSpec{
					Names: []*goast.Ident{ident},
					Type:  ContextType,
					Values: []goast.Expr{
						&goast.CallExpr{
							Fun: &goast.SelectorExpr{
								X:   ContextPackage,
								Sel: &goast.Ident{Name: "Background"},
							},
						},
					},
				},
			},
		},
	}
}
