package component

import (
	goast "go/ast"
	gotoken "go/token"
)

var arenaIdent = &goast.Ident{Name: "_arena"}

// ArenaInit generates: _arena := cog.NewArena()
func ArenaInit() goast.Stmt {
	return &goast.AssignStmt{
		Lhs: []goast.Expr{arenaIdent},
		Tok: gotoken.DEFINE,
		Rhs: []goast.Expr{
			&goast.CallExpr{
				Fun: &goast.SelectorExpr{
					X:   cogPkg,
					Sel: &goast.Ident{Name: "NewArena"},
				},
			},
		},
	}
}

// ArenaDeferFree generates: defer _arena.Free()
func ArenaDeferFree() goast.Stmt {
	return &goast.DeferStmt{
		Call: &goast.CallExpr{
			Fun: &goast.SelectorExpr{
				X:   arenaIdent,
				Sel: &goast.Ident{Name: "Free"},
			},
		},
	}
}

// RewriteSliceToArena rewrites a make([]T, len, cap) call in place to
// cog.MakeSlice[T](_arena, int(len), int(cap)).
// If capacity is absent, length is used as both len and cap.
func RewriteSliceToArena(call *goast.CallExpr) {
	arrayType := call.Args[0].(*goast.ArrayType)
	elemType := arrayType.Elt
	length := call.Args[1]

	var capacity goast.Expr
	if len(call.Args) >= 3 {
		capacity = call.Args[2]
	} else {
		capacity = length
	}

	call.Fun = &goast.IndexExpr{
		X: &goast.SelectorExpr{
			X:   cogPkg,
			Sel: &goast.Ident{Name: "MakeSlice"},
		},
		Index: elemType,
	}
	call.Args = []goast.Expr{
		arenaIdent,
		intCast(length),
		intCast(capacity),
	}
}

// intCast wraps an expression with int() for arena API compatibility.
// Integer literals are left unwrapped since they are untyped in Go.
func intCast(expr goast.Expr) goast.Expr {
	if lit, ok := expr.(*goast.BasicLit); ok && lit.Kind == gotoken.INT {
		return expr
	}

	return &goast.CallExpr{
		Fun:  &goast.Ident{Name: "int"},
		Args: []goast.Expr{expr},
	}
}
