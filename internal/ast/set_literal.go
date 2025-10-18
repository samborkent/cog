package ast

import (
	"strings"

	"github.com/samborkent/cog/internal/tokens"
	"github.com/samborkent/cog/internal/types"
)

var _ Expression = &SetLiteral{}

type SetLiteral struct {
	expression

	Token     tokens.Token
	ValueType types.Type
	Values    []Expression
}

func (l *SetLiteral) Pos() (uint32, uint16) {
	return l.Token.Ln, l.Token.Col
}

func (l *SetLiteral) Hash() uint64 {
	return hash(l)
}

func (l *SetLiteral) String() string {
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

func (l *SetLiteral) Type() types.Type {
	if l.ValueType == nil {
		panic("set with nil value type detected")
	}

	return &types.Set{
		Element: l.ValueType,
	}
}
