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
	Values    []ExprValue
}

func (e *TupleLiteral) Pos() (uint32, uint16) {
	return e.Token.Ln, e.Token.Col
}

func (e *TupleLiteral) Hash() uint64 {
	return hash(e)
}

func (e *TupleLiteral) stringTo(out *strings.Builder) {
	_ = out.WriteByte('{')

	for i, val := range e.Values {
		val.expr.stringTo(out)

		if i < len(e.Values)-1 {
			_, _ = out.WriteString(", ")
		}
	}

	_ = out.WriteByte('}')
}

func (e *TupleLiteral) String() string {
	var out strings.Builder
	e.stringTo(&out)

	return out.String()
}

func (e *TupleLiteral) Type() types.Type {
	if e.TupleType == nil {
		panic("tuple with nil type detected")
	}

	return e.TupleType
}
