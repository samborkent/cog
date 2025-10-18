package ast

import (
	"strings"

	"github.com/samborkent/cog/internal/tokens"
)

var _ Statement = &GoImport{}

type GoImport struct {
	statement

	Token   tokens.Token
	Imports []*Identifier
}

func (g *GoImport) Pos() (uint32, uint16) {
	return g.Token.Ln, g.Token.Col
}

func (g *GoImport) Hash() uint64 {
	return hash(g)
}

func (g *GoImport) String() string {
	var out strings.Builder

	_, _ = out.WriteString(g.Token.Type.String())
	_, _ = out.WriteString(" (\n")

	for _, imprt := range g.Imports {
		_, _ = out.WriteString("\t\"")
		_, _ = out.WriteString(imprt.Name)
		_, _ = out.WriteString("\"\n")
	}

	_ = out.WriteByte(')')

	return out.String()
}
