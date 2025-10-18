package types

var _ Type = &Enum{}

type Enum struct {
	Value Type
}

func (*Enum) Kind() Kind {
	return EnumKind
}

func (e *Enum) String() string {
	return "enum[" + e.Value.String() + "]"
}

func (e *Enum) Underlying() Type {
	return e
}
