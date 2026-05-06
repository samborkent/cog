package ast

import (
	"strings"

	"github.com/samborkent/cog/internal/tokens"
)

var _ Node = &Return{}

type Return struct {
	Token tokens.Token
	Value ExprIndex
}

func (a *AST) NewReturn(tok tokens.Token, value ExprIndex) NodeIndex {
	node := New[Return](a)
	node.Token = tok
	node.Value = value
	return a.AddNode(node)
}

func (n *Return) Pos() (uint32, uint16) {
	return n.Token.Ln, n.Token.Col
}

func (n *Return) Hash() uint64 {
	return hash(n)
}

func (n *Return) StringTo(out *strings.Builder, a *AST) {
	_, _ = out.WriteString(n.Token.Type.String())
	_ = out.WriteByte(' ')

	if n.Value != ZeroExprIndex {
		a.exprs[n.Value].StringTo(out, a)
	}
}
