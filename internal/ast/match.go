package ast

import (
	"strings"

	"github.com/samborkent/cog/internal/tokens"
)

var _ Node = &Match{}

// Match represents a match statement:
//
//	match [Identifier :=] Subject {
//	  case TypeA:
//	    ...
//	}
type Match struct {
	Token   tokens.Token
	Subject ExprValue
	Binding *Identifier // Optional binding variable.
	Cases   []*MatchCase
	Default *Default
}

func (m *Match) Kind() NodeKind {
	return KindMatch
}

func (m *Match) Pos() (uint32, uint16) {
	return m.Token.Ln, m.Token.Col
}

func (m *Match) Hash() uint64 {
	return hash(m)
}

func (m *Match) stringTo(out *strings.Builder) {
	out.WriteString("match ")

	if m.Binding != nil {
		m.Binding.stringTo(out)
		out.WriteString(" := ")
	}

	m.Subject.expr.stringTo(out)
	out.WriteString(" {\n")

	for _, c := range m.Cases {
		c.stringTo(out)
	}

	if m.Default != nil {
		m.Default.stringTo(out)
	}

	out.WriteString("}\n")
}

func (m *Match) String() string {
	var out strings.Builder
	m.stringTo(&out)

	return out.String()
}
