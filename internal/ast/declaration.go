package ast

import (
	"strings"

	"github.com/samborkent/cog/internal/tokens"
	"github.com/samborkent/cog/internal/types"
)

var _ Node = &Declaration{}

type Declaration struct {
	Token      tokens.Token
	Assignment *Assignment
}

func (a *AST) NewDeclaration(token tokens.Token, assignment *Assignment) NodeIndex {
	node := New[Declaration](a)
	node.Token = token
	node.Assignment = assignment
	return a.AddNode(node)
}

func (n *Declaration) Pos() (uint32, uint16) {
	return n.Token.Ln, n.Token.Col
}

func (n *Declaration) Hash() uint64 {
	return hash(n)
}

func (n *Declaration) StringTo(out *strings.Builder, a *AST) {
	if n.Assignment.Identifier.Exported {
		_, _ = out.WriteString("export ")
	}

	switch n.Assignment.Identifier.Qualifier {
	case QualifierVariable:
		_, _ = out.WriteString("var ")
	case QualifierDynamic:
		_, _ = out.WriteString("dyn ")
	}

	if n.Assignment.Expr == 0 {
		_, _ = out.WriteString(n.Assignment.Identifier.Name)
		_, _ = out.WriteString(" : ")
		_, _ = out.WriteString(n.Assignment.Identifier.ValueType.String())

		return
	}

	if n.Assignment.Identifier.ValueType == nil || n.Assignment.Identifier.ValueType == types.None {
		_, _ = out.WriteString(n.Assignment.Identifier.Name)
		_, _ = out.WriteString(" := ")
		a.exprs[n.Assignment.Expr].StringTo(out, a)

		return
	}

	n.Assignment.StringTo(out, a)
}

func (n *Declaration) String() string {
	var out strings.Builder
	n.StringTo(&out, nil)
	return out.String()
}
