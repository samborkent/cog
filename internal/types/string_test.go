package types

import "testing"

func TestKindString(t *testing.T) {
	t.Parallel()

	tests := []struct {
		kind Kind
		want string
	}{
		{ASCII, "ascii"},
		{Bool, "bool"},
		{Complex32, "complex32"},
		{Complex64, "complex64"},
		{Complex128, "complex128"},
		{Float16, "float16"},
		{Float32, "float32"},
		{Float64, "float64"},
		{Int8, "int8"},
		{Int16, "int16"},
		{Int32, "int32"},
		{Int64, "int64"},
		{Int128, "int128"},
		{Uint8, "uint8"},
		{Uint16, "uint16"},
		{Uint32, "uint32"},
		{Uint64, "uint64"},
		{Uint128, "uint128"},
		{UTF8, "utf8"},
		{ReferenceKind, "&"},
		{GenericKind, "generic"},
		{ArrayKind, "array"},
		{SliceKind, "slice"},
		{EnumKind, "enum"},
		{MapKind, "map"},
		{SetKind, "set"},
		{StructKind, "struct"},
		{TupleKind, "tuple"},
		{UnionKind, "union"},
		{OptionKind, "option"},
		{ProcedureKind, "proc"},
		{Invalid, ""},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			t.Parallel()

			if got := tt.kind.String(); got != tt.want {
				t.Errorf("Kind(%d).String() = %q, want %q", tt.kind, got, tt.want)
			}
		})
	}
}

func TestBasicString(t *testing.T) {
	t.Parallel()

	if s := Basics[Int64].String(); s != "int64" {
		t.Errorf("int64.String() = %q", s)
	}

	if s := None.String(); s != "" {
		t.Errorf("None.String() = %q", s)
	}
}

func TestSliceString(t *testing.T) {
	t.Parallel()

	s := &Slice{Element: Basics[Int64]}
	if got := s.String(); got != "[]int64" {
		t.Errorf("Slice.String() = %q, want []int64", got)
	}

	if s.Kind() != SliceKind {
		t.Error("Slice.Kind() != SliceKind")
	}

	if s.Underlying() != s {
		t.Error("Slice.Underlying() != self")
	}
}

func TestArrayString(t *testing.T) {
	t.Parallel()

	a := &Array{Element: Basics[UTF8], Length: mockExpr{str: "5"}}
	if got := a.String(); got != "[5]utf8" {
		t.Errorf("Array.String() = %q, want [5]utf8", got)
	}

	if a.Kind() != ArrayKind {
		t.Error("Array.Kind() != ArrayKind")
	}
}

func TestMapString(t *testing.T) {
	t.Parallel()

	m := &Map{Key: Basics[UTF8], Value: Basics[Int64]}
	if got := m.String(); got != "map[utf8]int64" {
		t.Errorf("Map.String() = %q, want map[utf8]int64", got)
	}

	if m.Kind() != MapKind {
		t.Error("Map.Kind() != MapKind")
	}
}

func TestSetString(t *testing.T) {
	t.Parallel()

	s := &Set{Element: Basics[ASCII]}
	if got := s.String(); got != "set[ascii]" {
		t.Errorf("Set.String() = %q, want set[ascii]", got)
	}

	if s.Kind() != SetKind {
		t.Error("Set.Kind() != SetKind")
	}
}

func TestOptionString(t *testing.T) {
	t.Parallel()

	o := &Option{Value: Basics[UTF8]}
	if got := o.String(); got != "utf8?" {
		t.Errorf("Option.String() = %q, want utf8?", got)
	}

	if o.Kind() != OptionKind {
		t.Error("Option.Kind() != OptionKind")
	}
}

func TestPointerString(t *testing.T) {
	t.Parallel()

	p := &Reference{Value: Basics[Int64]}

	if got := p.String(); got != "&int64" {
		t.Errorf("Reference.String() = %q, want &int64", got)
	}

	if p.Kind() != ReferenceKind {
		t.Error("Reference.Kind() != ReferenceKind")
	}
}

func TestTupleString(t *testing.T) {
	t.Parallel()

	tu := &Tuple{Types: []Type{Basics[UTF8], Basics[Int64]}}
	if got := tu.String(); got != "utf8 & int64" {
		t.Errorf("Tuple.String() = %q, want \"utf8 & int64\"", got)
	}

	if tu.Kind() != TupleKind {
		t.Error("Tuple.Kind() != TupleKind")
	}
}

func TestTupleIndex(t *testing.T) {
	t.Parallel()

	tu := &Tuple{Types: []Type{Basics[UTF8], Basics[Int64], Basics[Bool]}}
	if tu.Index(0) != Basics[UTF8] {
		t.Error("Tuple.Index(0) != utf8")
	}

	if tu.Index(2) != Basics[Bool] {
		t.Error("Tuple.Index(2) != bool")
	}
}

func TestTupleStringPanic(t *testing.T) {
	t.Parallel()

	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic for tuple with 1 type")
		}
	}()

	tu := &Tuple{Types: []Type{Basics[UTF8]}}
	_ = tu.String()
}

func TestUnionString(t *testing.T) {
	t.Parallel()

	u := &Union{Variants: []Type{Basics[UTF8], Basics[Int64]}}
	if got := u.String(); got != "utf8 | int64" {
		t.Errorf("Union.String() = %q, want \"utf8 | int64\"", got)
	}

	if u.Kind() != UnionKind {
		t.Error("Union.Kind() != UnionKind")
	}
}

