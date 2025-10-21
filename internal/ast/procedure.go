package ast

import (
	"strings"

	"github.com/samborkent/cog/internal/tokens"
	"github.com/samborkent/cog/internal/types"
)

var _ Statement = &Procedure{}

type Procedure struct {
	statement

	Token           tokens.Token
	Identifier      *Identifier
	InputParameters []*Parameter
	ReturnType      types.Type // optional
	Body            *Block
}

func (p *Procedure) Pos() (uint32, uint16) {
	return p.Token.Ln, p.Token.Col
}

func (p *Procedure) Hash() uint64 {
	return hash(p)
}

func (p *Procedure) String() string {
	var out strings.Builder

	if p.Identifier.Exported {
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

	if p.ReturnType != nil {
		_, _ = out.WriteString(p.ReturnType.String())
		_ = out.WriteByte(' ')
	}

	if p.Body != nil {
		_, _ = out.WriteString(p.Body.String())
	}

	return out.String()
}

func (p *Procedure) Type() tokens.Type {
	return tokens.Procedure
}
