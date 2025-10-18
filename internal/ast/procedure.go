package ast

import (
	"strings"

	"github.com/samborkent/cog/internal/tokens"
)

var _ Statement = &Procedure{}

type Procedure struct {
	statement

	Token            tokens.Token
	Identifier       *Identifier
	InputParameters  []*Parameter
	ReturnParameters []*Parameter
	Body             *Block
	Exported         bool
}

func (p *Procedure) Pos() (uint32, uint16) {
	return p.Token.Ln, p.Token.Col
}

func (p *Procedure) Hash() uint64 {
	return hash(p)
}

func (p *Procedure) String() string {
	var out strings.Builder

	if p.Exported {
		_, _ = out.WriteString("export ")
	}

	_, _ = out.WriteString(p.Identifier.String())
	_, _ = out.WriteString(" : ")
	_, _ = out.WriteString(p.Token.Type.String())
	_ = out.WriteByte('(')

	for i, param := range p.InputParameters {
		_, _ = out.WriteString(param.String())

		if i < len(p.InputParameters)-1 {
			_, _ = out.WriteString(", ")
		}
	}

	_, _ = out.WriteString(") ")

	if len(p.ReturnParameters) > 0 {
		if len(p.ReturnParameters) > 1 {
			_ = out.WriteByte('(')
		}

		for i, param := range p.ReturnParameters {
			_, _ = out.WriteString(param.String())

			if i < len(p.ReturnParameters)-1 {
				_, _ = out.WriteString(", ")
			}
		}

		if len(p.ReturnParameters) > 1 {
			_, _ = out.WriteString(") ")
		}
	}

	if p.Body != nil {
		_, _ = out.WriteString(p.Body.String())
	}

	return out.String()
}

func (p *Procedure) Type() tokens.Type {
	return tokens.Procedure
}
