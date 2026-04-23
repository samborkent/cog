package ast

import (
	"strings"

	"github.com/samborkent/cog/internal/tokens"
	"github.com/samborkent/cog/internal/types"
)

var _ Expr = &Index{}

type Index struct {
	Token      tokens.Token
	Identifier ExprValue
	Index      ExprValue
}

func (e *Index) Kind() NodeKind {
	return KindIndex
}

func (e *Index) Pos() (ln uint32, col uint16) {
	return e.Token.Ln, e.Token.Col
}

func (e *Index) Hash() uint64 {
	return hash(e)
}

func (e *Index) stringTo(out *strings.Builder) {
	e.Identifier.expr.stringTo(out)
	_ = out.WriteByte('[')
	e.Index.expr.stringTo(out)
	_ = out.WriteByte(']')
}

func (e *Index) String() string {
	var out strings.Builder
	e.stringTo(&out)

	return out.String()
}

func (e *Index) Type() types.Type {
	switch e.Identifier.TypeKind {
	case types.ArrayKind:
		return e.Identifier.expr.Type().(*types.Array).Element
	case types.MapKind:
		return e.Identifier.expr.Type().(*types.Map).Value
	case types.SetKind:
		return e.Identifier.expr.Type().(*types.Set).Element
	case types.SliceKind:
		return e.Identifier.expr.Type().(*types.Slice).Element
	default:
		panic("indexing non-indexable type")
	}
}
