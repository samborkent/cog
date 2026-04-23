package ast

import (
	"strings"

	"github.com/samborkent/cog/internal/tokens"
	"github.com/samborkent/cog/internal/types"
)

var _ Expr = &StructLiteral{}

type StructLiteral struct {
	Token      tokens.Token
	StructType types.Type
	Values     []FieldValue
}

type FieldValue struct {
	Name  string
	Value ExprValue
}

func (e *StructLiteral) Kind() NodeKind {
	return KindStructLiteral
}

func (e *StructLiteral) Pos() (uint32, uint16) {
	return e.Token.Ln, e.Token.Col
}

func (e *StructLiteral) Hash() uint64 {
	return hash(e)
}

func (e *StructLiteral) stringTo(out *strings.Builder) {
	_, _ = out.WriteString("({")

	for i, val := range e.Values {
		if i == 0 {
			_ = out.WriteByte('\n')
		}

		_, _ = out.WriteString(val.Name)
		_, _ = out.WriteString(" = ")
		val.Value.expr.stringTo(out)
		_, _ = out.WriteString(",\n")
	}

	_, _ = out.WriteString("} : ")
	_, _ = out.WriteString(e.Type().String())
	_ = out.WriteByte(')')
}

func (e *StructLiteral) String() string {
	var out strings.Builder
	e.stringTo(&out)

	return out.String()
}

func (e *StructLiteral) Type() types.Type {
	return e.StructType
}
