package ast

import (
	"strings"

	"github.com/samborkent/cog/internal/tokens"
	"github.com/samborkent/cog/internal/types"
)

var _ Expr = &Builtin{}

type Builtin struct {
	Token         tokens.Token
	Name          string
	TypeArguments []types.Type
	Arguments     []ExprIndex
	ReturnType    types.Type
}

func (a *AST) NewBuiltin(token tokens.Token, name string, typeArgs []types.Type, args []ExprIndex, returnType types.Type) ExprIndex {
	expr := New[Builtin](a)
	expr.Token = token
	expr.Name = name
	expr.TypeArguments = typeArgs
	expr.Arguments = args
	expr.ReturnType = returnType
	return a.AddExpr(expr)
}

func (e *Builtin) Pos() (uint32, uint16) {
	return e.Token.Ln, e.Token.Col
}

func (e *Builtin) Hash() uint64 {
	return hash(e)
}

func (e *Builtin) StringTo(out *strings.Builder, a *AST) {
	_ = out.WriteByte('@')
	_, _ = out.WriteString(e.Name)

	for i, arg := range e.TypeArguments {
		if i == 0 {
			_ = out.WriteByte('<')
		}

		_, _ = out.WriteString(arg.String())

		if i < len(e.TypeArguments)-1 {
			_, _ = out.WriteString(", ")
		} else {
			_ = out.WriteByte('>')
		}
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

func (e *Builtin) Type() types.Type {
	return e.ReturnType
}