func TestStructString(t *testing.T) {
	t.Parallel()

	t.Run("empty", func(t *testing.T) {
		s := &Struct{}
		if got := s.String(); got != "struct{}" {
			t.Errorf("empty Struct.String() = %q", got)
		}
	})

	t.Run("with fields", func(t *testing.T) {
		s := &Struct{Fields: []*Field{
			{Name: "x", Type: Basics[Float64], Exported: true},
			{Name: "y", Type: Basics[Float64]},
		}}

		got := s.String()
		if got != "struct {\nexport x : float64\ny : float64\n}" {
			t.Errorf("Struct.String() = %q", got)
		}
	})
}

func TestStructField(t *testing.T) {
	t.Parallel()

	s := &Struct{Fields: []*Field{
		{Name: "x", Type: Basics[Float64]},
		{Name: "y", Type: Basics[Int64]},
	}}

	if f := s.Field("x"); f == nil || f.Type != Basics[Float64] {
		t.Error("Struct.Field(x) failed")
	}

	if f := s.Field("z"); f != nil {
		t.Error("Struct.Field(z) should be nil")
	}
}

func TestEnumString(t *testing.T) {
	t.Parallel()

	t.Run("no values", func(t *testing.T) {
		e := &Enum{ValueType: Basics[UTF8]}
		if got := e.String(); got != "enum<utf8> {}" {
			t.Errorf("Enum.String() = %q", got)
		}
	})

	t.Run("with values", func(t *testing.T) {
		e := &Enum{
			ValueType: Basics[UTF8],
			Values: []*EnumValue{
				{Name: "Open", Value: mockExpr{str: `"open"`}},
			},
		}
		got := e.String()

		want := "enum<utf8> {\nOpen := \"open\",\n}"
		if got != want {
			t.Errorf("Enum.String() = %q, want %q", got, want)
		}
	})
}

func TestProcedureString(t *testing.T) {
	t.Parallel()

	t.Run("simple func", func(t *testing.T) {
		p := &Procedure{Function: true, ReturnType: Basics[Int64]}
		if got := p.String(); got != "func() int64" {
			t.Errorf("Procedure.String() = %q", got)
		}
	})

	t.Run("proc no return", func(t *testing.T) {
		p := &Procedure{Function: false}
		if got := p.String(); got != "proc()" {
			t.Errorf("Procedure.String() = %q", got)
		}
	})

	t.Run("with params", func(t *testing.T) {
		p := &Procedure{
			Function: true,
			Parameters: []*Parameter{
				{Name: "a", Type: Basics[UTF8]},
				{Name: "b", Type: Basics[Int64]},
			},
			ReturnType: Basics[Bool],
		}

		want := "func(a : utf8, b : int64) bool"
		if got := p.String(); got != want {
			t.Errorf("Procedure.String() = %q, want %q", got, want)
		}
	})

	t.Run("optional param", func(t *testing.T) {
		p := &Procedure{
			Function: true,
			Parameters: []*Parameter{
				{Name: "x", Type: Basics[UTF8], Optional: true},
			},
		}

		want := "func(x? : utf8)"
		if got := p.String(); got != want {
			t.Errorf("Procedure.String() = %q, want %q", got, want)
		}
	})

	t.Run("default param", func(t *testing.T) {
		p := &Procedure{
			Function: true,
			Parameters: []*Parameter{
				{Name: "x", Type: Basics[UTF8], Optional: true, Default: mockExpr{str: `"hi"`}},
			},
		}

		want := `func(x? : utf8 = "hi")`
		if got := p.String(); got != want {
			t.Errorf("Procedure.String() = %q, want %q", got, want)
		}
	})
}

func TestAliasResolution(t *testing.T) {
	t.Parallel()

	t.Run("simple alias", func(t *testing.T) {
		a := &Alias{Name: "MyInt", Derived: Basics[Int64]}
		if a.String() != "MyInt" {
			t.Errorf("Alias.String() = %q", a.String())
		}

		if a.Kind() != Int64 {
			t.Errorf("Alias.Kind() = %v, want Int64", a.Kind())
		}

		if a.Underlying() != Basics[Int64] {
			t.Error("Alias.Underlying() != int64")
		}
	})

	t.Run("forward alias", func(t *testing.T) {
		a := NewForwardAlias("Lazy", false, false, func() Type {
			return Basics[Float32]
		})
		// Before resolution, Derived is None.
		if a.Derived != None {
			t.Error("forward alias Derived should start as None")
		}
		// Kind triggers resolution.
		if a.Kind() != Float32 {
			t.Errorf("forward Alias.Kind() = %v, want Float32", a.Kind())
		}

		if a.Underlying() != Basics[Float32] {
			t.Error("forward Alias.Underlying() != float32")
		}
	})

	t.Run("nested alias", func(t *testing.T) {
		inner := &Alias{Name: "Inner", Derived: Basics[UTF8]}

		outer := &Alias{Name: "Outer", Derived: inner}
		if outer.Underlying() != Basics[UTF8] {
			t.Error("nested Alias.Underlying() != utf8")
		}

		if outer.Kind() != UTF8 {
			t.Errorf("nested Alias.Kind() = %v", outer.Kind())
		}
	})
}

func TestGeneric(t *testing.T) {
	t.Parallel()

	g := Generics["int"]
	if g.Kind() != GenericKind {
		t.Error("Generic.Kind() != GenericKind")
	}

	if g.String() != "int" {
		t.Errorf("Generic.String() = %q", g.String())
	}

	if g.Underlying() != g {
		t.Error("Generic.Underlying() != self")
	}
}

func TestIsHelpersThroughAlias(t *testing.T) {
	t.Parallel()

	alias := &Alias{Name: "MyInt", Derived: Basics[Int64]}
	if !IsInt(alias) {
		t.Error("IsInt(alias→int64) = false")
	}

	if !IsSigned(alias) {
		t.Error("IsSigned(alias→int64) = false")
	}

	if !IsNumber(alias) {
		t.Error("IsNumber(alias→int64) = false")
	}
}
