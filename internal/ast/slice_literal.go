package ast

import (
	"strings"

	"github.com/samborkent/cog/internal/tokens"
	"github.com/samborkent/cog/internal/types"
)

var _ Expression = &SliceLiteral{}

type SliceLiteral struct {
	expression

	Token       tokens.Token
	ElementType types.Type
	Values      []Expression
}

func (l *SliceLiteral) Pos() (uint32, uint16) {
	return l.Token.Ln, l.Token.Col
}

func (l *SliceLiteral) Hash() uint64 {
	return hash(l)
}

func (l *SliceLiteral) String() string {
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

func (l *SliceLiteral) Type() types.Type {
	if l.ElementType == nil {
		panic("slice with nil element type detected")
	}

	return &types.Slice{
		Element: l.ElementType,
	}
}
