package ast

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/samborkent/cog/internal/tokens"
	"github.com/samborkent/cog/internal/types"
)

var _ Expr = &Uint8Literal{}

type Uint8Literal struct {
	Token tokens.Token
	Value uint8
}

func (a *AST) NewUint8Literal(t tokens.Token) (ExprIndex, error) {
	value, err := strconv.ParseUint(t.Literal, 10, 8)
	if err != nil {
		return ZeroExprIndex, fmt.Errorf("unable to parse int literal to uint8: %w", err)
	}

	expr := New[Uint8Literal](a)
	expr.Token = t
	expr.Value = uint8(value)

	return a.AddExpr(expr), nil
}

func (l *Uint8Literal) Pos() (uint32, uint16) {
	return l.Token.Ln, l.Token.Col
}

func (l *Uint8Literal) Hash() uint64 {
	return hash(l)
}

func (l *Uint8Literal) StringTo(out *strings.Builder, _ *AST) {
	_ = out.WriteByte('(')
	_, _ = out.WriteString(strconv.FormatUint(uint64(l.Value), 10))
	_, _ = out.WriteString(" : uint8)")
}

func (l *Uint8Literal) String() string {
	var out strings.Builder
	l.StringTo(&out, nil)
	return out.String()
}

func (l *Uint8Literal) Type() types.Type {
	return types.Basics[types.Uint8]
}
