package ast

import (
	"strings"

	"github.com/samborkent/cog/internal/tokens"
	"github.com/samborkent/cog/internal/types"
)

var _ Expr = &GoCallExpression{}

type GoCallExpression struct {
	Token          tokens.Token
	Import         *Identifier
	CallIdentifier *Identifier
	Arguments      []ExprIndex
}

func (e *GoCallExpression) Pos() (uint32, uint16) {
	return e.Token.Ln, e.Token.Col
}

func (e *GoCallExpression) Hash() uint64 {
	return hash(e)
}

func (e *GoCallExpression) StringTo(out *strings.Builder, a *AST) {
	_, _ = out.WriteString("@go.")
	_, _ = out.WriteString(e.Import.Name)
	_ = out.WriteByte('.')
	_, _ = out.WriteString(e.CallIdentifier.Name)
	_ = out.WriteByte('(')

	for i, arg := range e.Arguments {
		if i > 0 {
			_, _ = out.WriteString(", ")
		}

		a.exprs[arg].StringTo(out, a)
	}

	_ = out.WriteByte(')')
}

func (e *GoCallExpression) String() string {
	var out strings.Builder
	e.StringTo(&out, nil)
	return out.String()
}

// TODO: figure out how to infer Go types and how to convert between Cog and Go types
func (e *GoCallExpression) Type() types.Type {
	// Currently, we cannot infer Go types.
	return types.None
}
