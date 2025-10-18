package ast

import (
	"fmt"

	"github.com/samborkent/cog/internal/tokens"
	"github.com/samborkent/cog/internal/types"
)

type complex32 = [2]float16

func complex32to64(c complex32) complex64 {
	return complex(c[0].Float32(), c[1].Float32())
}

var _ Expression = &Complex32Literal{}

type Complex32Literal struct {
	expression

	Token tokens.Token
	Value complex32
}

func (l *Complex32Literal) Pos() (uint32, uint16) {
	return l.Token.Ln, l.Token.Col
}

func (l *Complex32Literal) Hash() uint64 {
	return hash(l)
}

func (l *Complex32Literal) String() string {
	return fmt.Sprintf("(%g, %g : complex32)", float32(l.Value[0]), float32(l.Value[1]))
}

func (l *Complex32Literal) Type() types.Type {
	return types.Basics[types.Complex32]
}
