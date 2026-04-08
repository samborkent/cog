package types

import "testing"

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

	t.Run("ordered_exists", func(t *testing.T) {
		t.Parallel()

		g, ok := Generics["ordered"]
		if !ok {
			t.Fatal(`Generics["ordered"] not found`)
		}

		if g.String() != "ordered" {
			t.Errorf("expected name %q, got %q", "ordered", g.String())
		}
	})

	t.Run("comparable_exists", func(t *testing.T) {
		t.Parallel()

		g, ok := Generics["comparable"]
		if !ok {
			t.Fatal(`Generics["comparable"] not found`)
		}

		if g.String() != "comparable" {
			t.Errorf("expected name %q, got %q", "comparable", g.String())
		}
	})

	t.Run("ordered_excludes_complex", func(t *testing.T) {
		t.Parallel()

		for _, k := range []Kind{Complex32, Complex64, Complex128} {
			if Satisfies(Basics[k], Generics["ordered"]) {
				t.Errorf("%s should not satisfy ordered", Basics[k])
			}
		}
	})

	t.Run("summable_exists", func(t *testing.T) {
		t.Parallel()

		g, ok := Generics["summable"]
		if !ok {
			t.Fatal(`Generics["summable"] not found`)
		}

		if g.String() != "summable" {
			t.Errorf("expected name %q, got %q", "summable", g.String())
		}
	})

	// Verify that every type in each constraint actually supports the
	// operators that the constraint implies.

	t.Run("ordered_members_support_ordering_operators", func(t *testing.T) {
		t.Parallel()
		// Ordered types support <, >, <=, >= — must be real numeric or string.
		for _, member := range Generics["ordered"].Constraints {
			if !IsReal(member) && !IsString(member) {
				t.Errorf("%s is in ordered but does not support ordering operators (not real numeric or string)", member)
			}
		}
	})

	t.Run("ordered_members_support_equality_operators", func(t *testing.T) {
		t.Parallel()
		// Ordered types also support ==, != — all basic types do.
		for _, member := range Generics["ordered"].Constraints {
			if !IsBasic(member) {
				t.Errorf("%s is in ordered but is not a basic type (cannot use == or !=)", member)
			}
		}
	})

	t.Run("comparable_basic_members_support_equality", func(t *testing.T) {
		t.Parallel()
		// Every basic-type member of comparable must support ==, !=.
		for _, member := range Generics["comparable"].Constraints {
			// if !IsBasic(member) {
			// 	// Structural sentinels (Struct, Array, etc.) are
			// 	// comparable by kind; skip basic-type check.
			// 	continue
			// }

			if !IsComparable(member) {
				t.Errorf("%s is in comparable but does not support ==", member)
			}
		}
	})

	t.Run("comparable_structural_members_are_comparable_kinds", func(t *testing.T) {
		t.Parallel()
		// Structural sentinels must have kinds that support == in Go.
		comparableKinds := map[Kind]bool{
			StructKind: true, ArrayKind: true, EnumKind: true,
			ReferenceKind: true, TupleKind: true, SetKind: true,
		}

		for _, member := range Generics["comparable"].Constraints {
			if IsBasic(member) {
				continue
			}

			if !comparableKinds[member.Kind()] {
				t.Errorf("%s (kind %s) is a structural member of comparable but is not a comparable kind", member, member.Kind())
			}
		}
	})

	t.Run("comparable_excludes_non_comparable_kinds", func(t *testing.T) {
		t.Parallel()
		// Slices, maps, and procedures must NOT satisfy comparable.
		nonComparable := []Type{
			&Slice{Element: Basics[Int64]},
			&Map{Key: Basics[UTF8], Value: Basics[Int64]},
		}
		for _, typ := range nonComparable {
			if Satisfies(typ, Generics["comparable"]) {
				t.Errorf("%s (kind %s) should not satisfy comparable", typ, typ.Kind())
			}
		}
	})

	t.Run("summable_members_support_plus_operator", func(t *testing.T) {
		t.Parallel()
		// Every summable member must pass IsSummable (number or string).
		for _, member := range Generics["summable"].Constraints {
			if !IsSummable(member) {
				t.Errorf("%s is in summable but IsSummable() = false", member)
			}
		}
	})

	t.Run("summable_includes_complex_but_ordered_does_not", func(t *testing.T) {
		t.Parallel()
		// Complex types can be added (+) but not ordered (<).
		for _, k := range []Kind{Complex32, Complex64, Complex128} {
			if !Satisfies(Basics[k], Generics["summable"]) {
				t.Errorf("%s should satisfy summable", Basics[k])
			}

			if Satisfies(Basics[k], Generics["ordered"]) {
				t.Errorf("%s should not satisfy ordered", Basics[k])
			}
		}
	})

	t.Run("ordered_is_subset_of_summable", func(t *testing.T) {
		t.Parallel()
		// Every ordered type must also be summable.
		for _, member := range Generics["ordered"].Constraints {
			if !Satisfies(member, Generics["summable"]) {
				t.Errorf("%s satisfies ordered but not summable", member)
			}
		}
	})

	t.Run("ordered_is_subset_of_comparable", func(t *testing.T) {
		t.Parallel()
		// Every ordered type must also be comparable.
		for _, member := range Generics["ordered"].Constraints {
			if !Satisfies(member, Generics["comparable"]) {
				t.Errorf("%s satisfies ordered but not comparable", member)
			}
		}
	})
}

