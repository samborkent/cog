package ast

import (
	"strings"

	"github.com/samborkent/cog/internal/tokens"
	"github.com/samborkent/cog/internal/types"
)

var _ Expr = &Call{}

type Call struct {
	Token      tokens.Token
	Expr       ExprIndex // *Identfier or *Selector
	Arguments  []ExprIndex
	ReturnType types.Type
	TypeArgs   []types.Type // explicit or inferred type arguments for generic calls
}

func (a *AST) NewCall(t tokens.Token, expr ExprIndex, args []ExprIndex, returnType types.Type, typeArgs ...types.Type) ExprIndex {
	call := New[Call](a)
	call.Token = t
	call.Expr = expr
	call.Arguments = args
	call.ReturnType = returnType
	call.TypeArgs = typeArgs
	return a.AddExpr(call)
}

func (e *Call) Pos() (uint32, uint16) {
	return e.Token.Ln, e.Token.Col
}

func (e *Call) Hash() uint64 {
	return hash(e)
}

func (e *Call) StringTo(out *strings.Builder, a *AST) {
	a.exprs[e.Expr].StringTo(out, a)

	if len(e.TypeArgs) > 0 {
		_ = out.WriteByte('<')

		for i, ta := range e.TypeArgs {
			if i > 0 {
				_, _ = out.WriteString(", ")
			}

			_, _ = out.WriteString(ta.String())
		}

		_ = out.WriteByte('>')
	}

	_ = out.WriteByte('(')

	for i, arg := range e.Arguments {
		a.exprs[arg].StringTo(out, a)

		if i < len(e.Arguments)-1 {
			_, _ = out.WriteString(", ")
		}
	}

	_ = out.WriteByte(')')
}

func (e *Call) String() string {
	var out strings.Builder
	e.StringTo(&out, nil)
	return out.String()
}

func (e *Call) Type() types.Type {
	return e.ReturnType
}
