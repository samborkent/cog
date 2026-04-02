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
type TypeParameter struct {
	Name        string
	Constraints []Type // each entry is *Generic, Any, or a concrete type
}

var _ Type = &TypeParameter{}

func (tp *TypeParameter) Kind() Kind {
	return GenericKind
}

func (tp *TypeParameter) String() string {
	return tp.Name
}

// ConstraintString returns the constraint portion for display,
// e.g. "any", "int", or "string | int".
func (tp *TypeParameter) ConstraintString() string {
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

func (tp *TypeParameter) Underlying() Type {
	return tp
}

// SatisfiedBy reports whether a concrete type satisfies all of this
// type parameter's constraints (OR semantics: satisfies at least one).
func (tp *TypeParameter) SatisfiedBy(concrete Type) bool {
	for _, c := range tp.Constraints {
		if Satisfies(concrete, c) {
			return true
		}
	}

	return false
}
