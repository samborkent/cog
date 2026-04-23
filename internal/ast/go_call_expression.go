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
	Arguments      []ExprValue
}

func (e *GoCallExpression) Kind() NodeKind {
	return KindGoCallExpression
}

func (e *GoCallExpression) Pos() (uint32, uint16) {
	return e.Token.Ln, e.Token.Col
}

func (e *GoCallExpression) Hash() uint64 {
	return hash(e)
}

func (e *GoCallExpression) stringTo(out *strings.Builder) {
	_, _ = out.WriteString("@go.")
	_, _ = out.WriteString(e.Import.Name)
	_ = out.WriteByte('.')
	_, _ = out.WriteString(e.CallIdentifier.Name)
}

func (e *GoCallExpression) String() string {
	var out strings.Builder
	e.stringTo(&out)

	return out.String()
}

// TODO: figure out how to infer Go types and how to convert between Cog and Go types
func (e *GoCallExpression) Type() types.Type {
	// Currently, we cannot infer Go types.
	return types.None
}
