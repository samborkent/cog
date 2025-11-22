package ast

import (
	"strings"

	"github.com/samborkent/cog/internal/tokens"
	"github.com/samborkent/cog/internal/types"
)

var _ Expression = &Index{}

type Index struct {
	expression

	Token      tokens.Token
	Identifier Expression
	Index      Expression
}

func (e *Index) Pos() (ln uint32, col uint16) {
	return e.Token.Ln, e.Token.Col
}

func (e *Index) Hash() uint64 {
	return hash(e)
}

func (e *Index) String() string {
	var out strings.Builder

	_, _ = out.WriteString(e.Identifier.String())
	_ = out.WriteByte('[')
	_, _ = out.WriteString(e.Index.String())
	_ = out.WriteByte(']')

	return out.String()
}

func (e *Index) Type() types.Type {
	switch t := e.Identifier.Type().(type) {
	case *types.Array:
		return t.Element
	case *types.Map:
		return t.Value
	case *types.Set:
		return t.Element
	case *types.Slice:
		return t.Element
	default:
		panic("indexing non-indexable type")
	}
}
