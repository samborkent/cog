package ast

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/samborkent/cog/internal/tokens"
	"github.com/samborkent/cog/internal/types"
)

var _ Expression = &Int32Literal{}

type Int32Literal struct {
	expression

	Token tokens.Token
	Value int32
}

func NewInt32Literal(t tokens.Token) (*Int32Literal, error) {
	value, err := strconv.ParseInt(t.Literal, 10, 32)
	if err != nil {
		return nil, fmt.Errorf("unable to parse int literal to int32: %w", err)
	}

	return &Int32Literal{
		Token: t,
		Value: int32(value),
	}, nil
}

func (l *Int32Literal) Pos() (uint32, uint16) {
	return l.Token.Ln, l.Token.Col
}

func (l *Int32Literal) Hash() uint64 {
	return hash(l)
}

func (l *Int32Literal) stringTo(out *strings.Builder) {
	_ = out.WriteByte('(')
	_, _ = out.WriteString(strconv.FormatInt(int64(l.Value), 10))
	_, _ = out.WriteString(" : int32)")
}

func (l *Int32Literal) String() string {
	var out strings.Builder
	l.stringTo(&out)
	return out.String()
}

func (l *Int32Literal) Type() types.Type {
	return types.Basics[types.Int32]
}
