package transpiler_test

import "testing"

func TestConvertBinaryOperator(t *testing.T) {
	t.Parallel()

	t.Run("comparison_gt", func(t *testing.T) {
		t.Parallel()
		got := transpile(t, `package p
main : proc() = {
	x := 5
	if x > 3 {
		@print("gt")
	}
}`)
		mustContain(t, got, ">")
	})

	t.Run("comparison_lt", func(t *testing.T) {
		t.Parallel()
		got := transpile(t, `package p
main : proc() = {
	x := 5
	if x < 3 {
		@print("lt")
	}
}`)
		mustContain(t, got, "<")
	})

	t.Run("comparison_gte", func(t *testing.T) {
		t.Parallel()
		got := transpile(t, `package p
main : proc() = {
	x := 5
	if x >= 3 {
		@print("gte")
	}
}`)
		mustContain(t, got, ">=")
	})

	t.Run("comparison_lte", func(t *testing.T) {
		t.Parallel()
		got := transpile(t, `package p
main : proc() = {
	x := 5
	if x <= 3 {
		@print("lte")
	}
}`)
		mustContain(t, got, "<=")
	})

	t.Run("equality_ne", func(t *testing.T) {
		t.Parallel()
		got := transpile(t, `package p
main : proc() = {
	x := 1
	if x != 2 {
		@print("ne")
	}
}`)
		mustContain(t, got, "!=")
	})

	t.Run("boolean_and", func(t *testing.T) {
		t.Parallel()
		got := transpile(t, `package p
main : proc() = {
	if true && false {
		@print("and")
	}
}`)
		mustContain(t, got, "&&")
	})

	t.Run("boolean_or", func(t *testing.T) {
		t.Parallel()
		got := transpile(t, `package p
main : proc() = {
	if true || false {
		@print("or")
	}
}`)
		mustContain(t, got, "||")
	})

	t.Run("multiply", func(t *testing.T) {
		t.Parallel()
		got := transpile(t, `package p
main : proc() = {
	x := 3 * 4
	@print(x)
}`)
		mustContain(t, got, "*")
	})

	t.Run("divide", func(t *testing.T) {
		t.Parallel()
		got := transpile(t, `package p
main : proc() = {
	x := 10 / 2
	@print(x)
}`)
		mustContain(t, got, "/")
	})

	t.Run("subtract", func(t *testing.T) {
		t.Parallel()
		got := transpile(t, `package p
main : proc() = {
	x := 10 - 3
	@print(x)
}`)
		mustContain(t, got, "-")
	})
}

func TestConvertUnaryOperator(t *testing.T) {
	t.Parallel()

	t.Run("negation", func(t *testing.T) {
		t.Parallel()
		got := transpile(t, `package p
main : proc() = {
	x := -42
	@print(x)
}`)
		mustContain(t, got, "-42")
	})

	t.Run("not", func(t *testing.T) {
		t.Parallel()
		got := transpile(t, `package p
main : proc() = {
	x := !true
	@print(x)
}`)
		mustContain(t, got, "!")
	})
}

func TestConvertReassignment(t *testing.T) {
	t.Parallel()

	t.Run("variable_reassign", func(t *testing.T) {
		t.Parallel()
		got := transpile(t, `package p
main : proc() = {
	var x : int64 = 0
	x = x + 1
	@print(x)
}`)
		mustContain(t, got, "x + 1")
	})
}

func TestConvertIndex(t *testing.T) {
	t.Parallel()

	got := transpile(t, `package p
main : proc() = {
	xs : []int64 = {1, 2, 3}
	x := xs[0]
	@print(x)
}`)
	mustContain(t, got, "[0]")
}

func TestConvertSwitch(t *testing.T) {
	t.Parallel()

	got := transpile(t, `package p
main : proc() = {
	x := 1
	switch x {
	case 1:
		@print("one")
	case 2:
		@print("two")
	}
}`)
	mustContain(t, got, "switch")
	mustContain(t, got, "case")
}

func TestConvertForRange(t *testing.T) {
	t.Parallel()

	got := transpile(t, `package p
main : proc() = {
	xs := @slice<int64>(3)
	for v in xs {
		@print(v)
	}
}`)
	mustContain(t, got, "range")
}

func TestConvertForRangeIndex(t *testing.T) {
	t.Parallel()

	got := transpile(t, `package p
main : proc() = {
	xs := @slice<int64>(3)
	for v, i in xs {
		@print(i)
		@print(v)
	}
}`)
	mustContain(t, got, "range")
}

func TestConvertEnum(t *testing.T) {
	t.Parallel()

	got := transpile(t, `package p
Color ~ enum<utf8> {
	Red := "red",
	Blue := "blue",
}
main : proc() = {
	c := Color.Red
	@print(c)
}`)
	mustContain(t, got, "Color")
	mustContain(t, got, "Red")
}

func TestConvertMapBuiltin(t *testing.T) {
	t.Parallel()

	t.Run("with_capacity", func(t *testing.T) {
		t.Parallel()
		got := transpile(t, `package p
main : proc() = {
	m := @map<utf8, int64>(10)
	@print(m)
}`)
		mustContain(t, got, "make(map[string]int64,")
	})
}

func TestConvertSetBuiltin(t *testing.T) {
	t.Parallel()

	t.Run("with_capacity", func(t *testing.T) {
		t.Parallel()
		got := transpile(t, `package p
main : proc() = {
	s := @set<int64>(5)
	@print(s)
}`)
		mustContain(t, got, "make(cog.Set[int64],")
	})
}
