package ast

import (
	"fmt"
	goast "go/ast"
	gotoken "go/token"
	"unicode"
	"unsafe"

	"github.com/samborkent/cog/internal/tokens"
	"github.com/samborkent/cog/internal/types"
)

const (
	minPrintableASCII = 32
	maxPrintableASCII = unicode.MaxASCII
)

type ascii = []byte

var _ Expression = &ASCIILiteral{}

type ASCIILiteral struct {
	expression

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

func (l *ASCIILiteral) Go() *goast.CompositeLit {
	elems := make([]goast.Expr, len(l.Value))

	for i := range l.Value {
		elems[i] = &goast.BasicLit{
			Kind:  gotoken.CHAR,
			Value: "'" + string(l.Value[i]) + "'", // TODO: get rid of allocations
		}
	}

	return &goast.CompositeLit{
		Type: &goast.SelectorExpr{
			X:   &goast.Ident{Name: "cog"},
			Sel: &goast.Ident{Name: "ASCII"},
		},
		Elts: elems,
	}
}

func (l *ASCIILiteral) Hash() uint64 {
	return hash(l)
}

func (l *ASCIILiteral) String() string {
	//nolint:gosec // G103: unsafe use
	return "(\"" + unsafe.String(&l.Value[0], len(l.Value)) + "\" : ascii)"
}

func (l *ASCIILiteral) Type() types.Type {
	return types.Basics[types.ASCII]
}
