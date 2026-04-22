package ast

import (
	"strings"

	"github.com/samborkent/cog/internal/tokens"
)

var _ Node = &Return{}

type Return struct {
	Token  tokens.Token
	Values []ExprValue
}

func (r *Return) Pos() (uint32, uint16) {
	return r.Token.Ln, r.Token.Col
}

func (r *Return) Hash() uint64 {
	return hash(r)
}

func (r *Return) stringTo(out *strings.Builder) {
	_, _ = out.WriteString(r.Token.Type.String())
	_ = out.WriteByte(' ')

	for i, v := range r.Values {
		v.expr.stringTo(out)

		if i < len(r.Values)-1 {
			_, _ = out.WriteString(", ")
		}
	}
}

func (r *Return) String() string {
	var out strings.Builder
	r.stringTo(&out)

	return out.String()
}
