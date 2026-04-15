package cog

import "reflect"

// Can be called for basic types, but shouldn't
func Copy[T any](src T) T {
	v := reflect.ValueOf(src)

	if isBasic(v.Kind()) {
		return src
	}

	return copyVal(v).Interface().(T)
}

func copyVal(v reflect.Value) reflect.Value {
	k := v.Kind()

	if isBasic(k) {
		return v
	}

	switch k {
	case reflect.Interface:
		if v.IsNil() {
			return v
		}

		return copyVal(v.Elem())
	case reflect.Map:
		mapCopy := reflect.MakeMapWithSize(v.Type(), v.Len())

		for key, val := range v.Seq2() {
			mapCopy.SetMapIndex(key, copyVal(val))
		}

		return mapCopy
	case reflect.Pointer:
		pointer := reflect.New(v.Type().Elem())
		pointer.Elem().Set(copyVal(v.Elem()))

		return pointer
	case reflect.Slice:
		if v.IsNil() {
			// Preserve nil slices
			return v
		}

		slice := reflect.MakeSlice(v.Type(), v.Len(), v.Cap())

		if v.Len() == 0 {
			// Return empty slice without copy.
			return slice
		}

		if isBasic(v.Index(0).Kind()) {
			// Underlying type is basic type, so we can simply copy.
			_ = reflect.Copy(slice, v)
			return slice
		}

		index := 0

		// Element-wise deep copy.
		for _, elem := range v.Seq2() {
			slice.Index(index).Set(copyVal(elem))
			index++
		}

		return slice
	case reflect.Struct:
		t := v.Type()

		containsPointer := false

		// Check if struct contains any pointer-like fields.
		for field := range t.Fields() {
			if !isBasic(field.Type.Kind()) {
				containsPointer = true
				break
			}
		}

		if !containsPointer {
			// Struct is strictly value type, return as is.
			return v
		}

		structPtr := reflect.New(v.Type())
		structCopy := structPtr.Elem()

		index := 0

		// Deep-copy struct field values.
		for _, val := range v.Fields() {
			structCopy.Field(index).Set(copyVal(val))
			index++
		}

		return structCopy
	default:
		panic("copyVal: unhandled type kind " + k.String())
	}
}

func isBasic(k reflect.Kind) bool {
	switch k {
	case reflect.Bool,
		reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uint, reflect.Uintptr,
		reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64, reflect.Int,
		reflect.Float32, reflect.Float64,
		reflect.Complex64, reflect.Complex128,
		reflect.String,
		reflect.Array,
		reflect.Func: // TODO: check if correct. It should copy the pointer of the function it points to, not the variable pointer.
		return true
	default:
		return false
	}
}
