package ast

import "github.com/samborkent/cog/internal/types"

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

func (p *Parameter) String() string {
	str := p.ValueType.String()

	if p.Identifier != nil && p.Identifier.Name != "" {
		if p.Optional {
			str = p.Identifier.Name + "? : " + str
		} else {
			str = p.Identifier.Name + " : " + str
		}
	}

	if p.Default == nil {
		return str
	}

	return str + " = " + p.Default.String()
}
