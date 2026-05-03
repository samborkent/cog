package ast

import (
	"strings"

	"github.com/samborkent/cog/internal/tokens"
	"github.com/samborkent/cog/internal/types"
)

var _ Expr = &ArrayLiteral{}

type ArrayLiteral struct {
	Token     tokens.Token
	ArrayType *types.Array
	Values    []ExprIndex
}

func (a *AST) NewArrayLiteral(token tokens.Token, t *types.Array, values []ExprIndex) ExprIndex {
	arrayLiteral := New[ArrayLiteral](a)
	arrayLiteral.Token = token
	arrayLiteral.ArrayType = t
	arrayLiteral.Values = values
	return a.AddExpr(arrayLiteral)
}

func (l *ArrayLiteral) Pos() (uint32, uint16) {
	return l.Token.Ln, l.Token.Col
}

func (l *ArrayLiteral) Hash() uint64 {
	return hash(l)
}

func (l *ArrayLiteral) StringTo(out *strings.Builder, a *AST) {
	_, _ = out.WriteString("({")

	for i, v := range l.Values {
		a.exprs[v].StringTo(out, a)

		if i < len(l.Values)-1 {
			_, _ = out.WriteString(", ")
		}
	}

	_, _ = out.WriteString("} : ")
	_, _ = out.WriteString(l.Type().String())
	_ = out.WriteByte(')')
}

func (l *ArrayLiteral) Type() types.Type {
	return l.ArrayType
}
