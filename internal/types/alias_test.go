package types

import "testing"

func TestNewForwardAlias(t *testing.T) {
	t.Parallel()

	resolved := false
	a := NewForwardAlias("Foo", true, true, func() Type {
		resolved = true
		return Basics[Int64]
	})

	if a.Name != "Foo" {
		t.Errorf("Name = %q, want Foo", a.Name)
	}

	if resolved {
		t.Fatal("resolver called before access")
	}

	// Accessing Kind triggers lazy resolution.
	if k := a.Kind(); k != Int64 {
		t.Errorf("Kind() = %v, want %v", k, Int64)
	}

	if !resolved {
		t.Error("resolver not called after Kind()")
	}
}

func TestAliasInstantiate(t *testing.T) {
	t.Parallel()

	// List<T ~ any> ~ []T
	tparam := &Alias{Name: "T", Constraint: Any}
	listDef := &Alias{
		Name:       "List",
		Derived:    &Slice{Element: tparam},
		TypeParams: []*Alias{tparam},
		Global:     true,
	}

	result := listDef.Instantiate(map[string]Type{"T": Basics[UTF8]})

	alias, ok := result.(*Alias)
	if !ok {
		t.Fatalf("Instantiate returned %T, want *Alias", result)
	}

	if alias.Name != "List" {
		t.Errorf("Name = %q, want List", alias.Name)
	}

	if len(alias.TypeArgs) != 1 {
		t.Fatalf("TypeArgs len = %d, want 1", len(alias.TypeArgs))
	}

	if alias.TypeArgs[0] != Basics[UTF8] {
		t.Errorf("TypeArgs[0] = %v, want utf8", alias.TypeArgs[0])
	}

	if alias.Kind() != SliceKind {
		t.Errorf("Kind() = %v, want SliceKind", alias.Kind())
	}

	// Derived should be the substituted slice.
	sl, ok := alias.Derived.(*Slice)
	if !ok {
		t.Fatalf("Derived = %T, want *Slice", alias.Derived)
	}

	if sl.Element != Basics[UTF8] {
		t.Errorf("Derived element = %v, want utf8", sl.Element)
	}
}

func TestInstantiatedAliasKindThroughForward(t *testing.T) {
	t.Parallel()

	// Simulate: List<T ~ any> ~ []T defined AFTER first use.
	tparam := &Alias{Name: "T", Constraint: Any}
	listDef := &Alias{
		Name:       "List",
		Derived:    &Slice{Element: tparam},
		TypeParams: []*Alias{tparam},
		Global:     true,
	}

	// Forward stub (lazy resolver returns ident.ValueType).
	var identValueType Type = None
	forward := NewForwardAlias("List", false, true, func() Type {
		return identValueType
	})

	// Instantiated alias created during globals pass before List is defined.
	instantiated := &Alias{
		Name:     "List",
		Derived:  forward,
		Global:   true,
		TypeArgs: []Type{Basics[UTF8]},
	}

	// Simulate DefineGlobal resolving the forward stub.
	identValueType = listDef

	if k := instantiated.Kind(); k != SliceKind {
		t.Errorf("Kind() = %v, want SliceKind", k)
	}
}

func TestAliasInstantiateMultipleArgs(t *testing.T) {
	t.Parallel()

	// Dict<K ~ comparable, V ~ any> ~ map<K, V>
	kparam := &Alias{Name: "K", Constraint: Constraints["comparable"]}
	vparam := &Alias{Name: "V", Constraint: Any}
	dictDef := &Alias{
		Name:       "Dict",
		Derived:    &Map{Key: kparam, Value: vparam},
		TypeParams: []*Alias{kparam, vparam},
		Global:     true,
	}

	result := dictDef.Instantiate(map[string]Type{
		"K": Basics[UTF8],
		"V": Basics[Int64],
	})

	alias, ok := result.(*Alias)
	if !ok {
		t.Fatalf("Instantiate returned %T, want *Alias", result)
	}

	if len(alias.TypeArgs) != 2 {
		t.Fatalf("TypeArgs len = %d, want 2", len(alias.TypeArgs))
	}

	if alias.TypeArgs[0] != Basics[UTF8] {
		t.Errorf("TypeArgs[0] = %v, want utf8", alias.TypeArgs[0])
	}

	if alias.TypeArgs[1] != Basics[Int64] {
		t.Errorf("TypeArgs[1] = %v, want int64", alias.TypeArgs[1])
	}

	if alias.Kind() != MapKind {
		t.Errorf("Kind() = %v, want MapKind", alias.Kind())
	}
}
