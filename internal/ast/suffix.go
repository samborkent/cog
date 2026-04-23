package ast

import (
	"strings"

	"github.com/samborkent/cog/internal/tokens"
	"github.com/samborkent/cog/internal/types"
)

var _ Expr = &Suffix{}

type Suffix struct {
	Operator tokens.Token
	Left     ExprValue
}

func (p *Suffix) Kind() NodeKind {
	return KindSuffix
}

func (p *Suffix) Pos() (uint32, uint16) {
	return p.Operator.Ln, p.Operator.Col
}

func (p *Suffix) Hash() uint64 {
	return hash(p)
}

func (p *Suffix) stringTo(out *strings.Builder) {
	_ = out.WriteByte('(')
	p.Left.expr.stringTo(out)
	_, _ = out.WriteString(p.Operator.Type.String())
	_ = out.WriteByte(')')
}

func (p *Suffix) String() string {
	var out strings.Builder
	p.stringTo(&out)

	return out.String()
}

func (p *Suffix) Type() types.Type {
	// ? suffix is always a boolean check (option: is set?, result: is OK?).
	if p.Operator.Type == tokens.Question {
		return types.Basics[types.Bool]
	}

	// ! suffix extracts the error value from a result type.
	if p.Operator.Type == tokens.Not {
		underlying := p.Left.expr.Type()
		if alias, ok := underlying.(*types.Alias); ok {
			underlying = alias.Underlying()
		}

		if result, ok := underlying.(*types.Result); ok {
			return result.Error
		}

		panic("! suffix on non-result type")
	}

	return p.Left.expr.Type()
}
