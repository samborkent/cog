package ast

import (
	"strings"

	"github.com/samborkent/cog/internal/tokens"
	"github.com/samborkent/cog/internal/types"
)

var _ Expression = &EnumLiteral{}

type EnumLiteral struct {
	expression

	Token     tokens.Token
	ValueType types.Type
	Values    []*EnumValue
}

type EnumValue struct {
	Identifier *Identifier
	Value      Expression
}

func (e *EnumLiteral) Pos() (uint32, uint16) {
	return e.Token.Ln, e.Token.Col
}

func (e *EnumLiteral) Hash() uint64 {
	return hash(e)
}

func (e *EnumLiteral) String() string {
	var out strings.Builder

	_, _ = out.WriteString("({")

	for i, val := range e.Values {
		if i == 0 {
			_ = out.WriteByte('\n')
		}

		_, _ = out.WriteString(val.Identifier.Name)
		_, _ = out.WriteString(" := ")
		_, _ = out.WriteString(val.Value.String())
		_ = out.WriteByte('\n')
	}

	_, _ = out.WriteString("} : ")
	_, _ = out.WriteString(e.Type().String())
	_ = out.WriteByte(')')

	return out.String()
}

func (e *EnumLiteral) Type() types.Type {
	if e.ValueType == nil {
		panic("enum with nil value type detected")
	}

	return &types.Enum{
		Value: e.ValueType,
	}
}
