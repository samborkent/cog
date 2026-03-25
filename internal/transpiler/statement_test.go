package transpiler_test

import "testing"

func TestConvertStmt(t *testing.T) {
	t.Parallel()

	t.Run("assignment", func(t *testing.T) {
		t.Parallel()
		got := transpile(t, `package p
main : proc() = {
	var x := 1
	x = 2
	@print(x)
}`)
		mustContain(t, got, "x = 2")
	})

	t.Run("if_statement", func(t *testing.T) {
		t.Parallel()
		got := transpile(t, `package p
main : proc() = {
	if true {
		@print("yes")
	}
}`)
		mustContain(t, got, "if true")
	})

	t.Run("if_else", func(t *testing.T) {
		t.Parallel()
		got := transpile(t, `package p
main : proc() = {
	if true {
		@print("yes")
	} else {
		@print("no")
	}
}`)
		mustContain(t, got, "} else {")
	})

	t.Run("for_infinite", func(t *testing.T) {
		t.Parallel()
		got := transpile(t, `package p
main : proc() = {
	for {
		break
	}
}`)
		mustContain(t, got, "for {")
	})

	t.Run("for_in", func(t *testing.T) {
		t.Parallel()
		got := transpile(t, `package p
main : proc() = {
	xs := @slice<int64>(3)
	for v in xs {
		@print(v)
	}
}`)
		mustContain(t, got, "range")
	})

	t.Run("switch_bool", func(t *testing.T) {
		t.Parallel()
		got := transpile(t, `package p
main : proc() = {
	switch {
	case true:
		@print("yes")
	default:
		@print("no")
	}
}`)
		mustContain(t, got, "switch {")
	})

	t.Run("switch_ident", func(t *testing.T) {
		t.Parallel()
		got := transpile(t, `package p
main : proc() = {
	x := 1
	switch x {
	case 1:
		@print("one")
	default:
		@print("other")
	}
}`)
		mustContain(t, got, "switch x {")
	})

	t.Run("return", func(t *testing.T) {
		t.Parallel()
		got := transpile(t, `package p
add : func(a : int64, b : int64) int64 = {
	return a + b
}
main : proc() = {}`)
		mustContain(t, got, "return a + b")
	})

	t.Run("dyn_write", func(t *testing.T) {
		t.Parallel()
		got := transpile(t, `package p
dyn val : utf8 = "default"
writer : proc() = {
	val = "changed"
}
main : proc() = {}`)
		mustContain(t, got, "dyn.val =")
	})

	t.Run("dyn_write_in_func_errors", func(t *testing.T) {
		t.Parallel()
		mustFailTranspile(t, `package p
dyn val : utf8 = "default"
writer : func() utf8 = {
	val = "changed"
	return val
}
main : proc() = {}`, "func cannot assign dynamically scoped variable")
	})

	t.Run("local_declaration", func(t *testing.T) {
		t.Parallel()
		got := transpile(t, `package p
main : proc() = {
	x := 42
	@print(x)
}`)
		mustContain(t, got, "var x int64 = 42")
	})

	t.Run("labeled_break", func(t *testing.T) {
		t.Parallel()
		got := transpile(t, `package p
main : proc() = {
	var x := 0
outerLoop:
	for {
		x = x + 1
		break outerLoop
	}
}`)
		mustContain(t, got, "outerLoop")
	})
}
