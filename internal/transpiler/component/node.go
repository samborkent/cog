package component

import (
	goast "go/ast"
	gotoken "go/token"

	"github.com/samborkent/cog/internal/ast"
)

var This = []*goast.Ident{{Name: "this"}}

func Receiver(ident *ast.Identifier) *goast.FieldList {
	return &goast.FieldList{
		List: []*goast.Field{{
			Names: This,
			Type:  Ident(ident),
		}},
	}
}

// Selector creates a *goast.SelectorExpr for x.sel.
func Selector(x goast.Expr, sel string) *goast.SelectorExpr {
	return &goast.SelectorExpr{
		X:   x,
		Sel: cachedIdent(sel),
	}
}

// AssignDef creates a short variable declaration lhs := rhs.
func AssignDef(lhs *goast.Ident, rhs goast.Expr) *goast.AssignStmt {
	return &goast.AssignStmt{
		Lhs: []goast.Expr{lhs},
		Tok: gotoken.DEFINE,
		Rhs: []goast.Expr{rhs},
	}
}

// Not creates a logical NOT expression !x.
func Not(x goast.Expr) *goast.UnaryExpr {
	return &goast.UnaryExpr{
		Op: gotoken.NOT,
		X:  x,
	}
}

// Call creates a function call with arguments.
func Call(fun goast.Expr, args ...goast.Expr) *goast.CallExpr {
	return &goast.CallExpr{
		Fun:  fun,
		Args: args,
	}
}

// TypeAssert creates a type assertion x.(type).
// Note: If t is nil, it emits a type switch assertion x.(type).
func TypeAssert(x goast.Expr, t goast.Expr) *goast.TypeAssertExpr {
	return &goast.TypeAssertExpr{
		X:    x,
		Type: t,
	}
}

// IfStmt creates an if statement.
func IfStmt(cond goast.Expr, body []goast.Stmt, elseStmt goast.Stmt) *goast.IfStmt {
	return &goast.IfStmt{
		Cond: cond,
		Body: &goast.BlockStmt{List: body},
		Else: elseStmt,
	}
}

// BlockStmt creates a block statement.
func BlockStmt(stmts ...goast.Stmt) *goast.BlockStmt {
	return &goast.BlockStmt{List: stmts}
}

// KeyValue creates a key: value expression suitable for composite literals.
func KeyValue(key, value goast.Expr) *goast.KeyValueExpr {
	return &goast.KeyValueExpr{
		Key:   key,
		Value: value,
	}
}
