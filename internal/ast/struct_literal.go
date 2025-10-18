package ast

import (
	"strings"

	"github.com/samborkent/cog/internal/tokens"
	"github.com/samborkent/cog/internal/types"
)

var _ Expression = &StructLiteral{}

type StructLiteral struct {
	expression

	Token      tokens.Token
	StructType types.Type
	Values     []*FieldValue
}

type FieldValue struct {
	Name  string
	Value Expression
}

func (e *StructLiteral) Pos() (uint32, uint16) {
	return e.Token.Ln, e.Token.Col
}

func (e *StructLiteral) Hash() uint64 {
	return hash(e)
}

func (e *StructLiteral) String() string {
	var out strings.Builder

	_, _ = out.WriteString("({")

	for i, val := range e.Values {
		if i == 0 {
			_ = out.WriteByte('\n')
		}

		_, _ = out.WriteString(val.Name)
		_, _ = out.WriteString(" = ")
		_, _ = out.WriteString(val.Value.String())
		_, _ = out.WriteString(",\n")
	}

	_, _ = out.WriteString("} : ")
	_, _ = out.WriteString(e.Type().String())
	_ = out.WriteByte(')')

	return out.String()
}

func (e *StructLiteral) Type() types.Type {
	if e.StructType == nil {
		panic("struct with nil type detected")
	}

	return e.StructType
}
