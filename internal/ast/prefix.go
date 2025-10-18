package ast

import (
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

func (p *Prefix) String() string {
	return "(" + p.Operator.Type.String() + p.Right.String() + ")"
}

func (p *Prefix) Type() types.Type {
	if p.Right.Type() == nil {
		panic("prefix with nil type detected")
	}

	return p.Right.Type()
}
