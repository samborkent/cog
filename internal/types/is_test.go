package types

import "testing"

// mockExpr implements the unexported expression interface for testing Array.Length.
type mockExpr struct {
	str string
	typ Type
}

func (m mockExpr) String() string { return m.str }
func (m mockExpr) Type() Type     { return m.typ }

func TestEqual(t *testing.T) {
	t.Parallel()

	int8Type := Basics[Int8]
	int64Type := Basics[Int64]
	utf8Type := Basics[UTF8]
	asciiType := Basics[ASCII]
	boolType := Basics[Bool]

	tests := []struct {
		name string
		a, b Type
		want bool
	}{
		// Basic types
		{"same basic", int64Type, int64Type, true},
		{"different basic", int8Type, int64Type, false},
		{"utf8 vs ascii", utf8Type, asciiType, false},

		// Slices
		{"same slice", &Slice{Element: int64Type}, &Slice{Element: int64Type}, true},
		{"different slice element", &Slice{Element: int8Type}, &Slice{Element: int64Type}, false},
		{"nested slice equal", &Slice{Element: &Slice{Element: int8Type}}, &Slice{Element: &Slice{Element: int8Type}}, true},
		{"nested slice differ", &Slice{Element: &Slice{Element: int8Type}}, &Slice{Element: &Slice{Element: int64Type}}, false},

		// Arrays
		{"same array", &Array{Element: int64Type, Length: mockExpr{str: "3"}}, &Array{Element: int64Type, Length: mockExpr{str: "3"}}, true},
		{"different array element", &Array{Element: int8Type, Length: mockExpr{str: "3"}}, &Array{Element: int64Type, Length: mockExpr{str: "3"}}, false},
		{"different array length", &Array{Element: int64Type, Length: mockExpr{str: "3"}}, &Array{Element: int64Type, Length: mockExpr{str: "5"}}, false},

		// Maps
		{"same map", &Map{Key: utf8Type, Value: int64Type}, &Map{Key: utf8Type, Value: int64Type}, true},
		{"different map key", &Map{Key: utf8Type, Value: int64Type}, &Map{Key: asciiType, Value: int64Type}, false},
		{"different map value", &Map{Key: utf8Type, Value: int64Type}, &Map{Key: utf8Type, Value: int8Type}, false},

		// Sets
		{"same set", &Set{Element: utf8Type}, &Set{Element: utf8Type}, true},
		{"different set", &Set{Element: utf8Type}, &Set{Element: asciiType}, false},

		// Options
		{"same option", &Option{Value: utf8Type}, &Option{Value: utf8Type}, true},
		{"different option", &Option{Value: utf8Type}, &Option{Value: int64Type}, false},
		{"option vs non-option", &Option{Value: utf8Type}, utf8Type, false},

		// References
		{"same pointer", &Reference{Value: int64Type}, &Reference{Value: int64Type}, true},
		{"different pointer", &Reference{Value: int64Type}, &Reference{Value: int8Type}, false},

		// Tuples
		{"same tuple", &Tuple{Types: []Type{utf8Type, int64Type}}, &Tuple{Types: []Type{utf8Type, int64Type}}, true},
		{"different tuple element", &Tuple{Types: []Type{utf8Type, int64Type}}, &Tuple{Types: []Type{utf8Type, int8Type}}, false},
		{"different tuple length", &Tuple{Types: []Type{utf8Type, int64Type}}, &Tuple{Types: []Type{utf8Type, int64Type, boolType}}, false},

		// Either
		{"same either", &Either{Left: utf8Type, Right: int64Type}, &Either{Left: utf8Type, Right: int64Type}, true},
		{"different either left", &Either{Left: utf8Type, Right: int64Type}, &Either{Left: asciiType, Right: int64Type}, false},
		{"different either right", &Either{Left: utf8Type, Right: int64Type}, &Either{Left: utf8Type, Right: int8Type}, false},

		// Unions (constraints)
		{"same union", &Union{Variants: []Type{utf8Type, int64Type}}, &Union{Variants: []Type{utf8Type, int64Type}}, true},
		{"different union left", &Union{Variants: []Type{utf8Type, int64Type}}, &Union{Variants: []Type{asciiType, int64Type}}, false},
		{"different union right", &Union{Variants: []Type{utf8Type, int64Type}}, &Union{Variants: []Type{utf8Type, int8Type}}, false},

		// Structs
		{"same struct", &Struct{Fields: []*Field{{Name: "x", Type: int64Type}}}, &Struct{Fields: []*Field{{Name: "x", Type: int64Type}}}, true},
		{"different struct field name", &Struct{Fields: []*Field{{Name: "x", Type: int64Type}}}, &Struct{Fields: []*Field{{Name: "y", Type: int64Type}}}, false},
		{"different struct field type", &Struct{Fields: []*Field{{Name: "x", Type: int64Type}}}, &Struct{Fields: []*Field{{Name: "x", Type: int8Type}}}, false},
		{"different struct field count", &Struct{Fields: []*Field{{Name: "x", Type: int64Type}}}, &Struct{Fields: []*Field{{Name: "x", Type: int64Type}, {Name: "y", Type: int8Type}}}, false},

		// Enums
		{"same enum", &Enum{ValueType: utf8Type}, &Enum{ValueType: utf8Type}, true},
		{"different enum value type", &Enum{ValueType: utf8Type}, &Enum{ValueType: int64Type}, false},

		// Procedures
		{"same func", &Procedure{Function: true, Parameters: []*Parameter{{Type: utf8Type}}, ReturnType: int64Type}, &Procedure{Function: true, Parameters: []*Parameter{{Type: utf8Type}}, ReturnType: int64Type}, true},
		{"func vs proc", &Procedure{Function: true, ReturnType: int64Type}, &Procedure{Function: false, ReturnType: int64Type}, false},
		{"different param type", &Procedure{Function: true, Parameters: []*Parameter{{Type: utf8Type}}, ReturnType: int64Type}, &Procedure{Function: true, Parameters: []*Parameter{{Type: int8Type}}, ReturnType: int64Type}, false},
		{"different param count", &Procedure{Function: true, Parameters: []*Parameter{{Type: utf8Type}}, ReturnType: int64Type}, &Procedure{Function: true, Parameters: []*Parameter{{Type: utf8Type}, {Type: int8Type}}, ReturnType: int64Type}, false},
		{"different return type", &Procedure{Function: true, ReturnType: int64Type}, &Procedure{Function: true, ReturnType: int8Type}, false},
		{"return vs no return", &Procedure{Function: false, ReturnType: int64Type}, &Procedure{Function: false}, false},

		// Aliases resolve through
		{"alias to same basic", &Alias{Name: "MyInt", Derived: int64Type}, int64Type, true},
		{"alias to different basic", &Alias{Name: "MyInt", Derived: int64Type}, int8Type, false},
		{"alias to slice", &Alias{Name: "Ints", Derived: &Slice{Element: int64Type}}, &Slice{Element: int64Type}, true},
		{"alias to different slice", &Alias{Name: "Ints", Derived: &Slice{Element: int64Type}}, &Slice{Element: int8Type}, false},

		// Cross-kind always false
		{"slice vs map", &Slice{Element: int64Type}, &Map{Key: int64Type, Value: int64Type}, false},
		{"set vs slice", &Set{Element: int64Type}, &Slice{Element: int64Type}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := Equal(tt.a, tt.b)
			if got != tt.want {
				t.Errorf("Equal(%v, %v) = %v, want %v", tt.a, tt.b, got, tt.want)
			}
		})
	}
}

