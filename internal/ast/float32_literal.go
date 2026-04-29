package ast

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/samborkent/cog/internal/tokens"
	"github.com/samborkent/cog/internal/types"
)

var _ Expr = &Float32Literal{}

type Float32Literal struct {
	Token tokens.Token
	Value float32
}

func (a *AST) NewFloat32Literal(t tokens.Token) (ExprIndex, error) {
	value, err := strconv.ParseFloat(t.Literal, 32)
	if err != nil {
		return ZeroExprIndex, fmt.Errorf("unable to parse float literal to float32: %w", err)
	}

	expr := New[Float32Literal](a)
	expr.Token = t
	expr.Value = float32(value)

	return a.AddExpr(expr), nil
}

func (l *Float32Literal) Pos() (uint32, uint16) {
	return l.Token.Ln, l.Token.Col
}

func (l *Float32Literal) Hash() uint64 {
	return hash(l)
}

func (l *Float32Literal) StringTo(out *strings.Builder, _ *AST) {
	_ = out.WriteByte('(')
	_, _ = out.WriteString(strconv.FormatFloat(float64(l.Value), 'g', -1, 32))
	_, _ = out.WriteString(" : float32)")
}

func (l *Float32Literal) String() string {
	var out strings.Builder
	l.StringTo(&out, nil)
	return out.String()
}

func (l *Float32Literal) Type() types.Type {
	return types.Basics[types.Float32]
}
