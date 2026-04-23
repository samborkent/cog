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

func NewBoolLiteral(t tokens.Token) (*BoolLiteral, error) {
	var val bool

	switch t.Type {
	case tokens.False:
	case tokens.True:
		val = true
	default:
		return nil, fmt.Errorf("invalid bool token %q", t)
	}

	return &BoolLiteral{
		Token: t,
		Value: val,
	}, nil
}

func (b *BoolLiteral) Kind() NodeKind {
	return KindBoolLiteral
}

func (l *BoolLiteral) Pos() (uint32, uint16) {
	return l.Token.Ln, l.Token.Col
}

func (l *BoolLiteral) Hash() uint64 {
	return hash(l)
}

func (l *BoolLiteral) stringTo(out *strings.Builder) {
	_, _ = out.WriteString(l.Token.Type.String())
}

func (l *BoolLiteral) String() string {
	return l.Token.Type.String()
}

func (l *BoolLiteral) Type() types.Type {
	return types.Basics[types.Bool]
}
