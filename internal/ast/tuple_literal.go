package ast

import (
	"strings"

	"github.com/samborkent/cog/internal/tokens"
	"github.com/samborkent/cog/internal/types"
)

var _ Expr = &TupleLiteral{}

type TupleLiteral struct {
	Token     tokens.Token
	TupleType types.Type
	Values    []ExprIndex
}

func (a *AST) NewTupleLiteral(token tokens.Token, t *types.Tuple, values []ExprIndex) ExprIndex {
	tupleLiteral := New[TupleLiteral](a)
	tupleLiteral.Token = token
	tupleLiteral.TupleType = t
	tupleLiteral.Values = values
	return a.AddExpr(tupleLiteral)
}

func (l *TupleLiteral) Pos() (uint32, uint16) {
	return l.Token.Ln, l.Token.Col
}

func (l *TupleLiteral) Hash() uint64 {
	return hash(l)
}

func (l *TupleLiteral) StringTo(out *strings.Builder, a *AST) {
	_ = out.WriteByte('{')

	for i, val := range l.Values {
		a.exprs[val].StringTo(out, a)

		if i < len(l.Values)-1 {
			_, _ = out.WriteString(", ")
		}
	}

	_ = out.WriteByte('}')
}

func (l *TupleLiteral) Type() types.Type {
	return l.TupleType
}
