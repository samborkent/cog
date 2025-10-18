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

func (s *IfStatement) String() string {
	var out strings.Builder

	if s.Label != nil {
		_, _ = out.WriteString(s.Label.String())
		_ = out.WriteByte(' ')
	}

	_, _ = out.WriteString("if (")
	_, _ = out.WriteString(s.Condition.String())
	_, _ = out.WriteString(") ")
	_, _ = out.WriteString(s.Consequence.String())

	if s.Alternative != nil {
		_, _ = out.WriteString(" else ")
		_, _ = out.WriteString(s.Alternative.String())
	}

	return out.String()
}
