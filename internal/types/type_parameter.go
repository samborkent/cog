package types

import "strings"

// TypeParam represents a named type parameter with a set of constraints.
// Distinct from Generic: Generic is a constraint family (set of allowed types),
// TypeParam is a named placeholder bound to one or more constraints.
//
// Examples:
//
//	<T ~ any>           → Constraints: [Any]
//	<T ~ int>           → Constraints: [Generics["int"]]
//	<T ~ string | int>  → Constraints: [Generics["string"], Generics["int"]]
type TypeParam struct {
	Name        string
	Constraints []Type // each entry is *Generic, Any, or a concrete type
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
	if len(tp.Constraints) == 1 {
		return tp.Constraints[0].String()
	}

	var out strings.Builder

	for i, c := range tp.Constraints {
		if i > 0 {
			out.WriteString(" | ")
		}
		out.WriteString(c.String())
	}

	return out.String()
}

func (tp *TypeParam) Underlying() Type {
	return tp
}

// SatisfiedBy reports whether a concrete type satisfies this
// type parameter's constraints (OR semantics: satisfies at least one).
func (tp *TypeParam) SatisfiedBy(concrete Type) bool {
	for _, c := range tp.Constraints {
		if Satisfies(concrete, c) {
			return true
		}
	}

	return false
}