func TestLookupConstraint(t *testing.T) {
	t.Parallel()

	// All named constraints must be found.
	names := []string{
		"any", "int", "uint", "float", "complex", "string",
		"signed", "number", "ordered", "summable", "comparable",
	}
	for _, name := range names {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			typ, ok := LookupConstraint(name)
			if !ok {
				t.Fatalf("LookupConstraint(%q) not found", name)
			}

			if typ == nil {
				t.Fatalf("LookupConstraint(%q) returned nil", name)
			}

			if typ.String() != name {
				t.Errorf("LookupConstraint(%q).String() = %q", name, typ.String())
			}
		})
	}

	t.Run("unknown", func(t *testing.T) {
		t.Parallel()

		_, ok := LookupConstraint("nonexistent")
		if ok {
			t.Error("LookupConstraint(\"nonexistent\") should return false")
		}
	})
}

func TestTypeParam(t *testing.T) {
	t.Parallel()

	t.Run("kind", func(t *testing.T) {
		t.Parallel()

		tp := &TypeParam{Name: "T", Constraint: Any}
		if tp.Kind() != GenericKind {
			t.Errorf("TypeParam.Kind() = %v, want GenericKind", tp.Kind())
		}
	})

	t.Run("string", func(t *testing.T) {
		t.Parallel()

		tp := &TypeParam{Name: "T", Constraint: Any}
		if tp.String() != "T" {
			t.Errorf("TypeParam.String() = %q, want %q", tp.String(), "T")
		}
	})

	t.Run("constraint_string_single", func(t *testing.T) {
		t.Parallel()

		tp := &TypeParam{Name: "T", Constraint: Any}
		if tp.ConstraintString() != "any" {
			t.Errorf("ConstraintString() = %q, want %q", tp.ConstraintString(), "any")
		}
	})

	t.Run("constraint_string_multi", func(t *testing.T) {
		t.Parallel()

		tp := &TypeParam{
			Name:       "T",
			Constraint: &Union{Variants: []Type{Generics["string"], Generics["int"]}},
		}

		want := "string | int"
		if tp.ConstraintString() != want {
			t.Errorf("ConstraintString() = %q, want %q", tp.ConstraintString(), want)
		}
	})

	t.Run("satisfied_by_any", func(t *testing.T) {
		t.Parallel()

		tp := &TypeParam{Name: "T", Constraint: Any}
		if !tp.SatisfiedBy(Basics[Int64]) {
			t.Error("T ~ any should be satisfied by int64")
		}

		if !tp.SatisfiedBy(Basics[UTF8]) {
			t.Error("T ~ any should be satisfied by utf8")
		}
	})

	t.Run("satisfied_by_single_constraint", func(t *testing.T) {
		t.Parallel()

		tp := &TypeParam{Name: "T", Constraint: Generics["int"]}
		if !tp.SatisfiedBy(Basics[Int64]) {
			t.Error("T ~ int should be satisfied by int64")
		}

		if tp.SatisfiedBy(Basics[UTF8]) {
			t.Error("T ~ int should not be satisfied by utf8")
		}
	})

	t.Run("satisfied_by_multi_constraint_union", func(t *testing.T) {
		t.Parallel()

		tp := &TypeParam{
			Name:       "T",
			Constraint: &Union{Variants: []Type{Generics["string"], Generics["int"]}},
		}
		if !tp.SatisfiedBy(Basics[Int64]) {
			t.Error("T ~ string | int should be satisfied by int64")
		}

		if !tp.SatisfiedBy(Basics[UTF8]) {
			t.Error("T ~ string | int should be satisfied by utf8")
		}

		if tp.SatisfiedBy(Basics[Float64]) {
			t.Error("T ~ string | int should not be satisfied by float64")
		}
	})

	t.Run("underlying", func(t *testing.T) {
		t.Parallel()

		tp := &TypeParam{Name: "T", Constraint: Any}
		if tp.Underlying() != tp {
			t.Error("TypeParam.Underlying() should return itself")
		}
	})
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

		// ordered constraint (int + uint + float + string, no complex)
		{"int64 satisfies ordered", Basics[Int64], Generics["ordered"], true},
		{"uint32 satisfies ordered", Basics[Uint32], Generics["ordered"], true},
		{"float64 satisfies ordered", Basics[Float64], Generics["ordered"], true},
		{"ascii satisfies ordered", Basics[ASCII], Generics["ordered"], true},
		{"utf8 satisfies ordered", Basics[UTF8], Generics["ordered"], true},
		{"complex64 not ordered", Basics[Complex64], Generics["ordered"], false},
		{"bool not ordered", Basics[Bool], Generics["ordered"], false},

		// summable constraint (number + string)
		{"int64 satisfies summable", Basics[Int64], Generics["summable"], true},
		{"uint32 satisfies summable", Basics[Uint32], Generics["summable"], true},
		{"float64 satisfies summable", Basics[Float64], Generics["summable"], true},
		{"complex128 satisfies summable", Basics[Complex128], Generics["summable"], true},
		{"ascii satisfies summable", Basics[ASCII], Generics["summable"], true},
		{"utf8 satisfies summable", Basics[UTF8], Generics["summable"], true},
		{"bool not summable", Basics[Bool], Generics["summable"], false},

		// comparable constraint
		{"int64 satisfies comparable", Basics[Int64], Generics["comparable"], true},
		{"bool satisfies comparable", Basics[Bool], Generics["comparable"], true},
		{"complex128 satisfies comparable", Basics[Complex128], Generics["comparable"], true},
		{"utf8 satisfies comparable", Basics[UTF8], Generics["comparable"], true},
		{"struct satisfies comparable", &Struct{}, Generics["comparable"], true},
		{"array satisfies comparable", &Array{Element: Basics[Int64]}, Generics["comparable"], true},
		{"enum satisfies comparable", &Enum{ValueType: Basics[Int64]}, Generics["comparable"], true},
		{"pointer satisfies comparable", &Reference{Value: Basics[Int64]}, Generics["comparable"], true},
		{"tuple satisfies comparable", &Tuple{Types: []Type{Basics[Int64]}}, Generics["comparable"], true},
		{"set satisfies comparable", &Set{Element: Basics[Int64]}, Generics["comparable"], true},
		{"slice not comparable", &Slice{Element: Basics[Int64]}, Generics["comparable"], false},
		{"map not comparable", &Map{Key: Basics[UTF8], Value: Basics[Int64]}, Generics["comparable"], false},

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

func TestTypeParamEquality(t *testing.T) {
	t.Parallel()

	tp1 := &TypeParam{Name: "T", Constraint: Generics["int"]}
	tp2 := &TypeParam{Name: "T", Constraint: Generics["int"]}
	tp3 := &TypeParam{Name: "U", Constraint: Generics["int"]}
	tp4 := &TypeParam{Name: "T", Constraint: Generics["uint"]}

	if !Equal(tp1, tp2) {
		t.Error("same name, same constraints should be equal")
	}

	if Equal(tp1, tp3) {
		t.Error("different name should not be equal")
	}

	if Equal(tp1, tp4) {
		t.Error("different constraints should not be equal")
	}
}

func TestComparableRejectsNonComparableStruct(t *testing.T) {
	t.Parallel()

	// A struct with a slice field should NOT satisfy comparable.
	s := &Struct{Fields: []*Field{{Name: "data", Type: &Slice{Element: Basics[Int64]}}}}
	if Satisfies(s, Generics["comparable"]) {
		t.Error("struct with slice field should not satisfy comparable")
	}

	// A struct with all comparable fields should satisfy comparable.
	s2 := &Struct{Fields: []*Field{{Name: "x", Type: Basics[Int64]}, {Name: "y", Type: Basics[UTF8]}}}
	if !Satisfies(s2, Generics["comparable"]) {
		t.Error("struct with comparable fields should satisfy comparable")
	}
}
