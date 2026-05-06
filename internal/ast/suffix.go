package ast

import (
	"strings"

	"github.com/samborkent/cog/internal/tokens"
	"github.com/samborkent/cog/internal/types"
)

var _ Expr = &Suffix{}

type Suffix struct {
	Operator   tokens.Token
	SuffixType types.Type
	Left       ExprIndex
}

func (a *AST) NewSuffix(operator tokens.Token, suffixType types.Type, left ExprIndex) ExprIndex {
	suffixExpr := New[Suffix](a)
	suffixExpr.Operator = operator
	suffixExpr.SuffixType = suffixType
	suffixExpr.Left = left
	return a.AddExpr(suffixExpr)
}

func (e *Suffix) Pos() (uint32, uint16) {
	return e.Operator.Ln, e.Operator.Col
}

func (e *Suffix) Hash() uint64 {
	return hash(e)
}

func (e *Suffix) StringTo(out *strings.Builder, a *AST) {
	_ = out.WriteByte('(')
	a.exprs[e.Left].StringTo(out, a)
	_, _ = out.WriteString(e.Operator.Type.String())
	_ = out.WriteByte(')')
}

func (e *Suffix) Type() types.Type {
	if e.Operator.Type == tokens.Not {
		return e.SuffixType.Underlying().(*types.Result).Error
	}

	if e.Operator.Type == tokens.Question {
		return types.Basics[types.Bool]
	}

	return e.SuffixType
}
