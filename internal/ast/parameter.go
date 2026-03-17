package ast

import (
	"strings"

	"github.com/samborkent/cog/internal/types"
)

var _ Statement = &Parameter{}

type Parameter struct {
	statement

	Identifier *Identifier
	ValueType  types.Type
	Optional   bool
	Default    Expression // optional
}

func (p *Parameter) Pos() (uint32, uint16) {
	return p.Identifier.Token.Ln, p.Identifier.Token.Col
}

func (p *Parameter) Hash() uint64 {
	return hash(p)
}

func (p *Parameter) stringTo(out *strings.Builder) {
	if p.Identifier != nil && p.Identifier.Name != "" {
		_, _ = out.WriteString(p.Identifier.Name)

		if p.Optional {
			_, _ = out.WriteString("? : ")
		} else {
			_, _ = out.WriteString(" : ")
		}
	}

	_, _ = out.WriteString(p.ValueType.String())

	if p.Default != nil {
		_, _ = out.WriteString(" = ")
		p.Default.stringTo(out)
	}
}

func (p *Parameter) String() string {
	var out strings.Builder
	p.stringTo(&out)
	return out.String()
}
