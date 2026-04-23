package ast

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/samborkent/cog/internal/tokens"
	"github.com/samborkent/cog/internal/types"
)

var _ Expr = &Float64Literal{}

type Float64Literal struct {
	Token tokens.Token
	Value float64
}

func NewFloat64Literal(t tokens.Token) (*Float64Literal, error) {
	value, err := strconv.ParseFloat(t.Literal, 64)
	if err != nil {
		return nil, fmt.Errorf("unable to parse float literal to float64: %w", err)
	}

	return &Float64Literal{
		Token: t,
		Value: value,
	}, nil
}

func (l *Float64Literal) Kind() NodeKind {
	return KindFloat64Literal
}

func (l *Float64Literal) Pos() (uint32, uint16) {
	return l.Token.Ln, l.Token.Col
}

func (l *Float64Literal) Hash() uint64 {
	return hash(l)
}

func (l *Float64Literal) stringTo(out *strings.Builder) {
	_ = out.WriteByte('(')
	_, _ = out.WriteString(strconv.FormatFloat(l.Value, 'g', -1, 64))
	_, _ = out.WriteString(" : float64)")
}

func (l *Float64Literal) String() string {
	var out strings.Builder
	l.stringTo(&out)

	return out.String()
}

func (l *Float64Literal) Type() types.Type {
	return types.Basics[types.Float64]
}
