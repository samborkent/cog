package types

type Alias struct {
	Name       string
	Derived    Type
	Constraint Type // non-nil when this alias acts as a type parameter
	Exported   bool
	Global     bool
	TypeParams []*Alias
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
	if a.Constraint != nil {
		return GenericKind
	}

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
	if a.Constraint != nil {
		return a.Constraint.Underlying()
	}

	a.ensureResolved()

	alias, ok := a.Derived.(*Alias)
	if ok {
		return alias.Underlying()
	}

	return a.Derived
}

// IsTypeParam reports whether this alias acts as a type parameter.
func (a *Alias) IsTypeParam() bool {
	return a.Constraint != nil
}

// ConstraintString returns the constraint portion for display,
// e.g. "any", "int", or "string | int".
func (a *Alias) ConstraintString() string {
	if a.Constraint == nil {
		return ""
	}

	return a.Constraint.String()
}

// SatisfiedBy reports whether a concrete type satisfies this
// type parameter's constraints (OR semantics: satisfies at least one).
func (a *Alias) SatisfiedBy(concrete Type) bool {
	if a.Constraint == nil {
		return false
	}

	if union, ok := a.Constraint.(*Union); ok {
		for _, v := range union.Variants {
			if Satisfies(concrete, v) {
				return true
			}
		}

		return false
	}

	return Satisfies(concrete, a.Constraint)
}

// Instantiate substitutes TypeParam references in derived with concrete types.
// typeArgs maps type parameter names to their concrete replacements.
func (a *Alias) Instantiate(typeArgs map[string]Type) Type {
	a.ensureResolved()
	return SubstituteType(a.Derived, typeArgs)
}

// SubstituteType recursively replaces type parameter references with concrete types.
func SubstituteType(t Type, args map[string]Type) Type {
	switch v := t.(type) {
	case *Alias:
		if v.Constraint != nil {
			// Type parameter: substitute if matched.
			if concrete, ok := args[v.Name]; ok {
				return concrete
			}

			return v
		}

		v.ensureResolved()

		return SubstituteType(v.Derived, args)
	case *Slice:
		return &Slice{Element: SubstituteType(v.Element, args)}
	case *Array:
		return &Array{Element: SubstituteType(v.Element, args), Length: v.Length}
	case *Map:
		return &Map{Key: SubstituteType(v.Key, args), Value: SubstituteType(v.Value, args)}
	case *Set:
		return &Set{Element: SubstituteType(v.Element, args)}
	case *Option:
		return &Option{Value: SubstituteType(v.Value, args)}
	case *Reference:
		return &Reference{Value: SubstituteType(v.Value, args)}
	case *Tuple:
		types := make([]Type, len(v.Types))
		for i, elem := range v.Types {
			types[i] = SubstituteType(elem, args)
		}

		return &Tuple{Types: types, Exported: v.Exported, Global: v.Global}
	case *Either:
		return &Either{
			Left:     SubstituteType(v.Left, args),
			Right:    SubstituteType(v.Right, args),
			Exported: v.Exported,
			Global:   v.Global,
		}
	case *Union:
		variants := make([]Type, len(v.Variants))
		for i, variant := range v.Variants {
			variants[i] = SubstituteType(variant, args)
		}

		return &Union{
			Variants: variants,
			Exported: v.Exported,
			Global:   v.Global,
		}
	case *Result:
		return &Result{
			Value: SubstituteType(v.Value, args),
			Error: SubstituteType(v.Error, args),
		}
	case *Struct:
		fields := make([]*Field, len(v.Fields))
		for i, f := range v.Fields {
			fields[i] = &Field{
				Name:     f.Name,
				Type:     SubstituteType(f.Type, args),
				Exported: f.Exported,
			}
		}

		return &Struct{Fields: fields}
	case *Procedure:
		params := make([]*Parameter, len(v.Parameters))
		for i, p := range v.Parameters {
			params[i] = &Parameter{
				Name:     p.Name,
				Optional: p.Optional,
				Type:     SubstituteType(p.Type, args),
				Default:  p.Default,
			}
		}

		var retType Type
		if v.ReturnType != nil {
			retType = SubstituteType(v.ReturnType, args)
		}

		return &Procedure{
			Function:   v.Function,
			TypeParams: v.TypeParams,
			Parameters: params,
			ReturnType: retType,
		}
	default:
		// Basic types and other concrete types pass through unchanged.
		return t
	}
}
