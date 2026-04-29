package ast

import (
	"strings"

	"github.com/samborkent/cog/internal/tokens"
	"github.com/samborkent/cog/internal/types"
)

type utf8 = string

var _ Expr = &UTF8Literal{}

type UTF8Literal struct {
	Token tokens.Token
	Value utf8
}

func (a *AST) NewUTF8Literal(t tokens.Token) ExprIndex {
	expr := New[UTF8Literal](a)
	expr.Token = t
	expr.Value = t.Literal
	return a.AddExpr(expr)
}

func (l *UTF8Literal) Pos() (uint32, uint16) {
	return l.Token.Ln, l.Token.Col
}

func (l *UTF8Literal) Hash() uint64 {
	return hash(l)
}

func (l *UTF8Literal) StringTo(out *strings.Builder, _ *AST) {
	if strings.ContainsAny(l.Value, "\n\t") {
		_, _ = out.WriteString("(`")
		_, _ = out.WriteString(l.Value)
		_, _ = out.WriteString("` : utf8)")

		return
	}

	_, _ = out.WriteString("(\"")
	_, _ = out.WriteString(l.Value)
	_, _ = out.WriteString("\" : utf8)")
}

func (l *UTF8Literal) String() string {
	var out strings.Builder
	l.StringTo(&out, nil)
	return out.String()
}

func (l *UTF8Literal) Type() types.Type {
	return types.Basics[types.UTF8]
}
