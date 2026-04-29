package ast

import (
	"strings"

	"github.com/samborkent/cog/internal/tokens"
)

var _ Node = &GoImport{}

type GoImport struct {
	Token   tokens.Token
	Imports []*Identifier
}

func (a *AST) NewGoImport(token tokens.Token, imports []*Identifier) NodeIndex {
	node := New[GoImport](a)
	node.Token = token
	node.Imports = imports
	return a.AddNode(node)
}

func (n *GoImport) Pos() (uint32, uint16) {
	return n.Token.Ln, n.Token.Col
}

func (n *GoImport) Hash() uint64 {
	return hash(n)
}

func (n *GoImport) StringTo(out *strings.Builder, _ *AST) {
	_, _ = out.WriteString(n.Token.Type.String())
	_, _ = out.WriteString(" (\n")

	for _, imprt := range n.Imports {
		_, _ = out.WriteString("\t\"")
		_, _ = out.WriteString(imprt.Name)
		_, _ = out.WriteString("\"\n")
	}

	_ = out.WriteByte(')')
}

func (n *GoImport) String() string {
	var out strings.Builder
	n.StringTo(&out, nil)
	return out.String()
}
