package ast

import (
	"strings"

	"github.com/samborkent/cog/internal/tokens"
)

var _ Node = &Assignment{}

type Assignment struct {
	Token      tokens.Token
	Identifier *Identifier
	Expr       ExprValue
}

func (a *Assignment) Pos() (uint32, uint16) {
	return a.Token.Ln, a.Token.Col
}

func (a *Assignment) Hash() uint64 {
	return hash(a)
}

func (a *Assignment) stringTo(out *strings.Builder) {
	a.Identifier.stringTo(out)
	_ = out.WriteByte(' ')
	_, _ = out.WriteString(a.Token.Type.String())
	_ = out.WriteByte(' ')
	a.Expr.expr.stringTo(out)
}

func (a *Assignment) String() string {
	var out strings.Builder
	a.stringTo(&out)

	return out.String()
}
