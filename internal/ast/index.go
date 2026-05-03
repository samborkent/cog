package ast

import (
	"strings"

	"github.com/samborkent/cog/internal/tokens"
	"github.com/samborkent/cog/internal/types"
)

var _ Expr = &Index{}

type Index struct {
	Token    tokens.Token
	ElemType types.Type
	Expr     ExprIndex
	Index    ExprIndex
}

func (e *Index) Pos() (ln uint32, col uint16) {
	return e.Token.Ln, e.Token.Col
}

func (e *Index) Hash() uint64 {
	return hash(e)
}

func (e *Index) StringTo(out *strings.Builder, a *AST) {
	a.exprs[e.Expr].StringTo(out, a)
	_ = out.WriteByte('[')
	a.exprs[e.Index].StringTo(out, a)
	_ = out.WriteByte(']')
}

func (e *Index) Type() types.Type {
	return e.ElemType
}
