package ast

import (
	"strings"

	"github.com/samborkent/cog/internal/tokens"
)

var _ Node = &Comment{}

type Comment struct {
	Token tokens.Token
	Text  string
}

func (a *AST) NewComment(token tokens.Token) NodeIndex {
	node := New[Comment](a)
	node.Token = token
	node.Text = token.Literal
	return a.AddNode(node)
}

func (n *Comment) Hash() uint64 {
	return hash(n)
}

func (n *Comment) Pos() (uint32, uint16) {
	return n.Token.Ln, n.Token.Col
}

func (n *Comment) StringTo(out *strings.Builder, _ *AST) {
	_, _ = out.WriteString(n.Text)
}

func (n *Comment) String() string {
	var out strings.Builder

	n.StringTo(&out, nil)

	return out.String()
}
