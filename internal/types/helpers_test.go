package types

import "testing"

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

func TestAssignableTo(t *testing.T) {
	t.Parallel()

	int64Type := Basics[Int64]
	utf8Type := Basics[UTF8]

	tests := []struct {
		name     string
		src, dst Type
		want     bool
	}{
		{"same type", int64Type, int64Type, true},
		{"different type", int64Type, utf8Type, false},
		{"T to T?", utf8Type, &Option{Value: utf8Type}, true},
		{"T? to T?", &Option{Value: utf8Type}, &Option{Value: utf8Type}, true},
		{"wrong T to T?", int64Type, &Option{Value: utf8Type}, false},
		{"T to alias(T?)", utf8Type, &Alias{Name: "Opt", Derived: &Option{Value: utf8Type}}, true},
		{"wrong T to alias(T?)", int64Type, &Alias{Name: "Opt", Derived: &Option{Value: utf8Type}}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := AssignableTo(tt.src, tt.dst); got != tt.want {
				t.Errorf("AssignableTo(%v, %v) = %v, want %v", tt.src, tt.dst, got, tt.want)
			}
		})
	}
}

func TestAnyType(t *testing.T) {
	t.Parallel()

	if Any.Kind() != AnyKind {
		t.Errorf("Any.Kind() = %v, want AnyKind", Any.Kind())
	}
	if Any.String() != "any" {
		t.Errorf("Any.String() = %q, want %q", Any.String(), "any")
	}
	if Any.Underlying() != Any {
		t.Error("Any.Underlying() != Any")
	}
}

func TestEqualAny(t *testing.T) {
	t.Parallel()

	if !Equal(Any, Any) {
		t.Error("Equal(any, any) = false")
	}
	if Equal(Any, Basics[Int64]) {
		t.Error("Equal(any, int64) = true")
	}
	if Equal(Basics[Int64], Any) {
		t.Error("Equal(int64, any) = true")
	}
}
