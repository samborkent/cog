package ast

import (
	"strings"

	"github.com/samborkent/cog/internal/tokens"
)

var _ Node = &Defer{}

type Defer struct {
	Token tokens.Token
	Expr  ExprIndex
}

func (a *AST) NewDefer(tok tokens.Token, expr ExprIndex) NodeIndex {
	d := New[Defer](a)
	d.Token = tok
	d.Expr = expr
	return a.AddNode(d)
}

func (n *Defer) Pos() (uint32, uint16) {
	return n.Token.Ln, n.Token.Col
}

func (n *Defer) Hash() uint64 {
	return hash(n)
}

func (n *Defer) StringTo(out *strings.Builder, a *AST) {
	_, _ = out.WriteString("defer ")
	a.exprs[n.Expr].StringTo(out, a)
}