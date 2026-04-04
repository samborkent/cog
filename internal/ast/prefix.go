package ast

import (
	"strings"

	"github.com/samborkent/cog/internal/tokens"
	"github.com/samborkent/cog/internal/types"
)

var _ Expression = &Prefix{}

type Prefix struct {
	expression

	Operator tokens.Token
	Right    Expression
}

func (p *Prefix) Pos() (uint32, uint16) {
	return p.Operator.Ln, p.Operator.Col
}

func (p *Prefix) Hash() uint64 {
	return hash(p)
}

func (p *Prefix) stringTo(out *strings.Builder) {
	_ = out.WriteByte('(')
	_, _ = out.WriteString(p.Operator.Type.String())
	p.Right.stringTo(out)
	_ = out.WriteByte(')')
}

func (p *Prefix) String() string {
	var out strings.Builder
	p.stringTo(&out)

	return out.String()
}

func (p *Prefix) Type() types.Type {
	if p.Right.Type() == nil {
		panic("prefix with nil type detected")
	}

	// Return
	if p.Operator.Type == tokens.BitAnd {
		return &types.Reference{
			Value: p.Right.Type(),
		}
	}

	return p.Right.Type()
}
