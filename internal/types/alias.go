package types

type Alias struct {
	Name     string
	Derived  Type
	Exported bool
	lazy     func() Type
}

// NewForwardAlias creates an alias for a type that hasn't been fully resolved yet.
// The resolver function is called lazily when the type is first accessed.
func NewForwardAlias(name string, exported bool, resolver func() Type) *Alias {
	return &Alias{
		Name:     name,
		Derived:  None,
		Exported: exported,
		lazy:     resolver,
	}
}

func (a *Alias) ensureResolved() {
	if a.lazy != nil && IsNone(a.Derived) {
		a.Derived = a.lazy()
		a.lazy = nil
	}
}

func (a *Alias) Kind() Kind {
	a.ensureResolved()

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
	a.ensureResolved()

	alias, ok := a.Derived.(*Alias)
	if ok {
		return alias.Underlying()
	}

	return a.Derived
}
