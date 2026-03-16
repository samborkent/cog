package types

var _ Type = &Set{}

type Pointer struct {
	Value Type
}

func (s *Pointer) Kind() Kind {
	return PointerKind
}

func (s *Pointer) String() string {
	return "&" + s.Value.String()
}

func (s *Pointer) Underlying() Type {
	return s
}
