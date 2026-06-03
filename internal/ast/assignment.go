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

	FieldName     string // set for field-target assignments (c.data = ...)
	FieldExported bool   // export status of the struct field
}

type assignmentOption func(*Assignment)

func WithField(name string, exported bool) assignmentOption {
	return func(n *Assignment) {
		n.FieldName = name
		n.FieldExported = exported
	}
}

func (a *AST) NewAssignment(token tokens.Token, ident *Identifier, expr ExprIndex, opts ...assignmentOption) NodeIndex {
	node := New[Assignment](a)
	node.Token = token
	node.Identifier = ident
	node.Expr = expr

	for _, opt := range opts {
		opt(node)
	}

	return a.AddNode(node)
}

func (n *Assignment) Pos() (uint32, uint16) {
	return n.Token.Ln, n.Token.Col
}

func (n *Assignment) Hash() uint64 {
	return hash(n)
}

func (n *Assignment) StringTo(out *strings.Builder, a *AST) {
	_, _ = out.WriteString(n.Identifier.Token.Literal)

	if n.FieldName != "" {
		_ = out.WriteByte('.')
		_, _ = out.WriteString(n.FieldName)
	}

	_ = out.WriteByte(' ')
	_, _ = out.WriteString(n.Token.Type.String())
	_ = out.WriteByte(' ')
	a.exprs[n.Expr].StringTo(out, a)
}
