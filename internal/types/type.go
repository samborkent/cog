package types

type Type interface {
	Kind() Kind
	String() string
	Underlying() Type
}

// ExprIndex is an index into the AST expression slice.
// Defined here so types can reference expressions without importing ast.
type ExprIndex uint32

// Expression references an AST expression by index. The String field caches the
// textual representation for use in type String() methods and equality checks.
type Expression struct {
	Index  ExprIndex
	String string
}
