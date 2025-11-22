package types

var _ Type = &Map{}

type Map struct {
	Key, Value Type
}

func (m *Map) Kind() Kind {
	return MapKind
}

func (m *Map) String() string {
	return "map[" + m.Key.String() + "]" + m.Value.String()
}

func (m *Map) Underlying() Type {
	return m
}
