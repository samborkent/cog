package ast

import (
	"fmt"
	"strconv"

	"github.com/samborkent/cog/internal/tokens"
	"github.com/samborkent/cog/internal/types"
)

var _ Expression = &Int16Literal{}

type Int16Literal struct {
	expression

	Token tokens.Token
	Value int16
}

func NewInt16Literal(t tokens.Token) (*Int16Literal, error) {
	value, err := strconv.ParseInt(t.Literal, 10, 16)
	if err != nil {
		return nil, fmt.Errorf("unable to parse int literal to int16: %w", err)
	}

	return &Int16Literal{
		Token: t,
		Value: int16(value),
	}, nil
}

func (l *Int16Literal) Pos() (uint32, uint16) {
	return l.Token.Ln, l.Token.Col
}

func (l *Int16Literal) Hash() uint64 {
	return hash(l)
}

func (l *Int16Literal) String() string {
	return "(" + strconv.FormatInt(int64(l.Value), 10) + " : int16)"
}

func (l *Int16Literal) Type() types.Type {
	return types.Basics[types.Int16]
}
