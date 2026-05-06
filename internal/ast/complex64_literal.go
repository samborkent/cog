package ast

import (
	"fmt"
	"strings"

	"github.com/samborkent/cog/internal/tokens"
	"github.com/samborkent/cog/internal/types"
)

var _ Expr = &Complex64Literal{}

type Complex64Literal struct {
	Token tokens.Token
	Value complex64
}

func (l *Complex64Literal) Pos() (uint32, uint16) {
	return l.Token.Ln, l.Token.Col
}

func (l *Complex64Literal) Hash() uint64 {
	return hash(l)
}

func (l *Complex64Literal) StringTo(out *strings.Builder, _ *AST) {
	fmt.Fprintf(out, "(%g : complex64)", l.Value)
}

func (l *Complex64Literal) String() string {
	var out strings.Builder
	l.StringTo(&out, nil)
	return out.String()
}

func (l *Complex64Literal) Type() types.Type {
	return types.Basics[types.Complex64]
}
