package ast

import (
	"strings"

	"github.com/samborkent/cog/internal/tokens"
	"github.com/samborkent/cog/internal/types"
)

var _ Expression = &Call{}

type Call struct {
	expression

	Token     tokens.Token // The '(' token
	Procedure *Procedure
	Arguments []Expression
}

func (c *Call) Pos() (uint32, uint16) {
	return c.Procedure.Identifier.Pos()
}

func (c *Call) Hash() uint64 {
	return hash(c)
}

func (c *Call) String() string {
	var out strings.Builder

	_, _ = out.WriteString(c.Procedure.Identifier.String())
	_ = out.WriteByte('(')

	for i, arg := range c.Arguments {
		_, _ = out.WriteString(arg.String())

		if i < len(c.Arguments)-1 {
			_, _ = out.WriteString(", ")
		}
	}

	_ = out.WriteByte(')')

	return out.String()
}

// TODO: return a proper return type here
func (c *Call) Type() types.Type {
	if c.Procedure == nil || c.Procedure.ReturnType == nil {
		panic("call with nil return type detected")
	}

	return c.Procedure.ReturnType
}
