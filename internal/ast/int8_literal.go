package ast

import (
	"fmt"
	"strconv"

	"github.com/samborkent/cog/internal/tokens"
	"github.com/samborkent/cog/internal/types"
)

var _ Expression = &Int8Literal{}

type Int8Literal struct {
	expression

	Token tokens.Token
	Value int8
}

func NewInt8Literal(t tokens.Token) (*Int8Literal, error) {
	value, err := strconv.ParseInt(t.Literal, 10, 8)
	if err != nil {
		return nil, fmt.Errorf("unable to parse int literal to int8: %w", err)
	}

	return &Int8Literal{
		Token: t,
		Value: int8(value),
	}, nil
}

func (l *Int8Literal) Pos() (uint32, uint16) {
	return l.Token.Ln, l.Token.Col
}

func (l *Int8Literal) Hash() uint64 {
	return hash(l)
}

func (l *Int8Literal) String() string {
	return "(" + strconv.FormatInt(int64(l.Value), 10) + " : int8)"
}

func (l *Int8Literal) Type() types.Type {
	return types.Basics[types.Int8]
}
