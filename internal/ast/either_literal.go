package ast

import (
	"strings"

	"github.com/samborkent/cog/internal/tokens"
	"github.com/samborkent/cog/internal/types"
)

var _ Expression = &EitherLiteral{}

type EitherLiteral struct {
	expression

	Token      tokens.Token
	EitherType types.Type
	Value      Expression
	IsRight    bool
}

func (e *EitherLiteral) Pos() (uint32, uint16) {
	return e.Token.Ln, e.Token.Col
}

func (e *EitherLiteral) Hash() uint64 {
	return hash(e)
}

func (e *EitherLiteral) stringTo(out *strings.Builder) {
	e.Value.stringTo(out)
}

func (e *EitherLiteral) String() string {
	var out strings.Builder
	e.stringTo(&out)

	return out.String()
}

func (e *EitherLiteral) Type() types.Type {
	if e.EitherType == nil {
		panic("either with nil type detected")
	}

	return e.EitherType
}
