package parser_test

import (
	"testing"

	"github.com/samborkent/cog/internal/ast"
)

func TestExpressionExtended(t *testing.T) {
	t.Parallel()

	t.Run("index_expression", func(t *testing.T) {
		t.Parallel()

		f := parse(t, `package p
main : proc() = {
	xs : []int64 = {1, 2, 3}
	x := xs[0]
	@print(x)
}`)
		if len(f.Statements) == 0 {
			t.Fatal("expected statements")
		}
	})

	t.Run("boolean_or", func(t *testing.T) {
		t.Parallel()

		f := parse(t, `package p
main : proc() = {
	if true || false {
		@print("or")
	}
}`)
		if len(f.Statements) == 0 {
			t.Fatal("expected statements")
		}
	})

	t.Run("equality_not_equal", func(t *testing.T) {
		t.Parallel()

		f := parse(t, `package p
main : proc() = {
	x := 1
	if x != 2 {
		@print("not equal")
	}
}`)
		if len(f.Statements) == 0 {
			t.Fatal("expected statements")
		}
	})

	t.Run("comparison_gt", func(t *testing.T) {
		t.Parallel()

		f := parse(t, `package p
main : proc() = {
	x := 5
	if x > 3 {
		@print("greater")
	}
}`)
		if len(f.Statements) == 0 {
			t.Fatal("expected statements")
		}
	})

	t.Run("comparison_lt", func(t *testing.T) {
		t.Parallel()

		f := parse(t, `package p
main : proc() = {
	x := 5
	if x < 10 {
		@print("less")
	}
}`)
		if len(f.Statements) == 0 {
			t.Fatal("expected statements")
		}
	})

	t.Run("comparison_gte", func(t *testing.T) {
		t.Parallel()

		f := parse(t, `package p
main : proc() = {
	x := 5
	if x >= 5 {
		@print("gte")
	}
}`)
		if len(f.Statements) == 0 {
			t.Fatal("expected statements")
		}
	})

	t.Run("comparison_lte", func(t *testing.T) {
		t.Parallel()

		f := parse(t, `package p
main : proc() = {
	x := 5
	if x <= 5 {
		@print("lte")
	}
}`)
		if len(f.Statements) == 0 {
			t.Fatal("expected statements")
		}
	})

	t.Run("term_addition", func(t *testing.T) {
		t.Parallel()

		f := parse(t, `package p
main : proc() = {
	x := 1 + 2 + 3
	@print(x)
}`)
		if len(f.Statements) == 0 {
			t.Fatal("expected statements")
		}
	})

	t.Run("term_subtraction", func(t *testing.T) {
		t.Parallel()

		f := parse(t, `package p
main : proc() = {
	x := 10 - 3
	@print(x)
}`)
		if len(f.Statements) == 0 {
			t.Fatal("expected statements")
		}
	})

	t.Run("factor_multiply", func(t *testing.T) {
		t.Parallel()

		f := parse(t, `package p
main : proc() = {
	x := 3 * 4
	@print(x)
}`)
		if len(f.Statements) == 0 {
			t.Fatal("expected statements")
		}
	})

	t.Run("factor_divide", func(t *testing.T) {
		t.Parallel()

		f := parse(t, `package p
main : proc() = {
	x := 10 / 2
	@print(x)
}`)
		if len(f.Statements) == 0 {
			t.Fatal("expected statements")
		}
	})

	t.Run("negation", func(t *testing.T) {
		t.Parallel()
		f := parse(t, `package p
main : proc() = {
	x := -42
	@print(x)
}`)

		d := stmtAs[*ast.Declaration](t, f, 0)

		if d.Assignment.Expr == ast.ZeroExpr {
			t.Fatal("expected expression in main body")
		}
	})

	t.Run("not_operator", func(t *testing.T) {
		t.Parallel()

		f := parse(t, `package p
main : proc() = {
	x := !true
	@print(x)
}`)
		if len(f.Statements) == 0 {
			t.Fatal("expected statements")
		}
	})

	t.Run("string_concatenation", func(t *testing.T) {
		t.Parallel()

		f := parse(t, `package p
main : proc() = {
	s := "hello" + " " + "world"
	@print(s)
}`)
		if len(f.Statements) == 0 {
			t.Fatal("expected statements")
		}
	})

	t.Run("complex_expression", func(t *testing.T) {
		t.Parallel()

		f := parse(t, `package p
main : proc() = {
	x := (1 + 2) * 3 - 4 / 2
	@print(x)
}`)
		if len(f.Statements) == 0 {
			t.Fatal("expected statements")
		}
	})

	t.Run("enum_dot_access", func(t *testing.T) {
		t.Parallel()

		f := parse(t, `package p
Color ~ enum<utf8> {
	Red := "red",
	Blue := "blue",
}
main : proc() = {
	c := Color.Red
	@print(c)
}`)
		if len(f.Statements) == 0 {
			t.Fatal("expected statements")
		}
	})

	t.Run("call_expression", func(t *testing.T) {
		t.Parallel()

		f := parse(t, `package p
add : func(a : int64, b : int64) int64 = {
	return a + b
}
main : proc() = {
	x := add(1, 2)
	@print(x)
}`)
		if len(f.Statements) == 0 {
			t.Fatal("expected statements")
		}
	})

	t.Run("nested_calls", func(t *testing.T) {
		t.Parallel()

		f := parse(t, `package p
double : func(x : int64) int64 = {
	return x * 2
}
main : proc() = {
	x := double(double(5))
	@print(x)
}`)
		if len(f.Statements) == 0 {
			t.Fatal("expected statements")
		}
	})

	t.Run("grouped_expression", func(t *testing.T) {
		t.Parallel()

		f := parse(t, `package p
main : proc() = {
	x := (1 + 2)
	@print(x)
}`)
		if len(f.Statements) == 0 {
			t.Fatal("expected statements")
		}
	})

	t.Run("bool_literal_false", func(t *testing.T) {
		t.Parallel()

		f := parse(t, `package p
main : proc() = {
	x := false
	@print(x)
}`)
		if len(f.Statements) == 0 {
			t.Fatal("expected statements")
		}
	})
}

