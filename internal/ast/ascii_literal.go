package ast

import (
	"fmt"
	"strings"
	"unicode"

	"github.com/samborkent/cog/internal/tokens"
	"github.com/samborkent/cog/internal/types"
)

const (
	minPrintableASCII = 32
	maxPrintableASCII = unicode.MaxASCII
)

type ascii = []byte

var _ Expr = &ASCIILiteral{}

type ASCIILiteral struct {
	Token tokens.Token
	Value ascii
}

func (a *AST) NewASCIILiteral(t tokens.Token) (ExprIndex, error) {
	for _, r := range t.Literal {
		if r < minPrintableASCII || r > maxPrintableASCII {
			return ZeroExprIndex, fmt.Errorf("string literal contains non-printable ASCII character %q", r)
		}
	}

	expr := New[ASCIILiteral](a)
	expr.Token = t
	expr.Value = ascii(t.Literal)

	return a.AddExpr(expr), nil
}

func (l *ASCIILiteral) Pos() (uint32, uint16) {
	return l.Token.Ln, l.Token.Col
}

func (l *ASCIILiteral) Hash() uint64 {
	return hash(l)
}

func (l *ASCIILiteral) StringTo(out *strings.Builder, _ *AST) {
	_, _ = out.WriteString("(\"")
	out.Write(l.Value)
	_, _ = out.WriteString("\" : ascii)")
}

func (l *ASCIILiteral) String() string {
	var out strings.Builder
	l.StringTo(&out, nil)
	return out.String()
}

func (l *ASCIILiteral) Type() types.Type {
	return types.Basics[types.ASCII]
}
