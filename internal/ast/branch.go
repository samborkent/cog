package ast

import (
	"strings"

	"github.com/samborkent/cog/internal/tokens"
)

var _ Statement = &Branch{}

type Branch struct {
	statement

	Token tokens.Token // break or continue token
	Label *Identifier
}

func (b *Branch) Pos() (uint32, uint16) {
	return b.Token.Ln, b.Token.Col
}

func (b *Branch) Hash() uint64 {
	return hash(b)
}

func (b *Branch) stringTo(out *strings.Builder) {
	_, _ = out.WriteString(b.Token.Type.String())

	if b.Label != nil {
		_ = out.WriteByte(' ')
		_, _ = out.WriteString(b.Label.Name)
	}
}

func (b *Branch) String() string {
	var out strings.Builder
	b.stringTo(&out)

	return out.String()
}
