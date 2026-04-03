package types

// Any is the singleton representing the any type (accepts all types).
var Any = &anyType{}

var _ Type = &anyType{}

type anyType struct{}

func (a *anyType) Kind() Kind {
	return AnyKind
}

func (a *anyType) String() string {
	return "any"
}

func (a *anyType) Underlying() Type {
	return a
}
