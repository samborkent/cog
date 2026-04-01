package types

// AssignableTo reports whether a value of type src can be assigned to a
// variable of type dst. This is true when the types are Equal, or when
// dst is an Option type and src equals the option's inner type, or when
// dst is a Result type and src equals the result's value or error type.
func AssignableTo(src, dst Type) bool {
	if Equal(src, dst) {
		return true
	}

	// Allow assigning T to T? (Option[T]).
	if opt, ok := dst.(*Option); ok {
		return Equal(src, opt.Value)
	}
	if a, ok := dst.(*Alias); ok {
		if opt, ok := a.Underlying().(*Option); ok {
			return Equal(src, opt.Value)
		}
	}

	// Allow assigning T or E to T ! E (Result[T, E]).
	if r, ok := dst.(*Result); ok {
		return Equal(src, r.Value) || Equal(src, r.Error)
	}
	if a, ok := dst.(*Alias); ok {
		if r, ok := a.Underlying().(*Result); ok {
			return Equal(src, r.Value) || Equal(src, r.Error)
		}
	}

	return false
}

func Equal(a, b Type) bool {
	// Check Option before Underlying(), since Option.Underlying()
	// unwraps to the inner Value type.
	ao, aIsOpt := a.(*Option)
	bo, bIsOpt := b.(*Option)

	// Also check through aliases that resolve to Option.
	if !aIsOpt {
		ao, aIsOpt = a.Underlying().(*Option)
	}
	if !bIsOpt {
		bo, bIsOpt = b.Underlying().(*Option)
	}

	if aIsOpt != bIsOpt {
		return false
	}
	if aIsOpt {
		return Equal(ao.Value, bo.Value)
	}

	// Resolve aliases to their underlying types.
	au, bu := a.Underlying(), b.Underlying()

	// Fast path: same Kind check.
	if au.Kind() != bu.Kind() {
		return false
	}

	// For basic types, Kind equality is sufficient.
	// For composite types, compare structure recursively.
	switch at := au.(type) {
	case *Slice:
		bt := bu.(*Slice)
		return Equal(at.Element, bt.Element)
	case *Array:
		bt := bu.(*Array)
		if at.Length.String() != bt.Length.String() {
			return false
		}
		return Equal(at.Element, bt.Element)
	case *Map:
		bt := bu.(*Map)
		return Equal(at.Key, bt.Key) && Equal(at.Value, bt.Value)
	case *Set:
		bt := bu.(*Set)
		return Equal(at.Element, bt.Element)
	case *Pointer:
		bt := bu.(*Pointer)
		return Equal(at.Value, bt.Value)
	case *Tuple:
		bt := bu.(*Tuple)
		if len(at.Types) != len(bt.Types) {
			return false
		}
		for i := range at.Types {
			if !Equal(at.Types[i], bt.Types[i]) {
				return false
			}
		}
		return true
	case *Union:
		bt := bu.(*Union)
		return Equal(at.Either, bt.Either) && Equal(at.Or, bt.Or)
	case *Result:
		bt := bu.(*Result)
		return Equal(at.Value, bt.Value) && Equal(at.Error, bt.Error)
	case *Struct:
		bt := bu.(*Struct)
		if len(at.Fields) != len(bt.Fields) {
			return false
		}
		for i := range at.Fields {
			if at.Fields[i].Name != bt.Fields[i].Name {
				return false
			}
			if !Equal(at.Fields[i].Type, bt.Fields[i].Type) {
				return false
			}
		}
		return true
	case *Enum:
		bt := bu.(*Enum)
		return Equal(at.ValueType, bt.ValueType)
	case *Procedure:
		bt := bu.(*Procedure)
		if at.Function != bt.Function {
			return false
		}
		if len(at.Parameters) != len(bt.Parameters) {
			return false
		}
		for i := range at.Parameters {
			if !Equal(at.Parameters[i].Type, bt.Parameters[i].Type) {
				return false
			}
		}
		// Compare return types (both nil, or both equal).
		if (at.ReturnType == nil) != (bt.ReturnType == nil) {
			return false
		}
		if at.ReturnType != nil && !Equal(at.ReturnType, bt.ReturnType) {
			return false
		}
		return true
	default:
		// Basic types and Generic: Kind equality is sufficient.
		return true
	}
}

// Size returns the bit size of a numeric or bool kind.
// Returns -1 for non-numeric kinds (containers, strings, etc).
func Size(k Kind) int {
	switch k {
	case Bool, Int8, Uint8:
		return 8
	case Int16, Uint16, Float16:
		return 16
	case Int32, Uint32, Float32, Complex32:
		return 32
	case Int64, Uint64, Float64, Complex64:
		return 64
	case Int128, Uint128, Complex128:
		return 128
	default:
		return -1
	}
}
