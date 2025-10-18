package ast

import (
	"strings"

	"github.com/samborkent/cog/internal/tokens"
)

var _ Statement = &Return{}

type Return struct {
	statement
	Token  tokens.Token
	Values []Expression
}

func (r *Return) Pos() (uint32, uint16) {
	return r.Token.Ln, r.Token.Col
}

func (r *Return) Hash() uint64 {
	return hash(r)
}

func (r *Return) String() string {
	values := make([]string, 0, len(r.Values))

	for _, v := range r.Values {
		values = append(values, v.String())
	}

	return r.Token.Type.String() + " " + strings.Join(values, ", ")
}
