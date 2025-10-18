package ast

import (
	"github.com/samborkent/cog/internal/tokens"
	"github.com/samborkent/cog/internal/types"
)

var _ Expression = &UnionLiteral{}

type UnionLiteral struct {
	expression

	Token     tokens.Token
	UnionType types.Type
	Value     Expression
	Tag       bool // False: Either, True: Or
}

func (e *UnionLiteral) Pos() (uint32, uint16) {
	return e.Token.Ln, e.Token.Col
}

func (e *UnionLiteral) Hash() uint64 {
	return hash(e)
}

func (e *UnionLiteral) String() string {
	return e.Value.String()
}

func (e *UnionLiteral) Type() types.Type {
	if e.UnionType == nil {
		panic("union with nil type detected")
	}

	return e.UnionType
}
