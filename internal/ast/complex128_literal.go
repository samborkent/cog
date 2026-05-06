package ast

import (
	"fmt"
	"strings"

	"github.com/samborkent/cog/internal/tokens"
	"github.com/samborkent/cog/internal/types"
)

var _ Expr = &Complex128Literal{}

type Complex128Literal struct {
	Token tokens.Token
	Value complex128
}

func (l *Complex128Literal) Pos() (uint32, uint16) {
	return l.Token.Ln, l.Token.Col
}

func (l *Complex128Literal) Hash() uint64 {
	return hash(l)
}

func (l *Complex128Literal) StringTo(out *strings.Builder, _ *AST) {
	fmt.Fprintf(out, "(%g : complex128)", l.Value)
}

func (l *Complex128Literal) String() string {
	var out strings.Builder
	l.StringTo(&out, nil)
	return out.String()
}

func (l *Complex128Literal) Type() types.Type {
	return types.Basics[types.Complex128]
}
