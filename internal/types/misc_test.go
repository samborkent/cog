package types

import (
	"testing"
)

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
		{"int64 to any", int64Type, Any, true},
		{"utf8 to any", utf8Type, Any, true},
		{"any to any", Any, Any, true},
		{"any to int64", Any, int64Type, false},
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

func TestSize(t *testing.T) {
	t.Parallel()

	tests := []struct {
		kind Kind
		want int
	}{
		{Bool, 8},
		{Int8, 8},
		{Uint8, 8},
		{Int16, 16},
		{Uint16, 16},
		{Float16, 16},
		{Int32, 32},
		{Uint32, 32},
		{Float32, 32},
		{Complex32, 32},
		{Int64, 64},
		{Uint64, 64},
		{Float64, 64},
		{Complex64, 64},
		{Int128, 128},
		{Uint128, 128},
		{Complex128, 128},
		{ASCII, -1},
		{UTF8, -1},
		{SliceKind, -1},
		{MapKind, -1},
	}

	for _, tt := range tests {
		t.Run(tt.kind.String(), func(t *testing.T) {
			t.Parallel()

			if got := Size(tt.kind); got != tt.want {
				t.Errorf("Size(%s) = %d, want %d", tt.kind, got, tt.want)
			}
		})
	}
}

func TestInstantiate(t *testing.T) {
	t.Parallel()

	t.Run("slice_of_T", func(t *testing.T) {
		t.Parallel()

		a := &Alias{
			Name:       "List",
			Derived:    &Slice{Element: &Alias{Name: "T", Constraint: Any}},
			TypeParams: []*Alias{{Name: "T", Constraint: Any}},
		}
		result := a.Instantiate(map[string]Type{"T": Basics[Int32]})

		s, ok := result.(*Slice)
		if !ok {
			t.Fatalf("expected *Slice, got %T", result)
		}

		if s.Element.Kind() != Int32 {
			t.Errorf("expected element Int32, got %s", s.Element.Kind())
		}
	})

	t.Run("map_K_V", func(t *testing.T) {
		t.Parallel()

		a := &Alias{
			Name: "Dict",
			Derived: &Map{
				Key:   &Alias{Name: "K", Constraint: Any},
				Value: &Alias{Name: "V", Constraint: Any},
			},
			TypeParams: []*Alias{
				{Name: "K", Constraint: Any},
				{Name: "V", Constraint: Any},
			},
		}
		result := a.Instantiate(map[string]Type{"K": Basics[UTF8], "V": Basics[Int64]})

		m, ok := result.(*Map)
		if !ok {
			t.Fatalf("expected *Map, got %T", result)
		}

		if m.Key.Kind() != UTF8 {
			t.Errorf("expected key UTF8, got %s", m.Key.Kind())
		}

		if m.Value.Kind() != Int64 {
			t.Errorf("expected value Int64, got %s", m.Value.Kind())
		}
	})

	t.Run("tuple", func(t *testing.T) {
		t.Parallel()

		a := &Alias{
			Name: "Pair",
			Derived: &Tuple{
				Types: []Type{
					&Alias{Name: "A", Constraint: Any},
					&Alias{Name: "B", Constraint: Any},
				},
			},
			TypeParams: []*Alias{
				{Name: "A", Constraint: Any},
				{Name: "B", Constraint: Any},
			},
		}
		result := a.Instantiate(map[string]Type{"A": Basics[Int32], "B": Basics[UTF8]})

		tup, ok := result.(*Tuple)
		if !ok {
			t.Fatalf("expected *Tuple, got %T", result)
		}

		if len(tup.Types) != 2 {
			t.Fatalf("expected 2 types, got %d", len(tup.Types))
		}

		if tup.Types[0].Kind() != Int32 {
			t.Errorf("expected first type Int32, got %s", tup.Types[0].Kind())
		}

		if tup.Types[1].Kind() != UTF8 {
			t.Errorf("expected second type UTF8, got %s", tup.Types[1].Kind())
		}
	})

	t.Run("option", func(t *testing.T) {
		t.Parallel()

		a := &Alias{
			Name:       "Maybe",
			Derived:    &Option{Value: &Alias{Name: "T", Constraint: Any}},
			TypeParams: []*Alias{{Name: "T", Constraint: Any}},
		}
		result := a.Instantiate(map[string]Type{"T": Basics[Float64]})

		opt, ok := result.(*Option)
		if !ok {
			t.Fatalf("expected *Option, got %T", result)
		}

		if opt.Value.Kind() != Float64 {
			t.Errorf("expected Float64, got %s", opt.Value.Kind())
		}
	})

	t.Run("basic_passthrough", func(t *testing.T) {
		t.Parallel()

		a := &Alias{
			Name:       "MyInt",
			Derived:    Basics[Int32],
			TypeParams: []*Alias{{Name: "T", Constraint: Any}},
		}

		result := a.Instantiate(map[string]Type{"T": Basics[UTF8]})
		if result.Kind() != Int32 {
			t.Errorf("expected Int32 passthrough, got %s", result.Kind())
		}
	})

	t.Run("substitute_procedure", func(t *testing.T) {
		t.Parallel()

		proc := &Procedure{
			Function: true,
			Parameters: []*Parameter{
				{Name: "x", Type: &Alias{Name: "T", Constraint: Any}},
			},
			ReturnType: &Alias{Name: "T", Constraint: Any},
		}
		a := &Alias{
			Name:       "MyFunc",
			Derived:    proc,
			TypeParams: []*Alias{{Name: "T", Constraint: Any}},
		}
		result := a.Instantiate(map[string]Type{"T": Basics[Int64]})

		rp, ok := result.(*Procedure)
		if !ok {
			t.Fatalf("expected *Procedure, got %T", result)
		}

		if !Equal(rp.Parameters[0].Type, Basics[Int64]) {
			t.Errorf("expected parameter type int64, got %s", rp.Parameters[0].Type)
		}

		if !Equal(rp.ReturnType, Basics[Int64]) {
			t.Errorf("expected return type int64, got %s", rp.ReturnType)
		}
	})
}
