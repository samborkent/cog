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

	t.Run("comment", func(t *testing.T) {
		t.Parallel()
		got := transpile(t, `package p
main : proc() = {
	// this is a comment
	@print("hello")
}`)
		mustContain(t, got, "comment")
	})

	t.Run("for_condition_manual", func(t *testing.T) {
		t.Parallel()
		got := transpile(t, `package p
main : proc() = {
	var x := 0
	for {
		if x >= 10 {
			break
		}
		x = x + 1
	}
}`)
		mustContain(t, got, "for {")
	})

	t.Run("continue", func(t *testing.T) {
		t.Parallel()
		got := transpile(t, `package p
main : proc() = {
	xs := @slice<int64>(3)
	for v in xs {
		if v == 0 {
			continue
		}
		@print(v)
	}
}`)
		mustContain(t, got, "continue")
	})

	t.Run("if_else_nested", func(t *testing.T) {
		t.Parallel()
		got := transpile(t, `package p
main : proc() = {
	x := 5
	if x > 10 {
		@print("big")
	} else {
		if x > 0 {
			@print("small")
		} else {
			@print("zero")
		}
	}
}`)
		mustContain(t, got, "} else {")
	})

	t.Run("enum_access", func(t *testing.T) {
		t.Parallel()
		got := transpile(t, `package p
Status ~ enum<utf8> {
	Open := "open",
	Closed := "closed",
}
main : proc() = {
	s := Status.Open
	@print(s)
}`)
		mustContain(t, got, "Status")
		mustContain(t, got, "Open")
	})

	t.Run("generic_func_call_in_proc", func(t *testing.T) {
		t.Parallel()
		got := transpile(t, `package p
identity : func<T ~ any>(x : T) T = {
	return x
}
main : proc() = {
	result := identity(42)
	@print(result)
}`)
		mustContain(t, got, "identity[int64]")
	})

	t.Run("pointer_type", func(t *testing.T) {
		t.Parallel()
		got := transpile(t, `package p
Point ~ struct {
	x : int32
	y : int32
}
main : proc() = {
	p := @ref<Point>()
	@print(p)
}`)
		mustContain(t, got, "new")
	})

	t.Run("option_alias", func(t *testing.T) {
		t.Parallel()
		got := transpile(t, `package p
MaybeStr ~ utf8?
main : proc() = {}`)
		mustContain(t, got, "type _MaybeStr")
	})

	t.Run("exported_proc", func(t *testing.T) {
		t.Parallel()
		got := transpile(t, `package p
export greet : proc() = {
	@print("hello")
}
main : proc() = {}`)
		mustContain(t, got, "func Greet")
	})
}
