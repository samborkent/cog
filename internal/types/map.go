package types

var _ Type = &Map{}

type Map struct {
	Key, Value Type
}

func (s *Map) Kind() Kind {
	return MapKind
}

func (s *Map) String() string {
	return "map[" + s.Key.String() + "]" + s.Value.String()
}

func (s *Map) Underlying() Type {
	return s
}
