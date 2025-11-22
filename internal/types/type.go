package types

type Type interface {
	Kind() Kind
	String() string
	Underlying() Type
}

type Expression interface {
	String() string
	Type() Type
}
