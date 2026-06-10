package ast

import (
	"strings"

	"github.com/samborkent/cog/internal/tokens"
	"github.com/samborkent/cog/internal/types"
)

var _ Node = &Type{}

// Type is a type declaration.
type Type struct {
	Token tokens.Token
	Alias *types.Alias
}

func (a *AST) NewType(token tokens.Token, ident *Identifier, params []*types.Alias, alias types.Type) NodeIndex {
	node := New[Type](a)
	node.Token = token
	node.Alias = &types.Alias{
		Name:           ident.Token.Literal,
		Derived:        alias,
		Exported:       ident.Exported,
		Global:         ident.Global,
		TypeParameters: params,
	}
	return a.AddNode(node)
}

func (n *Type) Pos() (uint32, uint16) {
	return n.Token.Ln, n.Token.Col
}

func (n *Type) Hash() uint64 {
	return hash(n)
}

func (n *Type) StringTo(out *strings.Builder, _ *AST) {
	if n.Alias.Exported {
		_, _ = out.WriteString("export ")
	}

	_, _ = out.WriteString(n.Alias.Name)

	if len(n.Alias.TypeParameters) > 0 {
		_, _ = out.WriteString("<")

		for i, tp := range n.Alias.TypeParameters {
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
