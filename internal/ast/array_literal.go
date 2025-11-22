package ast

import (
	"strings"

	"github.com/samborkent/cog/internal/tokens"
	"github.com/samborkent/cog/internal/types"
)

var _ Expression = &ArrayLiteral{}

type ArrayLiteral struct {
	expression

	Token     tokens.Token
	ArrayType *types.Array
	Values    []Expression
}

func (l *ArrayLiteral) Pos() (uint32, uint16) {
	return l.Token.Ln, l.Token.Col
}

func (l *ArrayLiteral) Hash() uint64 {
	return hash(l)
}

func (l *ArrayLiteral) String() string {
	var out strings.Builder

	_, _ = out.WriteString("({")

	for i, v := range l.Values {
		_, _ = out.WriteString(v.String())

		if i < len(l.Values)-1 {
			_, _ = out.WriteString(", ")
		}
	}

	_, _ = out.WriteString("} : ")
	_, _ = out.WriteString(l.Type().String())
	_ = out.WriteByte(')')

	return out.String()
}

func (l *ArrayLiteral) Type() types.Type {
	if l.ArrayType == nil {
		panic("array with nil type detected")
	}

	return l.ArrayType
}
