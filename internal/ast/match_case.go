package ast

import (
	"strings"

	"github.com/samborkent/cog/internal/tokens"
	"github.com/samborkent/cog/internal/types"
)

var _ Node = &MatchCase{}

// MatchCase represents a single case arm in a match statement.
type MatchCase struct {
	Token     tokens.Token
	MatchType types.Type
	Tilde     bool
	Body      []Statement
}

func (m *MatchCase) Pos() (uint32, uint16) {
	return m.Token.Ln, m.Token.Col
}

func (m *MatchCase) Hash() uint64 {
	return hash(m)
}

func (m *MatchCase) stringTo(out *strings.Builder) {
	_, _ = out.WriteString("case ")

	if m.Tilde {
		_ = out.WriteByte('~')
	}

	_, _ = out.WriteString(m.MatchType.String())
	_, _ = out.WriteString(":\n")

	for _, stmt := range m.Body {
		stmt.stringTo(out)
	}
}

func (m *MatchCase) String() string {
	var out strings.Builder
	m.stringTo(&out)

	return out.String()
}
