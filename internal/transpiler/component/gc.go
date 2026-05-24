package component

import (
	goast "go/ast"
)

var (
	maxprocsSet = &goast.CallExpr{
		Fun: &goast.SelectorExpr{
			X:   &goast.Ident{Name: "_maxprocs"},
			Sel: &goast.Ident{Name: "Set"},
		},
	}
	memlimitSet = &goast.CallExpr{
		Fun: &goast.SelectorExpr{
			X:   &goast.Ident{Name: "_memlimit"},
			Sel: &goast.Ident{Name: "SetGoMemLimitWithOpts"},
		},
		Args: []goast.Expr{
			&goast.CallExpr{
				Fun: &goast.SelectorExpr{
					X:   &goast.Ident{Name: "_memlimit"},
					Sel: &goast.Ident{Name: "WithProvider"},
				},
				Args: []goast.Expr{
					&goast.CallExpr{
						Fun: &goast.SelectorExpr{
							X:   &goast.Ident{Name: "_memlimit"},
							Sel: &goast.Ident{Name: "ApplyFallback"},
						},
						Args: []goast.Expr{
							&goast.SelectorExpr{
								X:   &goast.Ident{Name: "_memlimit"},
								Sel: &goast.Ident{Name: "FromCgroup"},
							},
							&goast.SelectorExpr{
								X:   &goast.Ident{Name: "_memlimit"},
								Sel: &goast.Ident{Name: "FromSystem"},
							},
						},
					},
				},
			},
		},
	}
	autoAdapt = &goast.SelectorExpr{
		X:   &goast.Ident{Name: "_adaptivegc"},
		Sel: &goast.Ident{Name: "AutoAdapt"},
	}
)

func SetMaxProcs() *goast.ExprStmt {
	return &goast.ExprStmt{
		X: maxprocsSet,
	}
}

func SetMemLimit() *goast.ExprStmt {
	return &goast.ExprStmt{
		X: memlimitSet,
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
