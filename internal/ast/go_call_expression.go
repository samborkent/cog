package ast

import (
	"github.com/samborkent/cog/internal/tokens"
	"github.com/samborkent/cog/internal/types"
)

var _ Expression = &GoCallExpression{}

type GoCallExpression struct {
	expression

	Token  tokens.Token
	Import *Identifier
	Call   *Call
}

func (e *GoCallExpression) Pos() (uint32, uint16) {
	return e.Token.Ln, e.Token.Col
}

func (e *GoCallExpression) Hash() uint64 {
	return hash(e)
}

func (e *GoCallExpression) String() string {
	return "@go" + "." + e.Import.Name + "." + e.Call.String()
}

// TODO: figure out how to infer Go types and how to convert between Cog and Go types
func (e *GoCallExpression) Type() types.Type {
	if e.Call.Type() == nil {
		panic("go call with nil type detected")
	}

	return e.Call.Type()
}
