package ast

import (
	"strings"

	"github.com/samborkent/cog/internal/tokens"
)

var _ Node = &Import{}

type Import struct {
	Token   tokens.Token
	Imports []*Identifier
}

func (i *Import) Kind() NodeKind {
	return KindImport
}

func (i *Import) Pos() (uint32, uint16) {
	return i.Token.Ln, i.Token.Col
}

func (i *Import) Hash() uint64 {
	return hash(i)
}

func (i *Import) stringTo(out *strings.Builder) {
	_, _ = out.WriteString(i.Token.Type.String())
	_, _ = out.WriteString(" (\n")

	for _, imprt := range i.Imports {
		_, _ = out.WriteString("\t\"")
		_, _ = out.WriteString(imprt.Name)
		_, _ = out.WriteString("\"\n")
	}

	_ = out.WriteByte(')')
}

func (i *Import) String() string {
	var out strings.Builder
	i.stringTo(&out)

	return out.String()
}
