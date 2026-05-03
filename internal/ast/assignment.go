package ast

import (
	"strings"

	"github.com/samborkent/cog/internal/tokens"
)

var _ Node = &Assignment{}

type Assignment struct {
	Token      tokens.Token
	Identifier *Identifier
	Expr       ExprIndex
}

func (a *AST) NewAssignment(token tokens.Token, ident *Identifier, expr ExprIndex) NodeIndex {
	node := New[Assignment](a)
	node.Token = token
	node.Identifier = ident
	node.Expr = expr
	return a.AddNode(node)
}

func (n *Assignment) Pos() (uint32, uint16) {
	return n.Token.Ln, n.Token.Col
}

func (n *Assignment) Hash() uint64 {
	return hash(n)
}

func (n *Assignment) StringTo(out *strings.Builder, a *AST) {
	_, _ = out.WriteString(n.Identifier.Name)
	_ = out.WriteByte(' ')
	_, _ = out.WriteString(n.Token.Type.String())
	_ = out.WriteByte(' ')
	a.exprs[n.Expr].StringTo(out, a)
}
