package ast

import (
	"strings"

	"github.com/samborkent/cog/internal/tokens"
)

var _ Statement = &IfStatement{}

type IfStatement struct {
	statement

	Token       tokens.Token
	Label       *Label
	Condition   Expression
	Consequence *Block
	Alternative *Block
}

func (s *IfStatement) Pos() (uint32, uint16) {
	return s.Token.Ln, s.Token.Col
}

func (s *IfStatement) Hash() uint64 {
	return hash(s)
}

func (s *IfStatement) stringTo(out *strings.Builder) {
	if s.Label != nil {
		s.Label.stringTo(out)
		_ = out.WriteByte(' ')
	}

	_, _ = out.WriteString("if (")
	s.Condition.stringTo(out)
	_, _ = out.WriteString(") ")
	s.Consequence.stringTo(out)

	if s.Alternative != nil {
		_, _ = out.WriteString(" else ")
		s.Alternative.stringTo(out)
	}
}

func (s *IfStatement) String() string {
	var out strings.Builder
	s.stringTo(&out)

	return out.String()
}
