package ast

import (
	"strings"

	"github.com/samborkent/cog/internal/tokens"
)

var _ Node = &ExpressionStatement{}

type ExpressionStatement struct {
	Token tokens.Token
	Expr  ExprIndex
}

func (a *AST) NewExpressionStatement(token tokens.Token, expr ExprIndex) NodeIndex {
	node := New[ExpressionStatement](a)
	node.Token = token
	node.Expr = expr
	return a.AddNode(node)
}

func (n *ExpressionStatement) Hash() uint64 {
	return hash(n)
}

func (n *ExpressionStatement) Pos() (uint32, uint16) {
	return n.Token.Ln, n.Token.Col
}

func (n *ExpressionStatement) StringTo(out *strings.Builder, a *AST) {
	a.exprs[n.Expr].StringTo(out, a)
}

func (n *ExpressionStatement) String() string {
	var out strings.Builder
	n.StringTo(&out, nil)
	return out.String()
}
