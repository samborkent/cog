package ast

import (
	"strings"

	"github.com/samborkent/cog/internal/tokens"
)

var _ Node = &Switch{}

type Switch struct {
	Token      tokens.Token
	Label      *Identifier
	Identifier *Identifier // may be nil
	Cases      []*Case
	Default    *Default // may be nil
}

func (a *AST) NewSwitch(token tokens.Token, label *Identifier, ident *Identifier, cases []*Case, def *Default) NodeIndex {
	node := New[Switch](a)
	node.Token = token
	node.Label = label
	node.Identifier = ident
	node.Cases = cases
	node.Default = def

	return a.AddNode(node)
}

func (n *Switch) Pos() (ln uint32, col uint16) {
	return n.Token.Ln, n.Token.Col
}

func (n *Switch) Hash() uint64 {
	return hash(n)
}

func (n *Switch) StringTo(out *strings.Builder, a *AST) {
	if n.Label != nil {
		_, _ = out.WriteString(n.Label.Name)
		_, _ = out.WriteString(":\n")
	}

	_, _ = out.WriteString(n.Token.Type.String())
	_ = out.WriteByte(' ')

	if n.Identifier != nil {
		_, _ = out.WriteString(n.Identifier.Name)
	}

	_, _ = out.WriteString(" {\n")

	for _, c := range n.Cases {
		c.StringTo(out, a)
	}

	if n.Default != nil {
		n.Default.StringTo(out, a)
	}

	_ = out.WriteByte('}')
}
