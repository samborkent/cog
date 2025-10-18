package types

var Generics = map[string]*Generic{
	"complex": {constraints: []Type{Basics[Complex32], Basics[Complex64], Basics[Complex128]}, name: "complex"},
	"float":   {constraints: []Type{Basics[Float16], Basics[Float32], Basics[Float64]}, name: "float"},
	"int":     {constraints: []Type{Basics[Int8], Basics[Int16], Basics[Int32], Basics[Int64], Basics[Int128]}, name: "int"},
	"string":  {constraints: []Type{Basics[ASCII], Basics[UTF8]}, name: "string"},
	"uint":    {constraints: []Type{Basics[Uint8], Basics[Uint16], Basics[Uint32], Basics[Uint64], Basics[Uint128]}, name: "uint"},
}

var _ Type = &Generic{}

type Generic struct {
	constraints []Type
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
