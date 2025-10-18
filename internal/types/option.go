package types

type Option struct {
	Value Type
}

func (t *Option) Kind() Kind {
	return OptionKind
}

func (t *Option) String() string {
	return t.Value.String() + "?"
}

func (t *Option) Underlying() Type {
	return t
}
