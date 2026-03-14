package ast

import "github.com/samborkent/cog/internal/tokens"

var _ Statement = &Branch{}

type Branch struct {
	statement

	Token tokens.Token // break or continue token
	Label *Identifier
}

func (b *Branch) Pos() (uint32, uint16) {
	return b.Token.Ln, b.Token.Col
}

func (b *Branch) Hash() uint64 {
	return hash(b)
}

func (b *Branch) String() string {
	if b.Label != nil {
		return b.Token.Type.String() + " " + b.Label.Name
	}

	return b.Token.Type.String()
}
