package types

var Generics = map[string]*Generic{
	"complex": {Constraints: []Type{Basics[Complex32], Basics[Complex64], Basics[Complex128]}, name: "complex"},
	"float":   {Constraints: []Type{Basics[Float16], Basics[Float32], Basics[Float64]}, name: "float"},
	"int":     {Constraints: []Type{Basics[Int8], Basics[Int16], Basics[Int32], Basics[Int64], Basics[Int128]}, name: "int"},
	"string":  {Constraints: []Type{Basics[ASCII], Basics[UTF8]}, name: "string"},
	"uint":    {Constraints: []Type{Basics[Uint8], Basics[Uint16], Basics[Uint32], Basics[Uint64], Basics[Uint128]}, name: "uint"},
}

func init() {
	// Composite constraints reference other constraints, so they must be
	// initialised after the base entries exist.
	Generics["signed"] = &Generic{
		Constraints: flatten(
			Generics["int"].Constraints,
			Generics["float"].Constraints,
			Generics["complex"].Constraints,
		),
		name: "signed",
	}
	Generics["number"] = &Generic{
		Constraints: flatten(
			Generics["signed"].Constraints,
			Generics["uint"].Constraints,
		),
		name: "number",
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

var _ Type = &Generic{}

type Generic struct {
	Constraints []Type
	name        string
}

func (g *Generic) Kind() Kind {
	return GenericKind
}

func (g *Generic) String() string {
	return g.name
}

func (g *Generic) Underlying() Type {
	return g
}
