package types

type Alias struct {
	Name     string
	Derived  Type
	Exported bool
}

func (a *Alias) Kind() Kind {
	derived := a.Derived

	for derived.Underlying() != derived {
		derived = derived.Underlying()
	}

	return derived.Kind()
}

func (a *Alias) String() string {
	return a.Name
}

func (a *Alias) Underlying() Type {
	alias, ok := a.Derived.(*Alias)
	if ok {
		return alias.Underlying()
	}

	return a.Derived
}
