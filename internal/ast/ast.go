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

// Free releases the memory used by arenas in the merged AST.
// Shared arenas (from a pool) are freed only once.
func (a MergedAST) Free() {
	seen := make(map[*arena.Arena]struct{}, len(a))

	for _, ast := range a {
		if ast == nil {
			panic("HOW?")
		}

		if ast.arena == nil {
			continue
		}

		if _, ok := seen[ast.arena]; ok {
			continue
		}

		seen[ast.arena] = struct{}{}
		ast.arena.Free()
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

// ArenaTokenThreshold is the minimum token count (per worker) above which
// arena allocation becomes beneficial. With an arena pool sized by GOMAXPROCS,
// the 8MB chunk init cost is paid once per worker, not once per file.
// At ~30ns saved per node allocation, a single 8MB chunk (~5ms init) breaks
// even at roughly 170,000 node allocations, which corresponds to approximately
// 5,000 tokens (tokens/8 nodes + tokens/16 exprs ≈ tokens*3/16 allocations).
const ArenaTokenThreshold = 5_000

// AST is a single file AST that optionally uses arena-based memory management.
// When an arena is provided, all node allocations are performed on the arena,
// enabling batch deallocation and reducing GC pressure.
type AST struct {
	arena *arena.Arena

	fileIndex uint16
	nodes     []Node
	exprs     []Expr
	FileIndex NodeIndex // Index of the File node in nodes slice
}

// NewAST creates a new AST for a file. If a is non-nil, it is used for node
// allocations (arena mode); otherwise regular heap allocation is used.
// The caller manages the arena lifecycle — typically via a pool sized by
// GOMAXPROCS, with one arena per parser worker.
func NewAST(a *arena.Arena, fileIndex uint16, numBytes uint32) *AST {
	nodeCap := max(numBytes/4/8, 1)
	exprCap := max(numBytes/4/16, 1)

	return &AST{
		arena:     a,
		fileIndex: fileIndex,
		// Preallocate some memory for nodes and expressions to reduce the number of allocations.
		// First index needs to be nil.
		nodes: make([]Node, 1, nodeCap),
		exprs: make([]Expr, 1, exprCap),
	}
}

// Free releases the memory used by the arena in the AST.
// This is a no-op if the AST does not own an arena (arena managed externally).
// For multi-file programs with shared arenas, use [MergedAST.Free] instead.
func (a *AST) Free() {
	if a.arena != nil {
		a.arena.Free()
	}
}

// New creates a new node or expression in the AST and returns a pointer to it.
// Uses arena allocation when available, otherwise falls back to heap.
func New[T any](a *AST) *T {
	if a.arena != nil {
		return arena.New[T](a.arena)
	}

	return new(T)
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
