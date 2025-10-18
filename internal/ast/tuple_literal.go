package ast

import (
	"strings"

	"github.com/samborkent/cog/internal/tokens"
	"github.com/samborkent/cog/internal/types"
)

var _ Expression = &TupleLiteral{}

type TupleLiteral struct {
	expression

	Token     tokens.Token
	TupleType types.Type
	Values    []Expression
}

func (e *TupleLiteral) Pos() (uint32, uint16) {
	return e.Token.Ln, e.Token.Col
}

func (e *TupleLiteral) Hash() uint64 {
	return hash(e)
}

func (e *TupleLiteral) String() string {
	var out strings.Builder

	_, _ = out.WriteString("{")

	for i, val := range e.Values {
		_, _ = out.WriteString(val.String())

		if i < len(e.Values)-1 {
			_, _ = out.WriteString(", ")
		}
	}

	_, _ = out.WriteString("}")

	return out.String()
}

func (e *TupleLiteral) Type() types.Type {
	if e.TupleType == nil {
		panic("tuple with nil type detected")
	}

	return e.TupleType
}
