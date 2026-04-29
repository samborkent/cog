package ast

import (
	"fmt"
	"strconv"
	"strings"

	f16 "github.com/x448/float16"

	"github.com/samborkent/cog/internal/tokens"
	"github.com/samborkent/cog/internal/types"
)

type float16 = f16.Float16

var _ Expr = &Float16Literal{}

type Float16Literal struct {
	Token tokens.Token
	Value float16
}

func (a *AST) NewFloat16Literal(t tokens.Token) (ExprIndex, error) {
	value, err := strconv.ParseFloat(t.Literal, 32)
	if err != nil {
		return ZeroExprIndex, fmt.Errorf("unable to parse float literal to float16: %w", err)
	}

	expr := New[Float16Literal](a)
	expr.Token = t
	expr.Value = f16.Fromfloat32(float32(value))

	return a.AddExpr(expr), nil
}

func (l *Float16Literal) Pos() (uint32, uint16) {
	return l.Token.Ln, l.Token.Col
}

func (l *Float16Literal) Hash() uint64 {
	return hash(l)
}

func (l *Float16Literal) StringTo(out *strings.Builder, _ *AST) {
	_ = out.WriteByte('(')
	_, _ = out.WriteString(l.Value.String())
	_, _ = out.WriteString(" : float16)")
}

func (l *Float16Literal) String() string {
	var out strings.Builder
	l.StringTo(&out, nil)
	return out.String()
}

func (l *Float16Literal) Type() types.Type {
	return types.Basics[types.Float16]
}
