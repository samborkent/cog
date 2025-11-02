package ast

import (
	"github.com/samborkent/cog/internal/tokens"
	"github.com/samborkent/cog/internal/types"
)

var _ Statement = &Type{}

type Type struct {
	statement

	Token      tokens.Token
	Identifier *Identifier
	Alias      types.Type
}

func (s *Type) Pos() (uint32, uint16) {
	return s.Token.Ln, s.Token.Col
}

func (s *Type) Hash() uint64 {
	return hash(s)
}

func (s *Type) String() string {
	str := s.Identifier.Name + " ~ " + s.Alias.String()

	if s.Identifier.Exported {
		str = "export " + str
	}

	return str
}
