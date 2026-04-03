package ast

import (
	"strings"

	"github.com/samborkent/cog/internal/types"
)

var _ Expression = &Call{}

type Call struct {
	expression

	Identifier *Identifier
	Package    string // non-empty when calling an imported package's function
	Arguments  []Expression
	ReturnType types.Type
	TypeArgs   []types.Type // explicit or inferred type arguments for generic calls
}

func (c *Call) Pos() (uint32, uint16) {
	return c.Identifier.Pos()
}

func (c *Call) Hash() uint64 {
	return hash(c)
}

func (c *Call) stringTo(out *strings.Builder) {
	if c.Package != "" {
		_, _ = out.WriteString(c.Package)
		_ = out.WriteByte('.')
	}

	c.Identifier.stringTo(out)

	if len(c.TypeArgs) > 0 {
		_ = out.WriteByte('<')

		for i, ta := range c.TypeArgs {
			if i > 0 {
				_, _ = out.WriteString(", ")
			}

			_, _ = out.WriteString(ta.String())
		}

		_ = out.WriteByte('>')
	}

	_ = out.WriteByte('(')

	for i, arg := range c.Arguments {
		arg.stringTo(out)

		if i < len(c.Arguments)-1 {
			_, _ = out.WriteString(", ")
		}
	}

	_ = out.WriteByte(')')
}

func (c *Call) String() string {
	var out strings.Builder
	c.stringTo(&out)

	return out.String()
}

// TODO: return a proper return type here
func (c *Call) Type() types.Type {
	if c.ReturnType == nil {
		panic("call with nil return type detected")
	}

	return c.ReturnType
}
