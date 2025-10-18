package types

var None = &Basic{kind: Invalid}

func IsNone(t Type) bool {
	if t == nil {
		return true
	}

	n, ok := t.(*Basic)
	return ok && n == None
}

var Basics = []*Basic{
	None,
	{kind: ASCII, name: "ascii"},
	{kind: Bool, name: "bool"},
	{kind: Complex32, name: "complex32"},
	{kind: Complex64, name: "complex64"},
	{kind: Complex128, name: "complex128"},
	{kind: Int8, name: "int8"},
	{kind: Int16, name: "int16"},
	{kind: Int32, name: "int32"},
	{kind: Int64, name: "int64"},
	{kind: Int128, name: "int128"},
	{kind: Float16, name: "float16"},
	{kind: Float32, name: "float32"},
	{kind: Float64, name: "float64"},
	{kind: Uint8, name: "uint8"},
	{kind: Uint16, name: "uint16"},
	{kind: Uint32, name: "uint32"},
	{kind: Uint64, name: "uint64"},
	{kind: Uint128, name: "uint128"},
	{kind: UTF8, name: "utf8"},
	// Special types
	{kind: Context, name: "context"},
}

var _ Type = &Basic{}

type Basic struct {
	kind Kind
	name string
}

func (b *Basic) Kind() Kind {
	return b.kind
}

func (b *Basic) String() string {
	return b.name
}

func (b *Basic) Underlying() Type {
	return b
}
