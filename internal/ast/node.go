package ast

import (
	"strings"

	"github.com/samborkent/cog/internal/types"
)

type NodeValue struct {
	Kind NodeKind
	node Node
}

type ExprValue struct {
	NodeKind NodeKind
	TypeKind types.Kind
	expr     Expr
}

type Node interface {
	Pos() (ln uint32, col uint16)
	Hash() uint64
	String() string
	stringTo(out *strings.Builder)
}

type Expr interface {
	Node
	Type() types.Type
}
