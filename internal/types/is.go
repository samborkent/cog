package types

func IsBool(t Type) bool {
	return t.Kind() == Bool
}

func IsComplex(t Type) bool {
	kind := t.Kind()
	return kind == Complex32 || kind == Complex64 || kind == Complex128
}

func IsFixed(t Type) bool {
	return IsInt(t) || IsUint(t)
}

func IsFloat(t Type) bool {
	kind := t.Kind()
	return kind == Float16 || kind == Float32 || kind == Float64
}

func IsInt(t Type) bool {
	kind := t.Kind()
	return kind == Int8 || kind == Int16 || kind == Int32 || kind == Int64 || kind == Int128
}

func IsNumber(t Type) bool {
	return IsComplex(t) || IsReal(t)
}

func IsIndexable(t Type) bool {
	kind := t.Kind()
	return kind == ArrayKind || kind == SliceKind || kind == MapKind || kind == SetKind || IsString(t)
}

func IsIterator(t Type) bool {
	kind := t.Kind()
	return IsString(t) || kind == ArrayKind || kind == SliceKind || kind == MapKind || kind == SetKind || kind == EnumKind
}

func IsReal(t Type) bool {
	return IsUint(t) || IsSigned(t)
}

func IsSigned(t Type) bool {
	return IsComplex(t) || IsFloat(t) || IsInt(t)
}

func IsString(t Type) bool {
	kind := t.Kind()
	return kind == ASCII || kind == UTF8
}

func IsSummable(t Type) bool {
	return IsNumber(t) || IsString(t)
}

func IsUint(t Type) bool {
	kind := t.Kind()
	return kind == Uint8 || kind == Uint16 || kind == Uint32 || kind == Uint64 || kind == Uint128
}

func IsBasic(t Type) bool {
	return IsBool(t) || IsNumber(t) || IsString(t) || t.Kind() == ArrayKind ||
		// Basic (non-pointer containing) structs are also basic types.
		(t.Kind() == StructKind && !t.Underlying().(*Struct).IsComplex)
}

// IsComparable reports whether a type supports == in Go.
// Slices, maps, and functions are not comparable. Structs and arrays
// are comparable only if all their elements/fields are comparable.
func IsComparable(t Type) bool {
	switch v := t.Underlying().(type) {
	case *Basic:
		return true
	case *Enum:
		return true
	case *Reference:
		return true
	case *Set:
		if v.Element == nil {
			// Zero value is comparable.
			return true
		}

		return IsComparable(v.Element)
	case *Struct:
		for _, f := range v.Fields {
			if !IsComparable(f.Type) {
				return false
			}
		}

		return true
	case *Array:
		if v.Element == nil {
			// Zero value is comparable.
			return true
		}

		return IsComparable(v.Element)
	case *Tuple:
		for _, elem := range v.Types {
			if !IsComparable(elem) {
				return false
			}
		}

		return true
	case *Slice, *Map, *Procedure:
		return false
	default:
		return true
	}
}

// Pointer types are types which are pointer types under the hood.
func IsPointer(t Type) bool {
	kind := t.Kind()
	return kind == ReferenceKind || kind == SliceKind || kind == SetKind || kind == MapKind || kind == ProcedureKind
}

// IsPointerLike reports whether a type needs deep copy on immutable → var assignment.
// Slices, Maps, Sets, Procedures (closures) are always pointer-like.
// Structs are pointer-like only if they contain at least one pointer-like field (IsComplex).
// Arrays are pointer-like if their element type is pointer-like.
// References (&T) are NOT pointer-like (immutable, copied by pointer-copy).
// Primitives (int, float, bool, string) are never pointer-like.
// Options/Results/Eithers are pointer-like if any of their constituent types are.
// Aliases recurse through Underlying().
func IsPointerLike(t Type) bool {
	t = t.Underlying()

	switch v := t.(type) {
	case *Basic:
		return false
	case *Enum:
		return false
	case *Reference:
		return false
	case *Slice:
		return true
	case *Map:
		return true
	case *Set:
		return true
	case *Procedure:
		return true
	case *Struct:
		return v.IsComplex
	case *Array:
		if v.Element == nil {
			return false
		}
		return IsPointerLike(v.Element)
	case *Option:
		return IsPointerLike(v.Value)
	case *Result:
		return IsPointerLike(v.Value) || IsPointerLike(v.Error)
	case *Either:
		return IsPointerLike(v.Left) || IsPointerLike(v.Right)
	default:
		return false
	}
}
