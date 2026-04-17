package component

import (
	goast "go/ast"
	gotoken "go/token"
)

const (
	contextVar        = "ctx"
	contextPkg        = "go_context"
	contextType       = "Context"
	contextBackground = "Background"
	contextWithValue  = "WithValue"

	dynVar        = "dyn"
	dynKeyType    = "cogDynKey"
	dynStructType = "cogDyn"

	signalPkg            = "go_signal"
	syscallPkg           = "go_syscall"
	syscallNotifyContext = "NotifyContext"
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
	ContextVar        = &goast.Ident{Name: contextVar}
	ContextBackground = &goast.CallExpr{
		Fun: &goast.SelectorExpr{
			X:   ContextPackage,
			Sel: &goast.Ident{Name: contextBackground},
		},
	}

	DynVar        = &goast.Ident{Name: dynVar}
	DynKeyType    = &goast.Ident{Name: dynKeyType}
	DynStructType = &goast.Ident{Name: dynStructType}

	SignalPkg           = &goast.Ident{Name: signalPkg}
	SignalNotifyContext = &goast.SelectorExpr{
		X:   SignalPkg,
		Sel: &goast.Ident{Name: syscallNotifyContext},
	}
	SyscallPkg    = &goast.Ident{Name: syscallPkg}
	SyscallSigInt = &goast.SelectorExpr{
		X:   SyscallPkg,
		Sel: &goast.Ident{Name: "SIGINT"},
	}
	SyscallSigTerm = &goast.SelectorExpr{
		X:   SyscallPkg,
		Sel: &goast.Ident{Name: "SIGTERM"},
	}

	StopIdent = &goast.Ident{Name: "_stop"}
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

// DynMainInit generates the main function preamble for dynamic variables:
//
//	dyn := <structLit>
//	ctx := context.WithValue(context.Background(), cogDynKey{}, &dyn)
func DynMainInit(dynIdent, ctxIdent *goast.Ident, structLit goast.Expr) []goast.Stmt {
	return []goast.Stmt{
		&goast.AssignStmt{
			Tok: gotoken.DEFINE,
			Lhs: []goast.Expr{dynIdent},
			Rhs: []goast.Expr{structLit},
		},
		&goast.AssignStmt{
			Tok: gotoken.DEFINE,
			Lhs: []goast.Expr{ctxIdent},
			Rhs: []goast.Expr{
				&goast.CallExpr{
					Fun: &goast.SelectorExpr{
						X:   ContextPackage,
						Sel: &goast.Ident{Name: contextWithValue},
					},
					Args: []goast.Expr{
						&goast.CallExpr{
							Fun: &goast.SelectorExpr{
								X:   ContextPackage,
								Sel: &goast.Ident{Name: contextBackground},
							},
						},
						&goast.CompositeLit{Type: &goast.Ident{Name: dynKeyType}},
						&goast.UnaryExpr{
							Op: gotoken.AND,
							X:  &goast.Ident{Name: dynVar},
						},
					},
				},
			},
		},
	}
}

// DynProcEntry generates the proc entry preamble for dynamic variable isolation:
//
//	dyn := *ctx.Value(cogDynKey{}).(*cogDyn)
//	ctx = context.WithValue(ctx, cogDynKey{}, &dyn)
func DynProcEntry() []goast.Stmt {
	return []goast.Stmt{
		&goast.AssignStmt{
			Tok: gotoken.DEFINE,
			Lhs: []goast.Expr{&goast.Ident{Name: dynVar}},
			Rhs: []goast.Expr{
				&goast.StarExpr{
					X: &goast.TypeAssertExpr{
						X: &goast.CallExpr{
							Fun: &goast.SelectorExpr{
								X:   &goast.Ident{Name: contextVar},
								Sel: &goast.Ident{Name: "Value"},
							},
							Args: []goast.Expr{
								&goast.CompositeLit{Type: &goast.Ident{Name: dynKeyType}},
							},
						},
						Type: &goast.StarExpr{X: &goast.Ident{Name: dynStructType}},
					},
				},
			},
		},
		&goast.AssignStmt{
			Tok: gotoken.ASSIGN,
			Lhs: []goast.Expr{&goast.Ident{Name: contextVar}},
			Rhs: []goast.Expr{
				&goast.CallExpr{
					Fun: &goast.SelectorExpr{
						X:   &goast.Ident{Name: contextPkg},
						Sel: &goast.Ident{Name: contextWithValue},
					},
					Args: []goast.Expr{
						&goast.Ident{Name: contextVar},
						&goast.CompositeLit{Type: &goast.Ident{Name: dynKeyType}},
						&goast.UnaryExpr{
							Op: gotoken.AND,
							X:  &goast.Ident{Name: dynVar},
						},
					},
				},
			},
		},
	}
}

// DynRead generates a dynamic variable read expression: dyn.<fieldName>
func DynRead(fieldName string) goast.Expr {
	return &goast.SelectorExpr{
		X:   &goast.Ident{Name: dynVar},
		Sel: &goast.Ident{Name: fieldName},
	}
}

// DynWrite generates a dynamic variable write statement: dyn.<fieldName> = val
func DynWrite(fieldName string, val goast.Expr) *goast.AssignStmt {
	return &goast.AssignStmt{
		Tok: gotoken.ASSIGN,
		Lhs: []goast.Expr{
			&goast.SelectorExpr{
				X:   &goast.Ident{Name: dynVar},
				Sel: &goast.Ident{Name: fieldName},
			},
		},
		Rhs: []goast.Expr{val},
	}
}

func Signal(ctxIdent *goast.Ident, passCtx bool) []goast.Stmt {
	var ctxArg goast.Expr = ContextBackground
	if passCtx {
		ctxArg = ctxIdent
	}

	return []goast.Stmt{
		&goast.AssignStmt{
			Lhs: []goast.Expr{
				ctxIdent,
				StopIdent,
			},
			Tok: gotoken.DEFINE,
			Rhs: []goast.Expr{
				&goast.CallExpr{
					Fun: SignalNotifyContext,
					Args: []goast.Expr{
						ctxArg,
						SyscallSigInt,
						SyscallSigTerm,
					},
				},
			},
		},
		&goast.DeferStmt{
			Call: &goast.CallExpr{
				Fun: StopIdent,
			},
		},
	}
}
