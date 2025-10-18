package ast

import "github.com/samborkent/cog/internal/tokens"

var _ Statement = &IfStatement{}

type Label struct {
	statement

	Token tokens.Token
	Label *Identifier
}

func (s *Label) Pos() (uint32, uint16) {
	return s.Token.Ln, s.Token.Col
}

func (s *Label) Hash() uint64 {
	return hash(s)
}

func (s *Label) String() string {
	return s.Label.Name + ":"
}
