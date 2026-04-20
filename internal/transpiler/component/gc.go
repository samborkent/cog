package component

import (
	goast "go/ast"
	"go/token"
)

var (
	int64Ident     = &goast.Ident{Name: "int64"}
	setMemoryLimit = &goast.SelectorExpr{
		X:   &goast.Ident{Name: "go_debug"},
		Sel: &goast.Ident{Name: "SetMemoryLimit"},
	}
	freeMemory = &goast.SelectorExpr{
		X:   &goast.Ident{Name: "_memory"},
		Sel: &goast.Ident{Name: "FreeMemory"},
	}
	autoAdapt = &goast.SelectorExpr{
		X:   &goast.Ident{Name: "_adaptivegc"},
		Sel: &goast.Ident{Name: "AutoAdapt"},
	}
)

var freeMemoryIdent = &goast.Ident{Name: "freeMemory"}

func SetMemoryLimit() *goast.FuncDecl {
	return &goast.FuncDecl{
		Name: &goast.Ident{Name: "init"},
		Type: &goast.FuncType{Params: &goast.FieldList{}},
		Body: &goast.BlockStmt{
			List: []goast.Stmt{
				&goast.AssignStmt{
					Lhs: []goast.Expr{freeMemoryIdent},
					Tok: token.DEFINE,
					Rhs: []goast.Expr{&goast.CallExpr{Fun: freeMemory}},
				},
				&goast.IfStmt{
					Cond: &goast.BinaryExpr{
						X:  freeMemoryIdent,
						Op: token.GTR,
						Y:  &goast.BasicLit{Kind: token.INT, Value: "0"},
					},
					Body: &goast.BlockStmt{
						List: []goast.Stmt{
							&goast.ExprStmt{
								X: &goast.CallExpr{
									Fun: setMemoryLimit,
									Args: []goast.Expr{
										&goast.CallExpr{
											Fun: int64Ident,
											Args: []goast.Expr{
												&goast.CallExpr{
													Fun: freeMemory,
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}
}

func AdaptiveGC(ctxIdent *goast.Ident) *goast.ExprStmt {
	return &goast.ExprStmt{
		X: &goast.CallExpr{
			Fun: autoAdapt,
			Args: []goast.Expr{
				ctxIdent,
			},
		},
	}
}
