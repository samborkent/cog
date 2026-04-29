package ast

import (
	"strings"

	"github.com/samborkent/cog/internal/tokens"
	"github.com/samborkent/cog/internal/types"
)

var _ Node = &Type{}

type Type struct {
	Token          tokens.Token
	Identifier     *Identifier
	TypeParameters []*types.Alias
	Alias          types.Type
}

func (a *AST) NewType(token tokens.Token, ident *Identifier, params []*types.Alias, alias types.Type) NodeIndex {
	node := New[Type](a)
	node.Token = token
	node.Identifier = ident
	node.TypeParameters = params
	node.Alias = alias
	return a.AddNode(node)
}

func (n *Type) Pos() (uint32, uint16) {
	return n.Token.Ln, n.Token.Col
}

func (n *Type) Hash() uint64 {
	return hash(n)
}

func (n *Type) StringTo(out *strings.Builder, _ *AST) {
	if n.Identifier.Exported {
		_, _ = out.WriteString("export ")
	}

	_, _ = out.WriteString(n.Identifier.Name)

	if len(n.TypeParameters) > 0 {
		_, _ = out.WriteString("<")

		for i, tp := range n.TypeParameters {
			if i > 0 {
				_, _ = out.WriteString(", ")
			}

			_, _ = out.WriteString(tp.Name)
			_, _ = out.WriteString(" ~ ")
			_, _ = out.WriteString(tp.ConstraintString())
		}

		_, _ = out.WriteString(">")
	}

	_, _ = out.WriteString(" ~ ")
	_, _ = out.WriteString(n.Alias.String())
}

func (n *Type) String() string {
	var out strings.Builder
	n.StringTo(&out, nil)
	return out.String()
}
