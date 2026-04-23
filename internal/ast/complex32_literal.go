package ast

import (
	"fmt"
	"strings"

	"github.com/samborkent/cog/internal/tokens"
	"github.com/samborkent/cog/internal/types"
)

type complex32 = [2]float16

func Complex32To64(c complex32) complex64 {
	return complex(c[0].Float32(), c[1].Float32())
}

var _ Expr = &Complex32Literal{}

type Complex32Literal struct {
	Token tokens.Token
	Value complex32
}

func (l *Complex32Literal) Kind() NodeKind {
	return KindComplex32Literal
}

func (l *Complex32Literal) Pos() (uint32, uint16) {
	return l.Token.Ln, l.Token.Col
}

func (l *Complex32Literal) Hash() uint64 {
	return hash(l)
}

func (l *Complex32Literal) stringTo(out *strings.Builder) {
	fmt.Fprintf(out, "(%g, %g : complex32)", float32(l.Value[0]), float32(l.Value[1]))
}

func (l *Complex32Literal) String() string {
	var out strings.Builder
	l.stringTo(&out)

	return out.String()
}

func (l *Complex32Literal) Type() types.Type {
	return types.Basics[types.Complex32]
}
