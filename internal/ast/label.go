package ast

import (
	"strings"

	"github.com/samborkent/cog/internal/tokens"
)

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
	var out strings.Builder
	s.stringTo(&out)

	return out.String()
}

func (s *Label) stringTo(out *strings.Builder) {
	_, _ = out.WriteString(s.Label.Name)
	_, _ = out.WriteString(":")
}
