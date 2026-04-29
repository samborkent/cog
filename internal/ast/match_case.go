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
	Body      []NodeIndex
}

func (n *MatchCase) Pos() (uint32, uint16) {
	return n.Token.Ln, n.Token.Col
}

func (n *MatchCase) Hash() uint64 {
	return hash(n)
}

func (n *MatchCase) StringTo(out *strings.Builder, a *AST) {
	_, _ = out.WriteString("case ")

	if n.Tilde {
		_ = out.WriteByte('~')
	}

	_, _ = out.WriteString(n.MatchType.String())
	_, _ = out.WriteString(":\n")

	for _, stmt := range n.Body {
		a.nodes[stmt].StringTo(out, a)
	}
}

func (n *MatchCase) String() string {
	var out strings.Builder
	n.StringTo(&out, nil)
	return out.String()
}
