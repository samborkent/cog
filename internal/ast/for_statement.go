package ast

import (
	"strings"

	"github.com/samborkent/cog/internal/tokens"
)

var _ Node = &ForStatement{}

type ForStatement struct {
	Token tokens.Token
	Label *Identifier
	Value *Identifier
	Index *Identifier
	Range ExprIndex
	Loop  *Block
}

func (a *AST) NewForStatement(t tokens.Token, label *Identifier, value *Identifier, index *Identifier, rangeExpr ExprIndex, loop *Block) NodeIndex {
	expr := New[ForStatement](a)
	expr.Token = t
	expr.Label = label
	expr.Value = value
	expr.Index = index
	expr.Range = rangeExpr
	expr.Loop = loop

	return a.AddNode(expr)
}

func (n *ForStatement) Pos() (uint32, uint16) {
	return n.Token.Ln, n.Token.Col
}

func (n *ForStatement) Hash() uint64 {
	return hash(n)
}

func (n *ForStatement) StringTo(out *strings.Builder, a *AST) {
	if n.Label != nil {
		_, _ = out.WriteString(n.Label.Name)
		_, _ = out.WriteString(":\n")
	}

	_, _ = out.WriteString("for ")

	if n.Value != nil {
		_, _ = out.WriteString(n.Value.Name)
		_, _ = out.WriteString(" in ")
	}

	if n.Range != ZeroExprIndex {
		a.exprs[n.Range].StringTo(out, a)
		_ = out.WriteByte(' ')
	}

	n.Loop.StringTo(out, a)
}
