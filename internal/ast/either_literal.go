package ast

import (
	"strings"

	"github.com/samborkent/cog/internal/tokens"
	"github.com/samborkent/cog/internal/types"
)

var (
	_ Node = &EitherLiteral{}
	_ Expr = &EitherLiteral{}
)

type EitherLiteral struct {
	Token      tokens.Token
	EitherType types.Type
	Value      ExprValue
	IsRight    bool
}

func (e *EitherLiteral) Kind() NodeKind {
	return KindEitherLiteral
}

func (e *EitherLiteral) Pos() (uint32, uint16) {
	return e.Token.Ln, e.Token.Col
}

func (e *EitherLiteral) Hash() uint64 {
	return hash(e)
}

func (e *EitherLiteral) stringTo(out *strings.Builder) {
	e.Value.expr.stringTo(out)
}

func (e *EitherLiteral) String() string {
	var out strings.Builder
	e.stringTo(&out)

	return out.String()
}

func (e *EitherLiteral) Type() types.Type {
	return e.EitherType
}
