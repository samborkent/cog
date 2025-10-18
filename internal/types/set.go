package types

var _ Type = &Set{}

type Set struct {
	Element Type
}

func (s *Set) Kind() Kind {
	return SetKind
}

func (s *Set) String() string {
	return "set[" + s.Element.String() + "]"
}

func (s *Set) Underlying() Type {
	return s
}
