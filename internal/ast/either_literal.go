package ast

import (
	"strings"

	"github.com/samborkent/cog/internal/tokens"
	"github.com/samborkent/cog/internal/types"
)

var _ Expr = &EitherLiteral{}

type EitherLiteral struct {
	Token      tokens.Token
	EitherType types.Type
	Value      ExprIndex
	IsRight    bool
}

func (a *AST) NewEitherLiteral(token tokens.Token, eitherType types.Type, value ExprIndex, isRight bool) ExprIndex {
	expr := New[EitherLiteral](a)
	expr.Token = token
	expr.EitherType = eitherType
	expr.Value = value
	expr.IsRight = isRight
	return a.AddExpr(expr)
}

func (l *EitherLiteral) Pos() (uint32, uint16) {
	return l.Token.Ln, l.Token.Col
}

func (l *EitherLiteral) Hash() uint64 {
	return hash(l)
}

func (l *EitherLiteral) StringTo(out *strings.Builder, a *AST) {
	a.exprs[l.Value].StringTo(out, a)
}

func (l *EitherLiteral) String() string {
	var out strings.Builder
	l.StringTo(&out, nil)
	return out.String()
}

func (l *EitherLiteral) Type() types.Type {
	return l.EitherType
}
