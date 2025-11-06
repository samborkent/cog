package ast

import (
	"strings"

	"github.com/samborkent/cog/internal/tokens"
	"github.com/samborkent/cog/internal/types"
)

var _ Expression = &MapLiteral{}

type MapLiteral struct {
	expression

	Token     tokens.Token
	KeyType   types.Type
	ValueType types.Type
	Pairs     []*KeyValue
}

type KeyValue struct {
	Key   Expression
	Value Expression
}

func (l *MapLiteral) Pos() (uint32, uint16) {
	return l.Token.Ln, l.Token.Col
}

func (l *MapLiteral) Hash() uint64 {
	return hash(l)
}

func (l *MapLiteral) String() string {
	var out strings.Builder

	_, _ = out.WriteString("({")

	for i, pair := range l.Pairs {
		_, _ = out.WriteString(pair.Key.String())
		_, _ = out.WriteString(": ")
		_, _ = out.WriteString(pair.Value.String())

		if i < len(l.Pairs)-1 {
			_, _ = out.WriteString(", ")
		}
	}

	_, _ = out.WriteString("} : ")
	_, _ = out.WriteString(l.Type().String())
	_ = out.WriteByte(')')

	return out.String()
}

func (l *MapLiteral) Type() types.Type {
	if l.ValueType == nil {
		panic("map with nil value type detected")
	}

	return &types.Map{
		Key:   l.KeyType,
		Value: l.ValueType,
	}
}
