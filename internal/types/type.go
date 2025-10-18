package types

type Type interface {
	Kind() Kind
	String() string
	Underlying() Type
}