func TestIsNone(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		typ  Type
		want bool
	}{
		{"nil", nil, true},
		{"None", None, true},
		{"int64", Basics[Int64], false},
		{"slice", &Slice{Element: Basics[Int64]}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := IsNone(tt.typ); got != tt.want {
				t.Errorf("IsNone(%v) = %v, want %v", tt.typ, got, tt.want)
			}
		})
	}
}

func TestIsBool(t *testing.T) {
	t.Parallel()

	if !IsBool(Basics[Bool]) {
		t.Error("IsBool(bool) = false")
	}

	if IsBool(Basics[Int64]) {
		t.Error("IsBool(int64) = true")
	}
}

func TestIsComplex(t *testing.T) {
	t.Parallel()

	for _, k := range []Kind{Complex32, Complex64, Complex128} {
		if !IsComplex(Basics[k]) {
			t.Errorf("IsComplex(%s) = false", Basics[k])
		}
	}

	if IsComplex(Basics[Float64]) {
		t.Error("IsComplex(float64) = true")
	}
}

func TestIsFloat(t *testing.T) {
	t.Parallel()

	for _, k := range []Kind{Float16, Float32, Float64} {
		if !IsFloat(Basics[k]) {
			t.Errorf("IsFloat(%s) = false", Basics[k])
		}
	}

	if IsFloat(Basics[Int64]) {
		t.Error("IsFloat(int64) = true")
	}
}

func TestIsInt(t *testing.T) {
	t.Parallel()

	for _, k := range []Kind{Int8, Int16, Int32, Int64, Int128} {
		if !IsInt(Basics[k]) {
			t.Errorf("IsInt(%s) = false", Basics[k])
		}
	}

	if IsInt(Basics[Uint64]) {
		t.Error("IsInt(uint64) = true")
	}
}

func TestIsUint(t *testing.T) {
	t.Parallel()

	for _, k := range []Kind{Uint8, Uint16, Uint32, Uint64, Uint128} {
		if !IsUint(Basics[k]) {
			t.Errorf("IsUint(%s) = false", Basics[k])
		}
	}

	if IsUint(Basics[Int64]) {
		t.Error("IsUint(int64) = true")
	}
}

func TestIsFixed(t *testing.T) {
	t.Parallel()

	if !IsFixed(Basics[Int32]) {
		t.Error("IsFixed(int32) = false")
	}

	if !IsFixed(Basics[Uint16]) {
		t.Error("IsFixed(uint16) = false")
	}

	if IsFixed(Basics[Float64]) {
		t.Error("IsFixed(float64) = true")
	}
}

