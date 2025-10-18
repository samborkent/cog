package ast

import (
	"errors"
	"math/big"

	"github.com/samborkent/cog/internal/tokens"
	"github.com/samborkent/cog/internal/types"
)

// TODO: implement overflow and other fixed-size int behaviour
type int128 = *big.Int

var _ Expression = &Int128Literal{}

type Int128Literal struct {
	expression

	Token tokens.Token
	Value int128
}

func NewInt128Literal(t tokens.Token) (*Int128Literal, error) {
	value := new(big.Int)

	_, ok := value.SetString(t.Literal, 10)
	if !ok {
		return nil, errors.New("unable to parse int literal to int128")
	}

	return &Int128Literal{
		Token: t,
		Value: value,
	}, nil
}

func (l *Int128Literal) Pos() (uint32, uint16) {
	return l.Token.Ln, l.Token.Col
}

func (l *Int128Literal) Hash() uint64 {
	return hash(l)
}

func (l *Int128Literal) String() string {
	return "(" + l.Value.String() + " : int128)"
}

func (l *Int128Literal) Type() types.Type {
	return types.Basics[types.Int128]
}
