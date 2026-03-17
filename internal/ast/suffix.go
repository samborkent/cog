package ast

import (
	"strings"

	"github.com/samborkent/cog/internal/tokens"
	"github.com/samborkent/cog/internal/types"
)

var _ Expression = &Suffix{}

type Suffix struct {
	expression

	Operator tokens.Token
	Left     Expression
}

func (p *Suffix) Pos() (uint32, uint16) {
	return p.Operator.Ln, p.Operator.Col
}

func (p *Suffix) Hash() uint64 {
	return hash(p)
}

func (p *Suffix) stringTo(out *strings.Builder) {
	_ = out.WriteByte('(')
	p.Left.stringTo(out)
	_, _ = out.WriteString(p.Operator.Type.String())
	_ = out.WriteByte(')')
}

func (p *Suffix) String() string {
	var out strings.Builder
	p.stringTo(&out)
	return out.String()
}

func (p *Suffix) Type() types.Type {
	if p.Left.Type() == nil {
		panic("suffix with nil type detected")
	}

	return p.Left.Type()
}
