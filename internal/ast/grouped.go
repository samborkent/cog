package ast

import (
	"strings"

	"github.com/samborkent/cog/internal/tokens"
	"github.com/samborkent/cog/internal/types"
)

var _ Expr = &Grouped{}

// Grouped represents a parenthesized/grouped expression like (expr).
// This preserves parentheses in the AST so the transpiler can output them correctly.
type Grouped struct {
	Token    tokens.Token
	ExprType types.Type
	Expr     ExprIndex
}

func (a *AST) NewGrouped(token tokens.Token, expr ExprIndex, exprType types.Type) ExprIndex {
	grouped := New[Grouped](a)
	grouped.Token = token
	grouped.Expr = expr
	grouped.ExprType = exprType
	return a.AddExpr(grouped)
}

func (e *Grouped) Pos() (uint32, uint16) {
	return e.Token.Ln, e.Token.Col
}

func (e *Grouped) Hash() uint64 {
	return hash(e)
}

func (e *Grouped) StringTo(out *strings.Builder, a *AST) {
	_ = out.WriteByte('(')
	a.exprs[e.Expr].StringTo(out, a)
	_ = out.WriteByte(')')
}

// Type returns the type of the inner expression.
func (e *Grouped) Type() types.Type {
	return e.ExprType
}
