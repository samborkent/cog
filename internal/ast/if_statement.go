package ast

import (
	"strings"

	"github.com/samborkent/cog/internal/tokens"
)

var _ Node = &IfStatement{}

type IfStatement struct {
	Token       tokens.Token
	Label       *Identifier
	Condition   ExprIndex
	Consequence *Block
	Alternative *Block // may be nil
}

func (a *AST) NewIfStatement(token tokens.Token, label *Identifier, condition ExprIndex, consequence, alternative *Block) NodeIndex {
	node := New[IfStatement](a)
	node.Token = token
	node.Label = label
	node.Condition = condition
	node.Consequence = consequence
	node.Alternative = alternative
	return a.AddNode(node)
}

func (n *IfStatement) Pos() (uint32, uint16) {
	return n.Token.Ln, n.Token.Col
}

func (n *IfStatement) Hash() uint64 {
	return hash(n)
}

func (n *IfStatement) StringTo(out *strings.Builder, a *AST) {
	if n.Label != nil {
		_, _ = out.WriteString(n.Label.Name)
		_, _ = out.WriteString(":\n")
	}

	_, _ = out.WriteString("if (")
	a.exprs[n.Condition].StringTo(out, a)
	_, _ = out.WriteString(") ")
	n.Consequence.StringTo(out, a)

	if n.Alternative != nil {
		_, _ = out.WriteString(" else ")
		n.Alternative.StringTo(out, a)
	}
}
