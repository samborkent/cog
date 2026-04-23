package ast

import (
	"strings"
	"testing"
)

type Node interface {
	Kind() NodeKind
	Hash() uint64
	Pos() (ln uint32, col uint16)
	String() string
	stringTo(out *strings.Builder)
}

type NodeValue struct {
	NodeKind NodeKind
	node     Node
}

func NewNode[T Node](kind NodeKind, node T) NodeValue {
	return NodeValue{
		NodeKind: kind,
		node:     node,
	}
}

var ZeroNode NodeValue

func (v NodeValue) Node(t *testing.T) Node {
	t.Helper()
	return v.node
}

func (v NodeValue) AsDeclaration() *Declaration {
	return v.node.(*Declaration)
}

func (v NodeValue) AsExpressionStatement() *ExpressionStatement {
	return v.node.(*ExpressionStatement)
}

func (v NodeValue) AsForStatement() *ForStatement {
	return v.node.(*ForStatement)
}

func (v NodeValue) AsIfStatement() *IfStatement {
	return v.node.(*IfStatement)
}

func (v NodeValue) AsMatch() *Match {
	return v.node.(*Match)
}

func (v NodeValue) AsReturn() *Return {
	return v.node.(*Return)
}
