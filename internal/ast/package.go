package ast

import "github.com/samborkent/cog/internal/tokens"

var _ Statement = &Package{}

type Package struct {
	statement

	Token      tokens.Token
	Identifier *Identifier
}

func (p *Package) Pos() (uint32, uint16) {
	return p.Token.Ln, p.Token.Col
}

func (p *Package) Hash() uint64 {
	return hash(p)
}

func (p *Package) String() string {
	return p.Token.Type.String() + " " + p.Identifier.String()
}
