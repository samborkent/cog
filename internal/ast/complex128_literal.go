package ast

import (
	"fmt"

	"github.com/samborkent/cog/internal/tokens"
	"github.com/samborkent/cog/internal/types"
)

var _ Expression = &Complex128Literal{}

type Complex128Literal struct {
	expression

	Token tokens.Token
	Value complex128
}

func (l *Complex128Literal) Pos() (uint32, uint16) {
	return l.Token.Ln, l.Token.Col
}

func (l *Complex128Literal) Hash() uint64 {
	return hash(l)
}

func (l *Complex128Literal) String() string {
	return fmt.Sprintf("(%g : complex128)", l.Value)
}

func (l *Complex128Literal) Type() types.Type {
	return types.Basics[types.Complex128]
}
