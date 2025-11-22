package types

type Slice struct {
	Element Type
}

func (s *Slice) Kind() Kind {
	return SliceKind
}

func (s *Slice) String() string {
	return "[]" + s.Element.String()
}

func (s *Slice) Underlying() Type {
	return s
}
