package ast

import (
	"github.com/samborkent/cog/internal/tokens"
	"github.com/samborkent/cog/internal/types"
)

var _ Expression = &Selector{}

type Selector struct {
	expression

	Token      tokens.Token
	Identifier *Identifier
	Field      *Identifier
}

func (e *Selector) Pos() (uint32, uint16) {
	return e.Token.Ln, e.Token.Col
}

func (e *Selector) Hash() uint64 {
	return hash(e)
}

func (e *Selector) String() string {
	return e.Identifier.Name + "." + e.Field.Name
}

func (e *Selector) Type() types.Type {
	return e.Field.Type()
}
