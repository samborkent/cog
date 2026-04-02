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

func TestSatisfies(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		concrete   Type
		constraint Type
		want       bool
	}{
		// any constraint
		{"int64 satisfies any", Basics[Int64], Any, true},
		{"utf8 satisfies any", Basics[UTF8], Any, true},
		{"bool satisfies any", Basics[Bool], Any, true},

		// int constraint
		{"int8 satisfies int", Basics[Int8], Generics["int"], true},
		{"int16 satisfies int", Basics[Int16], Generics["int"], true},
		{"int32 satisfies int", Basics[Int32], Generics["int"], true},
		{"int64 satisfies int", Basics[Int64], Generics["int"], true},
		{"int128 satisfies int", Basics[Int128], Generics["int"], true},
		{"uint64 not int", Basics[Uint64], Generics["int"], false},
		{"utf8 not int", Basics[UTF8], Generics["int"], false},

		// uint constraint
		{"uint64 satisfies uint", Basics[Uint64], Generics["uint"], true},
		{"int64 not uint", Basics[Int64], Generics["uint"], false},

		// float constraint
		{"float16 satisfies float", Basics[Float16], Generics["float"], true},
		{"float32 satisfies float", Basics[Float32], Generics["float"], true},
		{"float64 satisfies float", Basics[Float64], Generics["float"], true},
		{"int64 not float", Basics[Int64], Generics["float"], false},

		// complex constraint
		{"complex32 satisfies complex", Basics[Complex32], Generics["complex"], true},
		{"complex64 satisfies complex", Basics[Complex64], Generics["complex"], true},
		{"complex128 satisfies complex", Basics[Complex128], Generics["complex"], true},
		{"float64 not complex", Basics[Float64], Generics["complex"], false},

		// string constraint
		{"ascii satisfies string", Basics[ASCII], Generics["string"], true},
		{"utf8 satisfies string", Basics[UTF8], Generics["string"], true},
		{"int64 not string", Basics[Int64], Generics["string"], false},

		// signed constraint (int + float + complex)
		{"int64 satisfies signed", Basics[Int64], Generics["signed"], true},
		{"float32 satisfies signed", Basics[Float32], Generics["signed"], true},
		{"complex128 satisfies signed", Basics[Complex128], Generics["signed"], true},
		{"uint64 not signed", Basics[Uint64], Generics["signed"], false},
		{"bool not signed", Basics[Bool], Generics["signed"], false},

		// number constraint (signed + uint)
		{"int64 satisfies number", Basics[Int64], Generics["number"], true},
		{"uint64 satisfies number", Basics[Uint64], Generics["number"], true},
		{"float32 satisfies number", Basics[Float32], Generics["number"], true},
		{"complex64 satisfies number", Basics[Complex64], Generics["number"], true},
		{"bool not number", Basics[Bool], Generics["number"], false},
		{"utf8 not number", Basics[UTF8], Generics["number"], false},

		// concrete constraint (falls back to Equal)
		{"int64 satisfies int64", Basics[Int64], Basics[Int64], true},
		{"int64 not utf8", Basics[Int64], Basics[UTF8], false},

		// alias satisfies constraint via Underlying
		{"alias(int64) satisfies int", &Alias{Name: "MyInt", Derived: Basics[Int64]}, Generics["int"], true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := Satisfies(tt.concrete, tt.constraint); got != tt.want {
				t.Errorf("Satisfies(%v, %v) = %v, want %v", tt.concrete, tt.constraint, got, tt.want)
			}
		})
	}
}

func TestGenericConstraints(t *testing.T) {
	t.Parallel()

	t.Run("signed_exists", func(t *testing.T) {
		t.Parallel()
		g, ok := Generics["signed"]
		if !ok {
			t.Fatal("Generics[\"signed\"] not found")
		}
		if g.String() != "signed" {
			t.Errorf("expected name \"signed\", got %q", g.String())
		}
		if len(g.Constraints) == 0 {
			t.Error("signed has no constraints")
		}
	})

	t.Run("number_exists", func(t *testing.T) {
		t.Parallel()
		g, ok := Generics["number"]
		if !ok {
			t.Fatal("Generics[\"number\"] not found")
		}
		if g.String() != "number" {
			t.Errorf("expected name \"number\", got %q", g.String())
		}
		if len(g.Constraints) == 0 {
			t.Error("number has no constraints")
		}
	})

	t.Run("number_covers_all_numeric", func(t *testing.T) {
		t.Parallel()
		numericKinds := []Kind{
			Int8, Int16, Int32, Int64, Int128,
			Uint8, Uint16, Uint32, Uint64, Uint128,
			Float16, Float32, Float64,
			Complex32, Complex64, Complex128,
		}
		for _, k := range numericKinds {
			if !Satisfies(Basics[k], Generics["number"]) {
				t.Errorf("expected %s to satisfy number", Basics[k])
			}
		}
	})
}
