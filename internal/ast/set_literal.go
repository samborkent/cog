package ast

import (
	"strings"

	"github.com/samborkent/cog/internal/tokens"
	"github.com/samborkent/cog/internal/types"
)

var _ Expr = &SetLiteral{}

type SetLiteral struct {
	Token   tokens.Token
	SetType types.Type
	Values  []ExprValue
}

func (l *SetLiteral) Pos() (uint32, uint16) {
	return l.Token.Ln, l.Token.Col
}

func (l *SetLiteral) Hash() uint64 {
	return hash(l)
}

func (l *SetLiteral) stringTo(out *strings.Builder) {
	_, _ = out.WriteString("({")

	for i, v := range l.Values {
		v.expr.stringTo(out)

		if i < len(l.Values)-1 {
			_, _ = out.WriteString(", ")
		}
	}

	_, _ = out.WriteString("} : ")
	_, _ = out.WriteString(l.Type().String())
	_ = out.WriteByte(')')
}

func (l *SetLiteral) String() string {
	var out strings.Builder
	l.stringTo(&out)

	return out.String()
}

func (l *SetLiteral) Type() types.Type {
	if l.SetType == nil {
		panic("set with nil set type detected")
	}

	if l.SetType.Kind() != types.SetKind {
		panic("set with non-set type detected")
	}

	return l.SetType
}
