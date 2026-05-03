package ast

import (
	"strings"

	"github.com/samborkent/cog/internal/tokens"
)

var _ Node = &Default{}

type Default struct {
	Token tokens.Token
	Body  []NodeIndex
}

func (n *Default) Pos() (ln uint32, col uint16) {
	return n.Token.Ln, n.Token.Col
}

func (n *Default) Hash() uint64 {
	return hash(n)
}

func (n *Default) StringTo(out *strings.Builder, a *AST) {
	_, _ = out.WriteString(n.Token.Type.String())
	_, _ = out.WriteString(":\n")

	for _, stmt := range n.Body {
		_ = out.WriteByte('\t')
		a.nodes[stmt].StringTo(out, a)
		_ = out.WriteByte('\n')
	}
}
