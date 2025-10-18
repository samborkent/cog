package ast

import (
	"fmt"
	"strconv"

	"github.com/samborkent/cog/internal/tokens"
	"github.com/samborkent/cog/internal/types"
)

var _ Expression = &Uint8Literal{}

type Uint8Literal struct {
	expression

	Token tokens.Token
	Value uint8
}

func NewUint8Literal(t tokens.Token) (*Uint8Literal, error) {
	value, err := strconv.ParseUint(t.Literal, 10, 8)
	if err != nil {
		return nil, fmt.Errorf("unable to parse int literal to int8: %w", err)
	}

	return &Uint8Literal{
		Token: t,
		Value: uint8(value),
	}, nil
}

func (l *Uint8Literal) Pos() (uint32, uint16) {
	return l.Token.Ln, l.Token.Col
}

func (l *Uint8Literal) Hash() uint64 {
	return hash(l)
}

func (l *Uint8Literal) String() string {
	return "(" + strconv.FormatUint(uint64(l.Value), 10) + " : uint8)"
}

func (l *Uint8Literal) Type() types.Type {
	return types.Basics[types.Uint8]
}
