package ast

import (
	"strings"

	"github.com/samborkent/cog/internal/tokens"
)

var _ Node = &Match{}

// Match represents a match statement:
//
//	match [Identifier :=] Subject {
//	  case TypeA:
//	    ...
//	}
type Match struct {
	Token   tokens.Token
	Label   *Identifier // Optional label for the match statement.
	Binding *Identifier // Optional binding variable.
	Subject ExprIndex
	Cases   []*MatchCase
	Default *Default
}

func (a *AST) NewMatch(t tokens.Token, label *Identifier, binding *Identifier, subject ExprIndex, cases []*MatchCase, defaultCase *Default) NodeIndex {
	node := New[Match](a)

	node.Token = t
	node.Label = label
	node.Binding = binding
	node.Subject = subject
	node.Cases = cases
	node.Default = defaultCase

	return a.AddNode(node)
}

func (n *Match) Pos() (uint32, uint16) {
	return n.Token.Ln, n.Token.Col
}

func (n *Match) Hash() uint64 {
	return hash(n)
}

func (n *Match) StringTo(out *strings.Builder, a *AST) {
	if n.Label != nil {
		_, _ = out.WriteString(n.Label.Name)
		_, _ = out.WriteString(":\n")
	}

	out.WriteString("match ")

	if n.Binding != nil {
		_, _ = out.WriteString(n.Binding.Name)
		_, _ = out.WriteString(" := ")
	}

	if a != nil {
		a.exprs[n.Subject].StringTo(out, a)
	} else {
		_, _ = out.WriteString("<expr>")
	}
	_, _ = out.WriteString(" {\n")

	for _, c := range n.Cases {
		c.StringTo(out, a)
	}

	if n.Default != nil {
		n.Default.StringTo(out, a)
	}

	_, _ = out.WriteString("}\n")
}
