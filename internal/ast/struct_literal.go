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
	Value ExprIndex
}

func (a *AST) NewStructLiteral(token tokens.Token, t *types.Struct, values []FieldValue) ExprIndex {
	structLiteral := New[StructLiteral](a)
	structLiteral.Token = token
	structLiteral.StructType = t
	structLiteral.Values = values
	return a.AddExpr(structLiteral)
}

func (l *StructLiteral) Pos() (uint32, uint16) {
	return l.Token.Ln, l.Token.Col
}

func (l *StructLiteral) Hash() uint64 {
	return hash(l)
}

func (l *StructLiteral) StringTo(out *strings.Builder, a *AST) {
	_, _ = out.WriteString("({")

	for i, val := range l.Values {
		if i == 0 {
			_ = out.WriteByte('\n')
		}

		_, _ = out.WriteString(val.Name)
		_, _ = out.WriteString(" = ")
		a.exprs[val.Value].StringTo(out, a)
		_, _ = out.WriteString(",\n")
	}

	_, _ = out.WriteString("} : ")
	_, _ = out.WriteString(l.Type().String())
	_ = out.WriteByte(')')
}

func (l *StructLiteral) String() string {
	var out strings.Builder
	l.StringTo(&out, nil)
	return out.String()
}

func (l *StructLiteral) Type() types.Type {
	return l.StructType
}
