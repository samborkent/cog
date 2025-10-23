package ast

import (
	"strings"

	"github.com/samborkent/cog/internal/tokens"
)

var _ Statement = &Procedure{}

type Procedure struct {
	statement

	Token      tokens.Token
	Identifier *Identifier
	Body       *Block
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

	_, _ = out.WriteString(p.Identifier.Name)
	_, _ = out.WriteString(" : ")

	_, _ = out.WriteString(p.Identifier.ValueType.String())

	if p.Body != nil {
		_, _ = out.WriteString(" = ")
		_, _ = out.WriteString(p.Body.String())
	}

	return out.String()
}

func (p *Procedure) Type() tokens.Type {
	return tokens.Procedure
}
