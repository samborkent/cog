package ast

import (
	"errors"
	"math/big"
	"strings"

	"github.com/ryanavella/wide"
	"github.com/samborkent/cog/internal/tokens"
	"github.com/samborkent/cog/internal/types"
)

type int128 = wide.Int128

var _ Expr = &Int128Literal{}

type Int128Literal struct {
	Token tokens.Token
	Value int128
}

func (a *AST) NewInt128Literal(t tokens.Token) (ExprIndex, error) {
	value := new(big.Int)

	_, ok := value.SetString(t.Literal, 10)
	if !ok {
		return ZeroExprIndex, errors.New("unable to parse int literal to int128")
	}

	expr := New[Int128Literal](a)
	expr.Token = t
	expr.Value = wide.Int128FromBigInt(value)

	return a.AddExpr(expr), nil
}

func (l *Int128Literal) Pos() (uint32, uint16) {
	return l.Token.Ln, l.Token.Col
}

func (l *Int128Literal) Hash() uint64 {
	return hash(l)
}

func (l *Int128Literal) StringTo(out *strings.Builder, _ *AST) {
	_ = out.WriteByte('(')
	_, _ = out.WriteString(l.Value.String())
	_, _ = out.WriteString(" : int128)")
}

func (l *Int128Literal) String() string {
	var out strings.Builder
	l.StringTo(&out, nil)
	return out.String()
}

func (l *Int128Literal) Type() types.Type {
	return types.Basics[types.Int128]
}
