package ast

import (
	"strings"

	"github.com/samborkent/cog/internal/tokens"
	"github.com/samborkent/cog/internal/types"
)

var _ Expr = &Selector{}

type Selector struct {
	Token  tokens.Token
	Fields []*Identifier
}

func (a *AST) NewSelector(t tokens.Token, fields ...*Identifier) ExprIndex {
	expr := New[Selector](a)
	expr.Token = t
	expr.Fields = fields
	return a.AddExpr(expr)
}

func (e *Selector) Pos() (uint32, uint16) {
	return e.Token.Ln, e.Token.Col
}

func (e *Selector) Hash() uint64 {
	return hash(e)
}

func (e *Selector) StringTo(out *strings.Builder, _ *AST) {
	for i := range e.Fields {
		_, _ = out.WriteString(e.Fields[i].Name)

		if i < len(e.Fields)-1 {
			_ = out.WriteByte('.')
		}
	}
}

func (e *Selector) String() string {
	var out strings.Builder
	e.StringTo(&out, nil)
	return out.String()
}

func (e *Selector) Type() types.Type {
	return e.Fields[len(e.Fields)-1].ValueType
}
