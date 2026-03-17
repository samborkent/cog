package ast

import (
	"strings"

	"github.com/samborkent/cog/internal/tokens"
)

var _ Statement = &Package{}

type Package struct {
	statement

	Token      tokens.Token
	Identifier *Identifier
}

func (p *Package) Pos() (uint32, uint16) {
	return p.Token.Ln, p.Token.Col
}

func (p *Package) Hash() uint64 {
	return hash(p)
}

func (p *Package) stringTo(out *strings.Builder) {
	_, _ = out.WriteString(p.Token.Type.String())
	_ = out.WriteByte(' ')
	p.Identifier.stringTo(out)
}

func (p *Package) String() string {
	var out strings.Builder
	p.stringTo(&out)
	return out.String()
}
