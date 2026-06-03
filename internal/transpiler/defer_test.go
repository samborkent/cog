package transpiler_test

import (
	"strings"
	"testing"
)

func TestDeferTranspile(t *testing.T) {
	t.Run("basic_defer_call", func(t *testing.T) {
		got := transpile(t, `package p
close : proc(f : int32) = {}
main : proc() = {
	defer close(1)
	@print("hi")
}`)
		mustContain(t, got, "close(ctx, 1)")
	})

	t.Run("defer_before_implicit_return", func(t *testing.T) {
		got := transpile(t, `package p
close : proc(f : int32) = {}
main : proc() = {
	defer close(1)
}`)
		mustContain(t, got, "close(ctx, 1)")
	})

	t.Run("multiple_defers_lifo", func(t *testing.T) {
		got := transpile(t, `package p
a : proc() = {}
b : proc() = {}
c : proc() = {}
main : proc() = {
	defer a()
	defer b()
	defer c()
}`)
		aIdx := strings.Index(got, "a(ctx)")
		bIdx := strings.Index(got, "b(ctx)")
		cIdx := strings.Index(got, "c(ctx)")
		if !(cIdx < bIdx && bIdx < aIdx) {
			t.Errorf("expected LIFO order c < b < a, got c=%d b=%d a=%d", cIdx, bIdx, aIdx)
		}
	})

	t.Run("defer_in_if_branch", func(t *testing.T) {
		got := transpile(t, `package p
cleanup : proc() = {}
main : proc() = {
	if true {
		defer cleanup()
	}
}`)
		mustContain(t, got, "cleanup(ctx)")
	})

	t.Run("defer_multiple_returns", func(t *testing.T) {
		got := transpile(t, `package p
cleanup : proc() = {}
getResult : func() bool = { return true }
main : proc() = {
	defer cleanup()
	if getResult() {
		return true
	}
	return false
}`)
		mustContain(t, got, "cleanup(ctx)")
	})

	t.Run("defer_no_return_void", func(t *testing.T) {
		got := transpile(t, `package p
cleanup : proc() = {}
main : proc() = {
	defer cleanup()
	@print("done")
}`)
		mustContain(t, got, "cleanup(ctx)")
	})

	t.Run("defer_inside_for_loop_injected", func(t *testing.T) {
		got := transpile(t, `package p
cleanup : proc(x : int32) = {}
main : proc() = {
	for {
		defer cleanup(1)
		break
	}
}`)
		mustContain(t, got, "cleanup(ctx,")
		mustNotContain(t, got, "defer cleanup")
	})

	t.Run("defer_switch_case", func(t *testing.T) {
		got := transpile(t, `package p
cleanup : proc() = {}
main : proc() = {
	x := true
	defer cleanup()
	switch x {
	case true:
		@print("ok")
	}
}`)
		mustContain(t, got, "cleanup(ctx)")
	})

	t.Run("defer_nested_function", func(t *testing.T) {
		got := transpile(t, `package p
a : proc() = {}
outerFn : proc() = {
	defer a()
	@print("hi")
}`)
		mustContain(t, got, "a(ctx)")
	})

	t.Run("defer_method", func(t *testing.T) {
		got := transpile(t, `package p
T ~ struct { x : int32 }
Method : proc(r : &T) = {
	defer cleanup()
}
cleanup : proc() = {}`)
		mustContain(t, got, "cleanup(ctx)")
	})

	t.Run("zero_defers", func(t *testing.T) {
		got := transpile(t, `package p
main : proc() = {
	@print("no defer")
}`)
		mustContain(t, got, "Print(\"no defer\")")
	})
}