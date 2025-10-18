package ast

var _ Statement = &Parameter{}

type Parameter struct {
	statement

	Identifier *Identifier
	Default    Expression // optional
}

func (p *Parameter) Pos() (uint32, uint16) {
	return p.Identifier.Token.Ln, p.Identifier.Token.Col
}

func (p *Parameter) Hash() uint64 {
	return hash(p)
}

func (p *Parameter) String() string {
	str := p.Identifier.Type().String()

	if p.Identifier.Name != "" {
		str = p.Identifier.Name + ": " + str
	}

	if p.Default == nil {
		return str
	}

	return str + " = " + p.Default.String()
}
