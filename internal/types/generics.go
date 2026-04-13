package types

// Constraints is the set of builtin named type constraints.
// Each entry is a named Union whose Variants list the concrete types
// that satisfy the constraint.
var Constraints = map[string]*Union{
	"complex": {Name: "complex", Variants: []Type{Basics[Complex32], Basics[Complex64], Basics[Complex128]}},
	"float":   {Name: "float", Variants: []Type{Basics[Float16], Basics[Float32], Basics[Float64]}},
	"int":     {Name: "int", Variants: []Type{Basics[Int8], Basics[Int16], Basics[Int32], Basics[Int64], Basics[Int128]}},
	"string":  {Name: "string", Variants: []Type{Basics[ASCII], Basics[UTF8]}},
	"uint":    {Name: "uint", Variants: []Type{Basics[Uint8], Basics[Uint16], Basics[Uint32], Basics[Uint64], Basics[Uint128]}},
}

func init() {
	// Composite constraints reference other constraints, so they must be
	// initialised after the base entries exist.
	Constraints["signed"] = &Union{
		Name: "signed",
		Variants: flatten(
			Constraints["int"].Variants,
			Constraints["float"].Variants,
			Constraints["complex"].Variants,
		),
	}
	Constraints["number"] = &Union{
		Name: "number",
		Variants: flatten(
			Constraints["signed"].Variants,
			Constraints["uint"].Variants,
		),
	}
	Constraints["ordered"] = &Union{
		Name: "ordered",
		Variants: flatten(
			Constraints["int"].Variants,
			Constraints["uint"].Variants,
			Constraints["float"].Variants,
			Constraints["string"].Variants,
		),
	}
	Constraints["summable"] = &Union{
		Name: "summable",
		Variants: flatten(
			Constraints["number"].Variants,
			Constraints["string"].Variants,
		),
	}
	Constraints["comparable"] = &Union{
		Name: "comparable",
		Variants: flatten(
			Constraints["ordered"].Variants,
			Constraints["complex"].Variants,
			[]Type{Basics[Bool]},
			// Sentinel zero-values: Satisfies matches by Kind(), so
			// these allow any struct, array, enum, pointer, tuple,
			// or set to satisfy comparable.
			[]Type{&Struct{}, &Array{}, &Enum{}, &Reference{}, &Tuple{}, &Set{}},
		),
	}
}

func flatten(slices ...[]Type) []Type {
	var n int
	for _, s := range slices {
		n += len(s)
	}

	out := make([]Type, 0, n)
	for _, s := range slices {
		out = append(out, s...)
	}

	return out
}

// LookupConstraint returns the builtin constraint type for the given name.
// This covers all entries in the Constraints map plus "any".
func LookupConstraint(name string) (Type, bool) {
	if name == "any" {
		return Any, true
	}

	u, ok := Constraints[name]
	if !ok {
		return nil, false
	}

	return u, true
}
