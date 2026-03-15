package component

import (
	goast "go/ast"
	gotoken "go/token"
)

const (
	contextVar        = "ctx"
	contextPkg        = "context"
	contextType       = "Context"
	contextBackground = "Background"
	contextWithValue  = "WithValue"
)

var (
	ContextArg = &goast.Field{
		Names: []*goast.Ident{ContextVar},
		Type:  ContextType,
	}
	ContextPackage = &goast.Ident{Name: contextPkg}
	ContextType    = &goast.SelectorExpr{
		X:   ContextPackage,
		Sel: &goast.Ident{Name: contextType},
	}
	ContextVar = &goast.Ident{Name: contextVar}
)

func ContextMain(ident *goast.Ident) *goast.DeclStmt {
	return &goast.DeclStmt{
		Decl: &goast.GenDecl{
			Tok: gotoken.VAR,
			Specs: []goast.Spec{
				&goast.ValueSpec{
					Names: []*goast.Ident{ident},
					Type:  ContextType,
					Values: []goast.Expr{
						&goast.CallExpr{
							Fun: &goast.SelectorExpr{
								X:   ContextPackage,
								Sel: &goast.Ident{Name: contextBackground},
							},
						},
					},
				},
			},
		},
	}
}

func ContextWithValue(keyIdent *goast.Ident, val goast.Expr) *goast.AssignStmt {
	return &goast.AssignStmt{
		Tok: gotoken.ASSIGN,
		Lhs: []goast.Expr{ContextVar},
		Rhs: []goast.Expr{
			&goast.CallExpr{
				Fun: &goast.SelectorExpr{
					X:   ContextPackage,
					Sel: &goast.Ident{Name: contextWithValue},
				},
				Args: []goast.Expr{
					ContextVar,
					&goast.CompositeLit{Type: keyIdent},
					val,
				},
			},
		},
	}
}
