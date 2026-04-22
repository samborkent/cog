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

func NewASCIILiteral(t tokens.Token) (*ASCIILiteral, error) {
	for _, r := range t.Literal {
		if r < minPrintableASCII || r > maxPrintableASCII {
			return nil, fmt.Errorf("string literal contains non-printable ASCII character %q", r)
		}
	}

	return &ASCIILiteral{
		Token: t,
		Value: ascii(t.Literal),
	}, nil
}

func (l *ASCIILiteral) Pos() (uint32, uint16) {
	return l.Token.Ln, l.Token.Col
}

func (l *ASCIILiteral) Hash() uint64 {
	return hash(l)
}

func (l *ASCIILiteral) stringTo(out *strings.Builder) {
	_, _ = out.WriteString("(\"")
	out.Write(l.Value)
	_, _ = out.WriteString("\" : ascii)")
}

func (l *ASCIILiteral) String() string {
	var out strings.Builder
	l.stringTo(&out)

	return out.String()
}

func (l *ASCIILiteral) Type() types.Type {
	return types.Basics[types.ASCII]
}
