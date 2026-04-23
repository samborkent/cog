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
	Value      ExprValue
	IsError    bool // False: Value (success), True: Error
}

func (e *ResultLiteral) Kind() NodeKind {
	return KindResultLiteral
}

func (e *ResultLiteral) Pos() (uint32, uint16) {
	return e.Token.Ln, e.Token.Col
}

func (e *ResultLiteral) Hash() uint64 {
	return hash(e)
}

func (e *ResultLiteral) stringTo(out *strings.Builder) {
	e.Value.expr.stringTo(out)
}

func (e *ResultLiteral) String() string {
	var out strings.Builder
	e.stringTo(&out)

	return out.String()
}

func (e *ResultLiteral) Type() types.Type {
	if e.ResultType == nil {
		panic("result with nil type detected")
	}

	return e.ResultType
}
