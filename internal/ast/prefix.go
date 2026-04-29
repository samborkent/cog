package ast

import (
	"strings"

	"github.com/samborkent/cog/internal/tokens"
	"github.com/samborkent/cog/internal/types"
)

var _ Expr = &Prefix{}

type Prefix struct {
	Operator   tokens.Token
	PrefixType types.Type
	Right      ExprIndex
}

func (a *AST) NewPrefix(operator tokens.Token, prefixType types.Type, right ExprIndex) ExprIndex {
	prefixExpr := New[Prefix](a)
	prefixExpr.Operator = operator
	prefixExpr.PrefixType = prefixType
	prefixExpr.Right = right
	return a.AddExpr(prefixExpr)
}

func (e *Prefix) Pos() (uint32, uint16) {
	return e.Operator.Ln, e.Operator.Col
}

func (e *Prefix) Hash() uint64 {
	return hash(e)
}

func (e *Prefix) StringTo(out *strings.Builder, a *AST) {
	_ = out.WriteByte('(')
	_, _ = out.WriteString(e.Operator.Type.String())
	a.exprs[e.Right].StringTo(out, a)
	_ = out.WriteByte(')')
}

func (e *Prefix) String() string {
	var out strings.Builder
	e.StringTo(&out, nil)
	return out.String()
}

func (e *Prefix) Type() types.Type {
	// Return
	if e.Operator.Type == tokens.BitAnd {
		return &types.Reference{
			Value: e.PrefixType,
		}
	}

	return e.PrefixType
}
