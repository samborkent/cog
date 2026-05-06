package ast

import (
	"strings"

	"github.com/samborkent/cog/internal/types"
)

type Node interface {
	Pos() (ln uint32, col uint16)
	// String() string
	Hash() uint64
	StringTo(out *strings.Builder, a *AST)
}

type Expr interface {
	Node
	Type() types.Type
}

type (
	NodeIndex uint32
	ExprIndex uint32
)

var (
	ZeroNodeIndex = NodeIndex(0)
	ZeroExprIndex = ExprIndex(0)
)
