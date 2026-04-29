package ast

import (
	"fmt"
	"strings"

	"github.com/samborkent/cog/internal/tokens"
	"github.com/samborkent/cog/internal/types"
)

type Complex32 = [2]float16

func Complex32To64(c Complex32) complex64 {
	return complex(c[0].Float32(), c[1].Float32())
}

var _ Expr = &Complex32Literal{}

type Complex32Literal struct {
	Token tokens.Token
	Value Complex32
}

func (a *AST) NewComplex32Literal(token tokens.Token, value Complex32) ExprIndex {
	complex32Literal := New[Complex32Literal](a)
	complex32Literal.Token = token
	complex32Literal.Value = value
	return a.AddExpr(complex32Literal)
}

func (l *Complex32Literal) Pos() (uint32, uint16) {
	return l.Token.Ln, l.Token.Col
}

func (l *Complex32Literal) Hash() uint64 {
	return hash(l)
}

func (l *Complex32Literal) StringTo(out *strings.Builder, _ *AST) {
	fmt.Fprintf(out, "(%g, %g : complex32)", float32(l.Value[0]), float32(l.Value[1]))
}

func (l *Complex32Literal) String() string {
	var out strings.Builder
	l.StringTo(&out, nil)
	return out.String()
}

func (l *Complex32Literal) Type() types.Type {
	return types.Basics[types.Complex32]
}
