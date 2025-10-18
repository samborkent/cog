package ast

import "github.com/samborkent/cog/internal/tokens"

var _ Statement = &Break{}

type Break struct {
	statement

	Token tokens.Token
	Label *Identifier
}

func (b *Break) Pos() (uint32, uint16) {
	return b.Token.Ln, b.Token.Col
}

func (b *Break) Hash() uint64 {
	return hash(b)
}

func (b *Break) String() string {
	if b.Label != nil {
		return b.Token.Type.String() + " " + b.Label.Name
	}

	return b.Token.Type.String()
}
