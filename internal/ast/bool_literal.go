package ast

import (
	"fmt"
	"strings"

	"github.com/samborkent/cog/internal/tokens"
	"github.com/samborkent/cog/internal/types"
)

var _ Expr = &BoolLiteral{}

type BoolLiteral struct {
	Token tokens.Token
	Value bool
}

func (a *AST) NewBoolLiteral(t tokens.Token) ExprIndex {
	var val bool

	switch t.Type {
	case tokens.False:
	case tokens.True:
		val = true
	default:
		panic(fmt.Sprintf("unexpected token type %q for bool literal", t.Type))
	}

	expr := New[BoolLiteral](a)
	expr.Token = t
	expr.Value = val

	return a.AddExpr(expr)
}

func (l *BoolLiteral) Pos() (uint32, uint16) {
	return l.Token.Ln, l.Token.Col
}

func (l *BoolLiteral) Hash() uint64 {
	return hash(l)
}

func (l *BoolLiteral) StringTo(out *strings.Builder, _ *AST) {
	_, _ = out.WriteString(l.Token.Type.String())
}

func (l *BoolLiteral) String() string {
	return l.Token.Type.String()
}

func (l *BoolLiteral) Type() types.Type {
	return types.Basics[types.Bool]
}
