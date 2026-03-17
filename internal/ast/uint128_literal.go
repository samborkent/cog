package ast

import (
	"fmt"

	u128 "lukechampine.com/uint128"

	"github.com/samborkent/cog/internal/tokens"
	"github.com/samborkent/cog/internal/types"
)

type uint128 = u128.Uint128

var _ Expression = &Uint128Literal{}

type Uint128Literal struct {
	expression

	Token tokens.Token
	Value uint128
}

func NewUint128Literal(t tokens.Token) (*Uint128Literal, error) {
	value, err := u128.FromString(t.Literal)
	if err != nil {
		return nil, fmt.Errorf("unable to parse int literal to uint128: %w", err)
	}

	return &Uint128Literal{
		Token: t,
		Value: value,
	}, nil
}

func (l *Uint128Literal) Pos() (uint32, uint16) {
	return l.Token.Ln, l.Token.Col
}

func (l *Uint128Literal) Hash() uint64 {
	return hash(l)
}

func (l *Uint128Literal) String() string {
	return "(" + l.Value.String() + " : uint128)"
}

func (l *Uint128Literal) Type() types.Type {
	return types.Basics[types.Uint128]
}