func TestIsNumber(t *testing.T) {
	t.Parallel()

	if !IsNumber(Basics[Int64]) {
		t.Error("IsNumber(int64) = false")
	}

	if !IsNumber(Basics[Float32]) {
		t.Error("IsNumber(float32) = false")
	}

	if !IsNumber(Basics[Complex64]) {
		t.Error("IsNumber(complex64) = false")
	}

	if !IsNumber(Basics[Uint128]) {
		t.Error("IsNumber(uint128) = false")
	}

	if IsNumber(Basics[Bool]) {
		t.Error("IsNumber(bool) = true")
	}
}

func TestIsReal(t *testing.T) {
	t.Parallel()

	if !IsReal(Basics[Int64]) {
		t.Error("IsReal(int64) = false")
	}

	if !IsReal(Basics[Uint32]) {
		t.Error("IsReal(uint32) = false")
	}

	if !IsReal(Basics[Float64]) {
		t.Error("IsReal(float64) = false")
	}
	// Complex is real via IsSigned.
	if !IsReal(Basics[Complex64]) {
		t.Error("IsReal(complex64) = false")
	}

	if IsReal(Basics[Bool]) {
		t.Error("IsReal(bool) = true")
	}
}

func TestIsSigned(t *testing.T) {
	t.Parallel()

	if !IsSigned(Basics[Int64]) {
		t.Error("IsSigned(int64) = false")
	}

	if !IsSigned(Basics[Float32]) {
		t.Error("IsSigned(float32) = false")
	}

	if !IsSigned(Basics[Complex128]) {
		t.Error("IsSigned(complex128) = false")
	}

	if IsSigned(Basics[Uint64]) {
		t.Error("IsSigned(uint64) = true")
	}
}

func TestIsString(t *testing.T) {
	t.Parallel()

	if !IsString(Basics[ASCII]) {
		t.Error("IsString(ascii) = false")
	}

	if !IsString(Basics[UTF8]) {
		t.Error("IsString(utf8) = false")
	}

	if IsString(Basics[Int64]) {
		t.Error("IsString(int64) = true")
	}
}

func TestIsSummable(t *testing.T) {
	t.Parallel()

	if !IsSummable(Basics[Int64]) {
		t.Error("IsSummable(int64) = false")
	}

	if !IsSummable(Basics[UTF8]) {
		t.Error("IsSummable(utf8) = false")
	}

	if IsSummable(Basics[Bool]) {
		t.Error("IsSummable(bool) = true")
	}
}

func TestIsIterator(t *testing.T) {
	t.Parallel()

	if !IsIterator(Basics[UTF8]) {
		t.Error("IsIterator(utf8) = false")
	}

	if !IsIterator(Basics[ASCII]) {
		t.Error("IsIterator(ascii) = false")
	}

	if !IsIterator(&Slice{Element: Basics[Int64]}) {
		t.Error("IsIterator([]int64) = false")
	}

	if !IsIterator(&Array{Element: Basics[Int64], Length: mockExpr{str: "3"}}) {
		t.Error("IsIterator([3]int64) = false")
	}

	if !IsIterator(&Map{Key: Basics[UTF8], Value: Basics[Int64]}) {
		t.Error("IsIterator(map) = false")
	}

	if !IsIterator(&Set{Element: Basics[UTF8]}) {
		t.Error("IsIterator(set) = false")
	}

	if !IsIterator(&Enum{ValueType: Basics[UTF8]}) {
		t.Error("IsIterator(enum) = false")
	}

	if IsIterator(Basics[Int64]) {
		t.Error("IsIterator(int64) = true")
	}
}

func TestIsComparable(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		typ  Type
		want bool
	}{
		{"basic int64", Basics[Int64], true},
		{"basic bool", Basics[Bool], true},
		{"reference", &Reference{Value: Basics[Int64]}, true},
		{"enum", &Enum{ValueType: Basics[Int64]}, true},
		{"struct with comparable fields", &Struct{Fields: []*Field{{Name: "x", Type: Basics[Int64]}}}, true},
		{"struct with slice field", &Struct{Fields: []*Field{{Name: "x", Type: &Slice{Element: Basics[Int64]}}}}, false},
		{"array of comparable", &Array{Element: Basics[Int64]}, true},
		{"array of slice", &Array{Element: &Slice{Element: Basics[Int64]}}, false},
		{"tuple of comparable", &Tuple{Types: []Type{Basics[Int64], Basics[UTF8]}}, true},
		{"tuple with map", &Tuple{Types: []Type{Basics[Int64], &Map{Key: Basics[UTF8], Value: Basics[Int64]}}}, false},
		{"slice", &Slice{Element: Basics[Int64]}, false},
		{"map", &Map{Key: Basics[UTF8], Value: Basics[Int64]}, false},
		{"set with comparable element", &Set{Element: Basics[Int64]}, true},
		{"set with slice element", &Set{Element: &Slice{Element: Basics[Int64]}}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := IsComparable(tt.typ); got != tt.want {
				t.Errorf("IsComparable(%v) = %v, want %v", tt.typ, got, tt.want)
			}
		})
	}
}
