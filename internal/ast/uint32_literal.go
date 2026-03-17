package ast

import (
	"fmt"
	"strconv"

	"github.com/samborkent/cog/internal/tokens"
	"github.com/samborkent/cog/internal/types"
)

var _ Expression = &Uint32Literal{}

type Uint32Literal struct {
	expression

	Token tokens.Token
	Value uint32
}

func NewUint32Literal(t tokens.Token) (*Uint32Literal, error) {
	value, err := strconv.ParseUint(t.Literal, 10, 32)
	if err != nil {
		return nil, fmt.Errorf("unable to parse int literal to int32: %w", err)
	}

	return &Uint32Literal{
		Token: t,
		Value: uint32(value),
	}, nil
}

func (l *Uint32Literal) Pos() (uint32, uint16) {
	return l.Token.Ln, l.Token.Col
}

func (l *Uint32Literal) Hash() uint64 {
	return hash(l)
}

func (l *Uint32Literal) String() string {
	return "(" + strconv.FormatUint(uint64(l.Value), 10) + " : uint32)"
}

func (l *Uint32Literal) Type() types.Type {
	return types.Basics[types.Uint32]
}
