package ast

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/samborkent/cog/internal/tokens"
	"github.com/samborkent/cog/internal/types"
)

var _ Expression = &Uint16Literal{}

type Uint16Literal struct {
	expression

	Token tokens.Token
	Value uint16
}

func NewUint16Literal(t tokens.Token) (*Uint16Literal, error) {
	value, err := strconv.ParseUint(t.Literal, 10, 16)
	if err != nil {
		return nil, fmt.Errorf("unable to parse int literal to int16: %w", err)
	}

	return &Uint16Literal{
		Token: t,
		Value: uint16(value),
	}, nil
}

func (l *Uint16Literal) Pos() (uint32, uint16) {
	return l.Token.Ln, l.Token.Col
}

func (l *Uint16Literal) Hash() uint64 {
	return hash(l)
}

func (l *Uint16Literal) stringTo(out *strings.Builder) {
	_ = out.WriteByte('(')
	_, _ = out.WriteString(strconv.FormatUint(uint64(l.Value), 10))
	_, _ = out.WriteString(" : uint16)")
}

func (l *Uint16Literal) String() string {
	var out strings.Builder
	l.stringTo(&out)

	return out.String()
}

func (l *Uint16Literal) Type() types.Type {
	return types.Basics[types.Uint16]
}
