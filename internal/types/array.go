package types

type Array struct {
	Element Type
	Length  Expression
}

func (a *Array) Kind() Kind {
	return ArrayKind
}

func (a *Array) String() string {
	return "[" + a.Length.String() + "]" + a.Element.String()
}

func (a *Array) Underlying() Type {
	return a
}
