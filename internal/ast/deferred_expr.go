package ast

import (
	"fmt"
	"strings"

	"github.com/samborkent/cog/internal/tokens"
	"github.com/samborkent/cog/internal/types"
)

var _ Expr = &DeferredExpr{}

// DeferredExpr is a placeholder expression stored during the globals pass when
// a {}-expression's expected type is an unresolved forward alias. It records
// the byte offset of the opening brace so the parser can seek back and parse
// the expression in the second pass, once all types are resolved.
type DeferredExpr struct {
	Token    tokens.Token
	Offset   uint32
	TypeHint types.Type // expected type context (forward alias that resolves lazily)
}

func (a *AST) NewDeferredExpr(token tokens.Token, offset uint32, typeHint types.Type) ExprIndex {
	de := New[DeferredExpr](a)
	de.Token = token
	de.Offset = offset
	de.TypeHint = typeHint
	return a.AddExpr(de)
}

func (e *DeferredExpr) Pos() (uint32, uint16) {
	return e.Token.Ln, e.Token.Col
}

func (e *DeferredExpr) Hash() uint64 {
	return hash(e)
}

func (e *DeferredExpr) Type() types.Type {
	return e.TypeHint
}

func (e *DeferredExpr) StringTo(out *strings.Builder, _ *AST) {
	fmt.Fprintf(out, "{expr@%d}", e.Offset)
}
