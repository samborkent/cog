package parser_test

import (
	"testing"

	"github.com/samborkent/cog/internal/ast"
	"github.com/samborkent/cog/internal/types"
)

func TestParseLiteral(t *testing.T) {
	t.Parallel()

	t.Run("struct", func(t *testing.T) {
		t.Parallel()

		f := parse(t, `package p
Point ~ struct {
	x : int32
	y : int32
}
main : proc() = {
	p : Point = {
		x = 1,
		y = 2,
	}
	@print(p)
}`)
		if f.LenNodes() == 0 {
			t.Fatal("expected statements")
		}
	})

	t.Run("array", func(t *testing.T) {
		t.Parallel()
		f := parse(t, `package p
arr : [3]int64 = {1, 2, 3}
main : proc() = {}`)

		d := stmtAs[*ast.Declaration](t, f, 0)
		if d.Assignment.Identifier.ValueType.Kind() != types.ArrayKind {
			t.Errorf("expected ArrayKind, got %s", d.Assignment.Identifier.ValueType.Kind())
		}
	})

	t.Run("slice", func(t *testing.T) {
		t.Parallel()
		f := parse(t, `package p
xs : []utf8 = {"foo", "bar"}
main : proc() = {}`)

		d := stmtAs[*ast.Declaration](t, f, 0)
		if d.Assignment.Identifier.ValueType.Kind() != types.SliceKind {
			t.Errorf("expected SliceKind, got %s", d.Assignment.Identifier.ValueType.Kind())
		}
	})

	t.Run("map", func(t *testing.T) {
		t.Parallel()

		f := parse(t, `package p
M ~ map<utf8, int64>
m : M = {"a": 1, "b": 2}
main : proc() = {}`)
		if f.LenNodes() < 2 {
			t.Fatal("expected at least 2 statements")
		}
	})
}

func TestParseTypedLiterals(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		src  string
		kind types.Kind
	}{
		{"int8", `x : int8 = 1`, types.Int8},
		{"int16", `x : int16 = 1`, types.Int16},
		{"int32", `x : int32 = 1`, types.Int32},
		{"int64", `x : int64 = 1`, types.Int64},
		{"int128", `x : int128 = 1`, types.Int128},
		{"uint8", `x : uint8 = 1`, types.Uint8},
		{"uint16", `x : uint16 = 1`, types.Uint16},
		{"uint32", `x : uint32 = 1`, types.Uint32},
		{"uint64", `x : uint64 = 1`, types.Uint64},
		{"uint128", `x : uint128 = 1`, types.Uint128},
		{"float16", `x : float16 = 1.0`, types.Float16},
		{"float32", `x : float32 = 1.0`, types.Float32},
		{"float64", `x : float64 = 1.0`, types.Float64},
		{"bool_true", `x : bool = true`, types.Bool},
		{"bool_false", `x : bool = false`, types.Bool},
		{"ascii", `x : ascii = "hello"`, types.ASCII},
		{"utf8", `x : utf8 = "hello"`, types.UTF8},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			f := parse(t, "package p\nmain : proc() = {\n"+tt.src+"\n}")
			if f.LenNodes() == 0 {
				t.Fatal("expected statements")
			}
		})
	}
}

func TestParseInferredLiterals(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		src  string
	}{
		{"inferred_int", `x := 42`},
		{"inferred_float", `x := 3.14`},
		{"inferred_string", `x := "hello"`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			f := parse(t, "package p\nmain : proc() = {\n"+tt.src+"\n}")
			if f.LenNodes() == 0 {
				t.Fatal("expected statements")
			}
		})
	}
}

func TestParseArithmeticExpressions(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		src  string
	}{
		{"add", `x : int64 = 1 + 2`},
		{"sub", `x : int64 = 5 - 3`},
		{"mul", `x : int64 = 2 * 3`},
		{"div", `x : int64 = 6 / 2`},
		{"comparison_lt", `x := 1 < 2`},
		{"comparison_gt", `x := 2 > 1`},
		{"comparison_eq", `x := 1 == 1`},
		{"comparison_neq", `x := 1 != 2`},
		{"comparison_lte", `x := 1 <= 2`},
		{"comparison_gte", `x := 2 >= 1`},
		{"boolean_and", `x : bool = true && false`},
		{"boolean_or", `x : bool = true || false`},
		{"unary_not", `x : bool = !true`},
		{"unary_neg", `x : int64 = -5`},
		{"string_concat", `x : utf8 = "hello" + " world"`},
		{"complex_expr", `x : int64 = (1 + 2) * 3 - 4 / 2`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			f := parse(t, "package p\nmain : proc() = {\n"+tt.src+"\n}")
			if f.LenNodes() == 0 {
				t.Fatal("expected statements")
			}
		})
	}
}
