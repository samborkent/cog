package ast

import (
	"strings"

	"github.com/samborkent/cog/internal/types"
)

var _ Node = &Parameter{}

type Parameter struct {
	Identifier *Identifier
	ValueType  types.Type
	Optional   bool
	Default    ExprIndex // optional
}

func (n *Parameter) Pos() (uint32, uint16) {
	return n.Identifier.Token.Ln, n.Identifier.Token.Col
}

func (n *Parameter) Hash() uint64 {
	return hash(n)
}

func (n *Parameter) StringTo(out *strings.Builder, a *AST) {
	if n.Identifier != nil && n.Identifier.Name != "" {
		_, _ = out.WriteString(n.Identifier.Name)

		if n.Optional {
			_, _ = out.WriteString("? : ")
		} else {
			_, _ = out.WriteString(" : ")
		}
	}

	_, _ = out.WriteString(n.ValueType.String())

	if n.Default != ZeroExprIndex {
		_, _ = out.WriteString(" = ")
		a.exprs[n.Default].StringTo(out, a)
	}
}
