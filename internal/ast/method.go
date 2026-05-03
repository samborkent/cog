package ast

import (
	"strings"

	"github.com/samborkent/cog/internal/tokens"
	"github.com/samborkent/cog/internal/types"
)

var _ Node = &Method{}

type Method struct {
	Token       tokens.Token
	Export      bool
	Receiver    *Identifier
	Type        types.Type
	Declaration NodeIndex
}

func (a *AST) NewMethod(t tokens.Token, export bool, receiver *Identifier, typ types.Type, declaration NodeIndex) NodeIndex {
	node := New[Method](a)

	node.Token = t
	node.Export = export
	node.Receiver = receiver
	node.Type = typ
	node.Declaration = declaration

	return a.AddNode(node)
}

func (n *Method) Pos() (uint32, uint16) {
	return n.Token.Ln, n.Token.Col
}

func (n *Method) Hash() uint64 {
	return hash(n)
}

func (n *Method) StringTo(out *strings.Builder, a *AST) {
	if n.Export {
		_, _ = out.WriteString("export ")
	}

	if n.Receiver != nil {
		_ = out.WriteByte('(')
		_, _ = out.WriteString(n.Receiver.Name)
		_, _ = out.WriteString(" : ")
		_, _ = out.WriteString(n.Type.String())
		_ = out.WriteByte(')')
	} else {
		_, _ = out.WriteString(n.Type.String())
	}

	_ = out.WriteByte('.')

	a.nodes[n.Declaration].StringTo(out, a)
}
