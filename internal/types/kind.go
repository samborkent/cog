package types

import "github.com/samborkent/cog/internal/tokens"

type Kind int8

const (
	Invalid Kind = iota

	// Basic types
	ASCII
	Bool
	Complex32
	Complex64
	Complex128
	Int8
	Int16
	Int32
	Int64
	Int128
	Float16
	Float32
	Float64
	Uint8
	Uint16
	Uint32
	Uint64
	Uint128
	UTF8

	// Generic type
	GenericKind

	// Container types
	ArrayKind
	SliceKind
	EnumKind
	MapKind
	SetKind
	StructKind

	// Combined types
	TupleKind
	UnionKind

	// Modified types
	OptionKind

	// Function type
	ProcedureKind
)

func (t Kind) String() string {
	switch t {
	case ASCII:
		return "ascii"
	case Bool:
		return "bool"
	case Complex32:
		return "complex32"
	case Complex64:
		return "complex64"
	case Complex128:
		return "complex128"
	case Float16:
		return "float16"
	case Float32:
		return "float32"
	case Float64:
		return "float64"
	case Int8:
		return "int8"
	case Int16:
		return "int16"
	case Int32:
		return "int32"
	case Int64:
		return "int64"
	case Int128:
		return "int128"
	case Uint8:
		return "uint8"
	case Uint16:
		return "uint16"
	case Uint32:
		return "uint32"
	case Uint64:
		return "uint64"
	case Uint128:
		return "uint128"
	case UTF8:
		return "utf8"
	case GenericKind:
		return "generic"
	case EnumKind:
		return "enum"
	case MapKind:
		return "map"
	case SetKind:
		return "set"
	case StructKind:
		return "struct"
	case TupleKind:
		return "tuple"
	case UnionKind:
		return "union"
	case OptionKind:
		return "option"
	case ProcedureKind:
		return "proc"
	case Invalid:
		fallthrough
	default:
		return ""
	}
}

var Lookup = map[tokens.Type]Type{
	// Basic types
	tokens.ASCII:      Basics[ASCII],
	tokens.Bool:       Basics[Bool],
	tokens.Complex32:  Basics[Complex32],
	tokens.Complex64:  Basics[Complex64],
	tokens.Complex128: Basics[Complex128],
	tokens.Float16:    Basics[Float16],
	tokens.Float32:    Basics[Float32],
	tokens.Float64:    Basics[Float64],
	tokens.Int8:       Basics[Int8],
	tokens.Int16:      Basics[Int16],
	tokens.Int32:      Basics[Int32],
	tokens.Int64:      Basics[Int64],
	tokens.Int128:     Basics[Int128],
	tokens.Uint8:      Basics[Uint8],
	tokens.Uint16:     Basics[Uint16],
	tokens.Uint32:     Basics[Uint32],
	tokens.Uint64:     Basics[Uint64],
	tokens.Uint128:    Basics[Uint128],
	tokens.UTF8:       Basics[UTF8],

	// Literal types
	tokens.Complex: Basics[Complex64],
	tokens.Float:   Basics[Float64],
	tokens.Int:     Basics[Int64],
	tokens.String:  Basics[UTF8],
	tokens.Uint:    Basics[Uint64],

	// Container types
	tokens.Map:    &Map{},
	tokens.Enum:   &Enum{},
	tokens.Set:    &Set{},
	tokens.Struct: &Struct{},

	// Procedure type
	tokens.Procedure: &Procedure{},
	tokens.Function:  &Procedure{Function: true},
}
