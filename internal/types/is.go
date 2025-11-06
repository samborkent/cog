package types

func Equal(a, b Type) bool {
	return a.Underlying().Kind() == b.Underlying().Kind()
}

func IsBool(t Type) bool {
	return t.Underlying().Kind() == Bool
}

func IsComplex(t Type) bool {
	kind := t.Underlying().Kind()
	return kind == Complex32 || kind == Complex64 || kind == Complex128
}

func IsFixed(t Type) bool {
	return IsInt(t) || IsUint(t)
}

func IsFloat(t Type) bool {
	kind := t.Underlying().Kind()
	return kind == Float16 || kind == Float32 || kind == Float64
}

func IsInt(t Type) bool {
	kind := t.Underlying().Kind()
	return kind == Int8 || kind == Int16 || kind == Int32 || kind == Int64 || kind == Int128
}

func IsNumber(t Type) bool {
	return IsComplex(t) || IsReal(t)
}

func IsReal(t Type) bool {
	return IsUint(t) || IsSigned(t)
}

func IsSigned(t Type) bool {
	return IsFloat(t) || IsInt(t)
}

func IsString(t Type) bool {
	kind := t.Underlying().Kind()
	return kind == ASCII || kind == UTF8
}

func IsSummable(t Type) bool {
	return IsNumber(t) || IsString(t)
}

func IsUint(t Type) bool {
	kind := t.Underlying().Kind()
	return kind == Uint8 || kind == Uint16 || kind == Uint32 || kind == Uint64 || kind == Uint128
}
