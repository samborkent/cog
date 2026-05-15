package ast

import (
	"fmt"
	"strings"

	"github.com/samborkent/cog/internal/tokens"
)

var _ Node = &DeferredBody{}

// DeferredBody is a placeholder node stored in place of a Block during the
// globals pass. It records the byte offset of the opening brace so the parser
// can seek back and parse the body in a second pass.
type DeferredBody struct {
	Token  tokens.Token
	Offset uint32
}

func (a *AST) NewDeferredBody(token tokens.Token, offset uint32) NodeIndex {
	db := New[DeferredBody](a)
	db.Token = token
	db.Offset = offset
	return a.AddNode(db)
}

func (n *DeferredBody) Pos() (uint32, uint16) {
	return n.Token.Ln, n.Token.Col
}

func (n *DeferredBody) Hash() uint64 {
	return hash(n)
}

func (n *DeferredBody) StringTo(out *strings.Builder, _ *AST) {
	fmt.Fprintf(out, "{deferred@%d}", n.Offset)
}
