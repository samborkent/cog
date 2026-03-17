package ast

import (
	"strings"

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

func (s *Type) stringTo(out *strings.Builder) {
	if s.Identifier.Exported {
		_, _ = out.WriteString("export ")
	}

	_, _ = out.WriteString(s.Identifier.Name)
	_, _ = out.WriteString(" ~ ")
	_, _ = out.WriteString(s.Alias.String())
}

func (s *Type) String() string {
	var out strings.Builder
	s.stringTo(&out)
	return out.String()
}
