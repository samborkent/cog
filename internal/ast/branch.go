package ast

import (
	"strings"

	"github.com/samborkent/cog/internal/tokens"
)

var _ Node = &Branch{}

type Branch struct {
	Token tokens.Token // break or continue token
	Label *Identifier  // may be nil
}

func (a *AST) NewBranch(token tokens.Token, label *Identifier) NodeIndex {
	node := New[Branch](a)
	node.Token = token
	node.Label = label
	return a.AddNode(node)
}

func (n *Branch) Pos() (uint32, uint16) {
	return n.Token.Ln, n.Token.Col
}

func (n *Branch) Hash() uint64 {
	return hash(n)
}

func (n *Branch) StringTo(out *strings.Builder, _ *AST) {
	_, _ = out.WriteString(n.Token.Type.String())

	if n.Label != nil {
		_ = out.WriteByte(' ')
		_, _ = out.WriteString(n.Label.Name)
	}
}

func (n *Branch) String() string {
	var out strings.Builder
	n.StringTo(&out, nil)
	return out.String()
}
