package ast

import (
	"strings"

	"github.com/samborkent/cog/internal/tokens"
	"github.com/samborkent/cog/internal/types"
)

var _ Expression = &Builtin{}

type Builtin struct {
	expression

	Token         tokens.Token
	Name          string
	TypeArguments []types.Type
	Arguments     []Expression
	ReturnType    types.Type
}

func (b *Builtin) Pos() (uint32, uint16) {
	return b.Token.Ln, b.Token.Col
}

func (b *Builtin) Hash() uint64 {
	return hash(b)
}

func (b *Builtin) String() string {
	var out strings.Builder

	_ = out.WriteByte('@')
	_, _ = out.WriteString(b.Name)

	for i, arg := range b.TypeArguments {
		if i == 0 {
			_ = out.WriteByte('<')
		}

		_, _ = out.WriteString(arg.String())

		if i < len(b.TypeArguments)-1 {
			_, _ = out.WriteString(", ")
		} else {
			_ = out.WriteByte('>')
		}
	}

	_ = out.WriteByte('(')

	for i, arg := range b.Arguments {
		_, _ = out.WriteString(arg.String())

		if i < len(b.Arguments)-1 {
			_, _ = out.WriteString(", ")
		}
	}

	_ = out.WriteByte(')')

	return out.String()
}

func (b *Builtin) Type() types.Type {
	if b.ReturnType == nil {
		panic("builtin with nil-type detected")
	}

	return b.ReturnType
}
