package ast

import (
	"fmt"
	"strconv"

	f16 "github.com/x448/float16"

	"github.com/samborkent/cog/internal/tokens"
	"github.com/samborkent/cog/internal/types"
)

type float16 = f16.Float16

var _ Expression = &Float16Literal{}

type Float16Literal struct {
	expression

	Token tokens.Token
	Value float16
}

func NewFloat16Literal(t tokens.Token) (*Float16Literal, error) {
	value, err := strconv.ParseFloat(t.Literal, 32)
	if err != nil {
		return nil, fmt.Errorf("unable to parse float literal to float16: %w", err)
	}

	return &Float16Literal{
		Token: t,
		Value: f16.Fromfloat32(float32(value)),
	}, nil
}

func (l *Float16Literal) Pos() (uint32, uint16) {
	return l.Token.Ln, l.Token.Col
}

func (l *Float16Literal) Hash() uint64 {
	return hash(l)
}

func (l *Float16Literal) String() string {
	return "(" + l.Value.String() + " : float16)"
}

func (l *Float16Literal) Type() types.Type {
	return types.Basics[types.Float16]
}
