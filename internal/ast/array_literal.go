package ast

import (
	"strings"

	"github.com/samborkent/cog/internal/tokens"
	"github.com/samborkent/cog/internal/types"
)

var _ Expr = &ArrayLiteral{}

type ArrayLiteral struct {
	Token     tokens.Token
	ArrayType *types.Array
	Values    []ExprValue
}

func (l *ArrayLiteral) Pos() (uint32, uint16) {
	return l.Token.Ln, l.Token.Col
}

func (l *ArrayLiteral) Hash() uint64 {
	return hash(l)
}

func (l *ArrayLiteral) stringTo(out *strings.Builder) {
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

func (l *ArrayLiteral) String() string {
	var out strings.Builder
	l.stringTo(&out)

	return out.String()
}

func (l *ArrayLiteral) Type() types.Type {
	if l.ArrayType == nil {
		panic("array with nil type detected")
	}

	return l.ArrayType
}
