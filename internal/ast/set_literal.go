package ast

import (
	"strings"

	"github.com/samborkent/cog/internal/tokens"
	"github.com/samborkent/cog/internal/types"
)

var _ Expr = &SetLiteral{}

type SetLiteral struct {
	Token   tokens.Token
	SetType types.Type
	Values  []ExprIndex
}

func (a *AST) NewSetLiteral(token tokens.Token, t *types.Set, values []ExprIndex) ExprIndex {
	setLiteral := New[SetLiteral](a)
	setLiteral.Token = token
	setLiteral.SetType = t
	setLiteral.Values = values
	return a.AddExpr(setLiteral)
}

func (l *SetLiteral) Pos() (uint32, uint16) {
	return l.Token.Ln, l.Token.Col
}

func (l *SetLiteral) Hash() uint64 {
	return hash(l)
}

func (l *SetLiteral) StringTo(out *strings.Builder, a *AST) {
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

func (l *SetLiteral) Type() types.Type {
	return l.SetType
}
