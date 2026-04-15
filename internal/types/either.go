package types

type Either struct {
	Left, Right      Type
	Exported, Global bool
}

func (t *Either) Kind() Kind {
	return EitherKind
}

func (t *Either) String() string {
	return t.Left.String() + " ^ " + t.Right.String()
}

func (t *Either) Underlying() Type {
	return t
}
