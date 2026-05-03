package ast

import (
	"arena"
	"iter"
)

// MergedAST is a structure that combines multiple ASTs into one.
// The parser will create one AST per file, and then we will merge them together to pass it to the transpiler.
type MergedAST []*AST

// MergeASTs takes a slice of ASTs and merges them into a single merged AST.
func MergeASTs(asts ...*AST) MergedAST {
	return asts
}

// Free releases the memory used by all arenas in the merged AST.
// This should be called after the transpilation is done to free up memory.
func (a MergedAST) Free() {
	for _, ast := range a {
		ast.Free()
	}
}

func (a MergedAST) Node(fileIndex uint16, nodeIndex NodeIndex) Node {
	return a[fileIndex].Node(nodeIndex)
}

func (a MergedAST) AllNodes() iter.Seq2[uint16, []Node] {
	return func(yield func(uint16, []Node) bool) {
		for i, ast := range a {
			if !yield(uint16(i), ast.nodes) {
				return
			}
		}
	}
}

func (a MergedAST) Expr(fileIndex uint16, exprIndex ExprIndex) Expr {
	return a[fileIndex].Expr(exprIndex)
}

// AST is a single file AST. That uses arena based memory mangement.
type AST struct {
	arena *arena.Arena

	fileIndex uint16
	nodes     []Node
	exprs     []Expr
}

func NewAST(fileIndex uint16, preallocationSize uint32) *AST {
	return &AST{
		arena:     arena.NewArena(),
		fileIndex: fileIndex,
		// Preallocate some memory for nodes and expressions to reduce the number of allocations.
		// First index needs to be nil.
		nodes: make([]Node, 1, int(preallocationSize)),
		exprs: make([]Expr, 1, int(preallocationSize)),
	}
}

// Free releases the memory used by the arena in the AST.
// This should be called after the transpilation is done to free up memory.
// This should only be used for single-file programs. For multi-file programs, use [MergedAST.Free] instead.
func (a *AST) Free() {
	a.arena.Free()
}

// New creates a new node or expression in the AST and returns a pointer to it.
func New[T any](a *AST) *T {
	return arena.New[T](a.arena)
}

// AddNode adds a node to the AST and returns its index.
func (a *AST) AddNode(node Node) NodeIndex {
	a.nodes = append(a.nodes, node)
	return NodeIndex(len(a.nodes) - 1)
}

// Node returns the node at the given index.
func (a *AST) Node(i NodeIndex) Node {
	return a.nodes[i]
}

// LenNodes returns the number of nodes in the AST.
func (a *AST) LenNodes() int {
	return len(a.nodes) - 1
}

// AddExpr adds an expression to the AST and returns its index.
func (a *AST) AddExpr(expr Expr) ExprIndex {
	a.exprs = append(a.exprs, expr)
	return ExprIndex(len(a.exprs) - 1)
}

// Expr returns the expression at the given index.
func (a *AST) Expr(i ExprIndex) Expr {
	return a.exprs[i]
}

// LenExprs returns the number of expressions in the AST.
func (a *AST) LenExprs() int {
	return len(a.exprs) - 1
}
