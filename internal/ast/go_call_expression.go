package ast

import (
	"github.com/samborkent/cog/internal/tokens"
	"github.com/samborkent/cog/internal/types"
)

var _ Expression = &GoCallExpression{}

type GoCallExpression struct {
	expression

	Token          tokens.Token
	Import         *Identifier
	CallIdentifier *Identifier
	Arguments      []Expression
}

func (e *GoCallExpression) Pos() (uint32, uint16) {
	return e.Token.Ln, e.Token.Col
}

func (e *GoCallExpression) Hash() uint64 {
	return hash(e)
}

func (e *GoCallExpression) String() string {
	return "@go" + "." + e.Import.Name + "." + e.CallIdentifier.Name
}

// TODO: figure out how to infer Go types and how to convert between Cog and Go types
func (e *GoCallExpression) Type() types.Type {
	// Currently, we cannot infer Go types.
	return types.None
}
