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

func (a *AST) NewFloat64Literal(t tokens.Token) (ExprIndex, error) {
	value, err := strconv.ParseFloat(t.Literal, 64)
	if err != nil {
		return ZeroExprIndex, fmt.Errorf("unable to parse float literal to float64: %w", err)
	}

	expr := New[Float64Literal](a)
	expr.Token = t
	expr.Value = value

	return a.AddExpr(expr), nil
}

func (l *Float64Literal) Pos() (uint32, uint16) {
	return l.Token.Ln, l.Token.Col
}

func (l *Float64Literal) Hash() uint64 {
	return hash(l)
}

func (l *Float64Literal) StringTo(out *strings.Builder, _ *AST) {
	_ = out.WriteByte('(')
	_, _ = out.WriteString(strconv.FormatFloat(l.Value, 'g', -1, 64))
	_, _ = out.WriteString(" : float64)")
}

func (l *Float64Literal) String() string {
	var out strings.Builder
	l.StringTo(&out, nil)
	return out.String()
}

func (l *Float64Literal) Type() types.Type {
	return types.Basics[types.Float64]
}
