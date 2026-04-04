package types

// TODO: TypeParam is redundant, this could be handled by types.Alias

// TypeParam represents a named type parameter with a set of constraints.
// Distinct from Generic: Generic is a constraint family (set of allowed types),
// TypeParam is a named placeholder bound to one or more constraints.
//
// Examples:
//
//	<T ~ any>           → Constraint: Any
//	<T ~ int>           → Constraint: Generics["int"]
//	<T ~ string | int>  → Constraint: Union{Variants: [Generics["string"], Generics["int"]]}
type TypeParam struct {
	Name       string
	Constraint Type // *Generic, *Union, Any, or a concrete type
}

var _ Type = &TypeParam{}

func (tp *TypeParam) Kind() Kind {
	return GenericKind
}

func (tp *TypeParam) String() string {
	return tp.Name
}

// ConstraintString returns the constraint portion for display,
// e.g. "any", "int", or "string | int".
func (tp *TypeParam) ConstraintString() string {
	return tp.Constraint.String()
}

func (tp *TypeParam) Underlying() Type {
	return tp
}

// SatisfiedBy reports whether a concrete type satisfies this
// type parameter's constraints (OR semantics: satisfies at least one).
func (tp *TypeParam) SatisfiedBy(concrete Type) bool {
	if union, ok := tp.Constraint.(*Union); ok {
		for _, v := range union.Variants {
			if Satisfies(concrete, v) {
				return true
			}
		}

		return false
	}

	return Satisfies(concrete, tp.Constraint)
}
