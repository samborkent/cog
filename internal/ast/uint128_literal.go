package ast

import (
	"fmt"
	"strings"

	u128 "lukechampine.com/uint128"

	"github.com/samborkent/cog/internal/tokens"
	"github.com/samborkent/cog/internal/types"
)

type uint128 = u128.Uint128

var _ Expr = &Uint128Literal{}

type Uint128Literal struct {
	Token tokens.Token
	Value uint128
}

func (a *AST) NewUint128Literal(t tokens.Token) (ExprIndex, error) {
	value, err := u128.FromString(t.Literal)
	if err != nil {
		return ZeroExprIndex, fmt.Errorf("unable to parse int literal to uint128: %w", err)
	}

	expr := New[Uint128Literal](a)
	expr.Token = t
	expr.Value = value

	return a.AddExpr(expr), nil
}

func (l *Uint128Literal) Pos() (uint32, uint16) {
	return l.Token.Ln, l.Token.Col
}

func (l *Uint128Literal) Hash() uint64 {
	return hash(l)
}

func (l *Uint128Literal) StringTo(out *strings.Builder, _ *AST) {
	_ = out.WriteByte('(')
	_, _ = out.WriteString(l.Value.String())
	_, _ = out.WriteString(" : uint128)")
}

func (l *Uint128Literal) String() string {
	var out strings.Builder
	l.StringTo(&out, nil)
	return out.String()
}

func (l *Uint128Literal) Type() types.Type {
	return types.Basics[types.Uint128]
}
