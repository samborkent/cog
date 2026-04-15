package types

import "github.com/samborkent/cog/internal/tokens"

var _ Type = &Reference{}

type Reference struct {
	Value Type
}

func (s *Reference) Kind() Kind {
	return ReferenceKind
}

func (s *Reference) String() string {
	return tokens.BitAnd.String() + s.Value.String()
}

func (s *Reference) Underlying() Type {
	return s
}
