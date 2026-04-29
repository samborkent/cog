package ast

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/samborkent/cog/internal/tokens"
	"github.com/samborkent/cog/internal/types"
)

var _ Expr = &Int32Literal{}

type Int32Literal struct {
	Token tokens.Token
	Value int32
}

func (a *AST) NewInt32Literal(t tokens.Token) (ExprIndex, error) {
	value, err := strconv.ParseInt(t.Literal, 10, 32)
	if err != nil {
		return ZeroExprIndex, fmt.Errorf("unable to parse int literal to int32: %w", err)
	}

	expr := New[Int32Literal](a)
	expr.Token = t
	expr.Value = int32(value)

	return a.AddExpr(expr), nil
}

func (l *Int32Literal) Pos() (uint32, uint16) {
	return l.Token.Ln, l.Token.Col
}

func (l *Int32Literal) Hash() uint64 {
	return hash(l)
}

func (l *Int32Literal) StringTo(out *strings.Builder, _ *AST) {
	_ = out.WriteByte('(')
	_, _ = out.WriteString(strconv.FormatInt(int64(l.Value), 10))
	_, _ = out.WriteString(" : int32)")
}

func (l *Int32Literal) String() string {
	var out strings.Builder
	l.StringTo(&out, nil)
	return out.String()
}

func (l *Int32Literal) Type() types.Type {
	return types.Basics[types.Int32]
}
