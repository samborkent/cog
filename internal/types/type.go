package types

type Type interface {
	Kind() Kind
	String() string
	Underlying() Type
}

type Expression struct {
	Expr   expression
	String string
}

type expression interface {
	Type() Type
}
