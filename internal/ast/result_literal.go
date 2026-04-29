package ast

import (
	"strings"

	"github.com/samborkent/cog/internal/tokens"
	"github.com/samborkent/cog/internal/types"
)

var _ Expr = &ResultLiteral{}

type ResultLiteral struct {
	Token      tokens.Token
	ResultType types.Type
	Value      ExprIndex
	IsError    bool // False: Value (success), True: Error
}

func (a *AST) NewResultLiteral(tok tokens.Token, resultType types.Type, value ExprIndex, isError bool) ExprIndex {
	expr := New[ResultLiteral](a)
	expr.Token = tok
	expr.ResultType = resultType
	expr.Value = value
	expr.IsError = isError
	return a.AddExpr(expr)
}

func (e *ResultLiteral) Pos() (uint32, uint16) {
	return e.Token.Ln, e.Token.Col
}

func (e *ResultLiteral) Hash() uint64 {
	return hash(e)
}

func (e *ResultLiteral) StringTo(out *strings.Builder, a *AST) {
	a.exprs[e.Value].StringTo(out, a)
}

func (e *ResultLiteral) String() string {
	var out strings.Builder
	e.StringTo(&out, nil)
	return out.String()
}

func (e *ResultLiteral) Type() types.Type {
	return e.ResultType
}
