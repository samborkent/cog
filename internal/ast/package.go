package ast

import (
	"strings"

	"github.com/samborkent/cog/internal/tokens"
)

var _ Node = &Package{}

type Package struct {
	Token      tokens.Token
	Identifier *Identifier
}

func (n *Package) Pos() (uint32, uint16) {
	return n.Token.Ln, n.Token.Col
}

func (n *Package) Hash() uint64 {
	return hash(n)
}

func (n *Package) StringTo(out *strings.Builder, _ *AST) {
	_, _ = out.WriteString(n.Token.Type.String())
	_ = out.WriteByte(' ')
	_, _ = out.WriteString(n.Identifier.Name)
}

func (n Package) String() string {
	var out strings.Builder
	n.StringTo(&out, nil)
	return out.String()
}
