package ast

import (
	"strings"

	"github.com/samborkent/cog/internal/tokens"
)

var _ Node = &Case{}

type Case struct {
	Token     tokens.Token
	Condition ExprIndex
	Body      []NodeIndex
}

func (n *Case) Pos() (ln uint32, col uint16) {
	return n.Token.Ln, n.Token.Col
}

func (n *Case) Hash() uint64 {
	return hash(n)
}

func (n *Case) StringTo(out *strings.Builder, a *AST) {
	_, _ = out.WriteString(n.Token.Type.String())
	_ = out.WriteByte(' ')
	a.exprs[n.Condition].StringTo(out, a)
	_, _ = out.WriteString(":\n")

	for _, stmt := range n.Body {
		_ = out.WriteByte('\t')
		a.nodes[stmt].StringTo(out, a)
		_ = out.WriteByte('\n')
	}
}

func (n *Case) String() string {
	var out strings.Builder
	n.StringTo(&out, nil)
	return out.String()
}
