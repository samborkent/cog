package ast

import (
	"strings"

	"github.com/samborkent/cog/internal/tokens"
	"github.com/samborkent/cog/internal/types"
)

var _ Expression = &UnionLiteral{}

type UnionLiteral struct {
	expression

	Token     tokens.Token
	UnionType types.Type
	Value     Expression
	IsRight   bool
}

func (e *UnionLiteral) Pos() (uint32, uint16) {
	return e.Token.Ln, e.Token.Col
}

func (e *UnionLiteral) Hash() uint64 {
	return hash(e)
}

func (e *UnionLiteral) stringTo(out *strings.Builder) {
	e.Value.stringTo(out)
}

func (e *UnionLiteral) String() string {
	var out strings.Builder
	e.stringTo(&out)

	return out.String()
}

func (e *UnionLiteral) Type() types.Type {
	if e.UnionType == nil {
		panic("union with nil type detected")
	}

	return e.UnionType
}
