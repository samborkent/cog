package ast

import (
	"strings"

	"github.com/samborkent/cog/internal/tokens"
	"github.com/samborkent/cog/internal/types"
)

var _ Expr = &MapLiteral{}

type MapLiteral struct {
	Token   tokens.Token
	MapType types.Type
	Pairs   []KeyValue
}

type KeyValue struct {
	Key   ExprIndex
	Value ExprIndex
}

func (a *AST) NewMapLiteral(token tokens.Token, t *types.Map, pairs []KeyValue) ExprIndex {
	mapLiteral := New[MapLiteral](a)
	mapLiteral.Token = token
	mapLiteral.MapType = t
	mapLiteral.Pairs = pairs
	return a.AddExpr(mapLiteral)
}

func (l *MapLiteral) Pos() (uint32, uint16) {
	return l.Token.Ln, l.Token.Col
}

func (l *MapLiteral) Hash() uint64 {
	return hash(l)
}

func (l *MapLiteral) StringTo(out *strings.Builder, a *AST) {
	_, _ = out.WriteString("({")

	for i, pair := range l.Pairs {
		a.exprs[pair.Key].StringTo(out, a)
		_, _ = out.WriteString(": ")
		a.exprs[pair.Value].StringTo(out, a)

		if i < len(l.Pairs)-1 {
			_, _ = out.WriteString(", ")
		}
	}

	_, _ = out.WriteString("} : ")
	_, _ = out.WriteString(l.Type().String())
	_ = out.WriteByte(')')
}

func (l *MapLiteral) Type() types.Type {
	return l.MapType
}