func TestForStatementExtended(t *testing.T) {
	t.Parallel()

	t.Run("for_value_index", func(t *testing.T) {
		t.Parallel()

		f := parse(t, `package p
main : proc() = {
	xs := @slice<int64>(3)
	for v, i in xs {
		@print(i)
		@print(v)
	}
}`)
		if len(f.Statements) == 0 {
			t.Fatal("expected statements")
		}
	})

	t.Run("for_value_only", func(t *testing.T) {
		t.Parallel()

		f := parse(t, `package p
main : proc() = {
	xs := @slice<int64>(3)
	for v in xs {
		@print(v)
	}
}`)
		if len(f.Statements) == 0 {
			t.Fatal("expected statements")
		}
	})
}

func TestSwitchStatement(t *testing.T) {
	t.Parallel()

	t.Run("basic_switch", func(t *testing.T) {
		t.Parallel()

		f := parse(t, `package p
main : proc() = {
	x := 1
	switch x {
	case 1:
		@print("one")
	case 2:
		@print("two")
	}
}`)
		if len(f.Statements) == 0 {
			t.Fatal("expected statements")
		}
	})

	t.Run("bool_switch", func(t *testing.T) {
		t.Parallel()

		f := parse(t, `package p
main : proc() = {
	switch {
	case true:
		@print("yes")
	default:
		@print("no")
	}
}`)
		if len(f.Statements) == 0 {
			t.Fatal("expected statements")
		}
	})
}

func TestBuiltinExtended(t *testing.T) {
	t.Parallel()

	t.Run("map_with_capacity", func(t *testing.T) {
		t.Parallel()

		f := parse(t, `package p
main : proc() = {
	m := @map<utf8, int64>(10)
	@print(m)
}`)
		if len(f.Statements) == 0 {
			t.Fatal("expected statements")
		}
	})

	t.Run("set_with_capacity", func(t *testing.T) {
		t.Parallel()

		f := parse(t, `package p
main : proc() = {
	s := @set<int64>(5)
	@print(s)
}`)
		if len(f.Statements) == 0 {
			t.Fatal("expected statements")
		}
	})

	t.Run("print_two_args", func(t *testing.T) {
		t.Parallel()

		f := parse(t, `package p
main : proc() = {
	@print("x:")
	@print(1)
}`)
		if len(f.Statements) == 0 {
			t.Fatal("expected statements")
		}
	})

	t.Run("slice_with_capacity", func(t *testing.T) {
		t.Parallel()

		f := parse(t, `package p
main : proc() = {
	xs := @slice<utf8>(0, 10)
	@print(xs)
}`)
		if len(f.Statements) == 0 {
			t.Fatal("expected statements")
		}
	})

	t.Run("if_with_strings", func(t *testing.T) {
		t.Parallel()

		f := parse(t, `package p
main : proc() = {
	x := @if(true, "yes", "no")
	@print(x)
}`)
		if len(f.Statements) == 0 {
			t.Fatal("expected statements")
		}
	})
}
