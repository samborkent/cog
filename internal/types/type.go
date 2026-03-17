package types

type Type interface {
	Kind() Kind
	String() string
	Underlying() Type
}

type expression interface {
	String() string
	Type() Type
}
