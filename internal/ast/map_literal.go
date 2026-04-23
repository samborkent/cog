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
	Pairs   []*KeyValue
}

type KeyValue struct {
	Key   ExprValue
	Value ExprValue
}

func (l *MapLiteral) Kind() NodeKind {
	return KindMapLiteral
}

func (l *MapLiteral) Pos() (uint32, uint16) {
	return l.Token.Ln, l.Token.Col
}

func (l *MapLiteral) Hash() uint64 {
	return hash(l)
}

func (l *MapLiteral) stringTo(out *strings.Builder) {
	_, _ = out.WriteString("({")

	for i, pair := range l.Pairs {
		pair.Key.expr.stringTo(out)
		_, _ = out.WriteString(": ")
		pair.Value.expr.stringTo(out)

		if i < len(l.Pairs)-1 {
			_, _ = out.WriteString(", ")
		}
	}

	_, _ = out.WriteString("} : ")
	_, _ = out.WriteString(l.Type().String())
	_ = out.WriteByte(')')
}

func (l *MapLiteral) String() string {
	var out strings.Builder
	l.stringTo(&out)

	return out.String()
}

func (l *MapLiteral) Type() types.Type {
	if l.MapType == nil {
		panic("map with nil value type detected")
	}

	if l.MapType.Kind() != types.MapKind {
		panic("map literal with non-map type detected")
	}

	return l.MapType
}
