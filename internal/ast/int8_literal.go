package ast

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/samborkent/cog/internal/tokens"
	"github.com/samborkent/cog/internal/types"
)

var _ Expr = &Int8Literal{}

type Int8Literal struct {
	Token tokens.Token
	Value int8
}

func (a *AST) NewInt8Literal(t tokens.Token) (ExprIndex, error) {
	value, err := strconv.ParseInt(t.Literal, 10, 8)
	if err != nil {
		return ZeroExprIndex, fmt.Errorf("unable to parse int literal to int8: %w", err)
	}

	expr := New[Int8Literal](a)
	expr.Token = t
	expr.Value = int8(value)

	return a.AddExpr(expr), nil
}

func (l *Int8Literal) Pos() (uint32, uint16) {
	return l.Token.Ln, l.Token.Col
}

func (l *Int8Literal) Hash() uint64 {
	return hash(l)
}

func (l *Int8Literal) StringTo(out *strings.Builder, _ *AST) {
	_ = out.WriteByte('(')
	_, _ = out.WriteString(strconv.FormatInt(int64(l.Value), 10))
	_, _ = out.WriteString(" : int8)")
}

func (l *Int8Literal) String() string {
	var out strings.Builder
	l.StringTo(&out, nil)
	return out.String()
}

func (l *Int8Literal) Type() types.Type {
	return types.Basics[types.Int8]
}
