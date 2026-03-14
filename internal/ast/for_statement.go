package ast

import (
	"strings"

	"github.com/samborkent/cog/internal/tokens"
)

var _ Statement = &ForStatement{}

type ForStatement struct {
	statement

	Token tokens.Token
	Label *Label
	Value *Identifier
	Index *Identifier
	Range Expression
	Loop  *Block
}

func (s *ForStatement) Pos() (uint32, uint16) {
	return s.Token.Ln, s.Token.Col
}

func (s *ForStatement) Hash() uint64 {
	return hash(s)
}

func (s *ForStatement) String() string {
	var out strings.Builder
	s.stringTo(&out)
	return out.String()
}

func (s *ForStatement) stringTo(out *strings.Builder) {
	if s.Label != nil {
		s.Label.stringTo(out)
		_ = out.WriteByte(' ')
	}

	_, _ = out.WriteString("for ")

	if s.Value != nil {
		_, _ = out.WriteString(s.Value.Name)
		_, _ = out.WriteString(" in ")
	}

	if s.Range != nil {
		_, _ = out.WriteString(s.Range.String())
		_ = out.WriteByte(' ')
	}

	s.Loop.stringTo(out)
}
