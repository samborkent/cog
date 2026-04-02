package types

type Alias struct {
	Name       string
	Derived    Type
	Exported   bool
	Global     bool
	TypeParams []*TypeParam
	lazy       func() Type
}

// NewForwardAlias creates an alias for a type that hasn't been fully resolved yet.
// The resolver function is called lazily when the type is first accessed.
func NewForwardAlias(name string, exported, global bool, resolver func() Type) *Alias {
	return &Alias{
		Name:     name,
		Derived:  None,
		Exported: exported,
		Global:   global,
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

// Instantiate substitutes TypeParam references in derived with concrete types.
// typeArgs maps type parameter names to their concrete replacements.
func (a *Alias) Instantiate(typeArgs map[string]Type) Type {
	a.ensureResolved()
	return substituteType(a.Derived, typeArgs)
}

// substituteType recursively replaces TypeParam references with concrete types.
func substituteType(t Type, args map[string]Type) Type {
	switch v := t.(type) {
	case *TypeParam:
		if concrete, ok := args[v.Name]; ok {
			return concrete
		}
		return v
	case *Alias:
		v.ensureResolved()
		return substituteType(v.Derived, args)
	case *Slice:
		return &Slice{Element: substituteType(v.Element, args)}
	case *Array:
		return &Array{Element: substituteType(v.Element, args), Length: v.Length}
	case *Map:
		return &Map{Key: substituteType(v.Key, args), Value: substituteType(v.Value, args)}
	case *Set:
		return &Set{Element: substituteType(v.Element, args)}
	case *Option:
		return &Option{Value: substituteType(v.Value, args)}
	case *Pointer:
		return &Pointer{Value: substituteType(v.Value, args)}
	case *Tuple:
		types := make([]Type, len(v.Types))
		for i, elem := range v.Types {
			types[i] = substituteType(elem, args)
		}
		return &Tuple{Types: types, Exported: v.Exported}
	case *Union:
		return &Union{
			Either:   substituteType(v.Either, args),
			Or:       substituteType(v.Or, args),
			Exported: v.Exported,
		}
	case *Result:
		return &Result{
			Value: substituteType(v.Value, args),
			Error: substituteType(v.Error, args),
		}
	case *Struct:
		fields := make([]*Field, len(v.Fields))
		for i, f := range v.Fields {
			fields[i] = &Field{
				Name:     f.Name,
				Type:     substituteType(f.Type, args),
				Exported: f.Exported,
			}
		}
		return &Struct{Fields: fields}
	default:
		// Basic types and other concrete types pass through unchanged.
		return t
	}
}
