package transpiler_test

import "testing"

func TestConvertDecl(t *testing.T) {
	t.Parallel()

	t.Run("const_int", func(t *testing.T) {
		t.Parallel()
		got := transpile(t, `package p
x := 42
main : proc() = {}`)
		mustContain(t, got, "const x int64 = 42")
	})

	t.Run("typed_int", func(t *testing.T) {
		t.Parallel()
		got := transpile(t, `package p
x : int64 = 42
main : proc() = {}`)
		mustContain(t, got, "const x int64 = 42")
	})

	t.Run("var_declaration", func(t *testing.T) {
		t.Parallel()
		got := transpile(t, `package p
main : proc() = {
	x : var = 1
	x = 2
	@print(x)
}`)
		mustContain(t, got, "var x int64 = 1")
	})

	t.Run("exported", func(t *testing.T) {
		t.Parallel()
		got := transpile(t, `package p
export myVal := 1
main : proc() = {}`)
		mustContain(t, got, "const MyVal int64 = 1")
	})

	t.Run("string_const", func(t *testing.T) {
		t.Parallel()
		got := transpile(t, `package p
s := "hello"
main : proc() = {}`)
		mustContain(t, got, `const s string = "hello"`)
	})

	t.Run("bool_const", func(t *testing.T) {
		t.Parallel()
		got := transpile(t, `package p
b := true
main : proc() = {}`)
		mustContain(t, got, "const b bool = true")
	})

	t.Run("dyn_skipped", func(t *testing.T) {
		t.Parallel()
		got := transpile(t, `package p
val : dyn utf8 = "default"
main : proc() = {}`)
		mustNotContain(t, got, "const val")
		mustNotContain(t, got, "var val")
	})

	t.Run("main_func", func(t *testing.T) {
		t.Parallel()
		got := transpile(t, `package p
main : proc() = {}`)
		mustContain(t, got, "func main()")
	})

	t.Run("proc_declaration", func(t *testing.T) {
		t.Parallel()
		got := transpile(t, `package p
greet : proc(name : utf8) = {
	@print(name)
}
main : proc() = {}`)
		mustContain(t, got, "func greet(ctx go_context.Context, name string)")
	})

	t.Run("func_declaration", func(t *testing.T) {
		t.Parallel()
		got := transpile(t, `package p
add : func(a : int64, b : int64) int64 = {
	return a + b
}
main : proc() = {}`)
		mustContain(t, got, "func add(a int64, b int64) int64")
		mustContain(t, got, "return a + b")
	})

	t.Run("proc_no_dyn_no_preamble", func(t *testing.T) {
		t.Parallel()
		got := transpile(t, `package p
val : dyn utf8 = "default"
noop : proc() = {
	@print("hello")
}
main : proc() = {}`)
		mustNotContain(t, got, "dyn := *ctx.Value")
	})

	t.Run("proc_uses_dyn_has_preamble", func(t *testing.T) {
		t.Parallel()
		got := transpile(t, `package p
val : dyn utf8 = "default"
reader : proc() = {
	@print(val)
}
main : proc() = {}`)
		mustContain(t, got, "dyn := *ctx.Value(cogDynKey{}).(*cogDyn)")
		mustContain(t, got, "ctx = go_context.WithValue(ctx, cogDynKey{}, &dyn)")
	})

	t.Run("func_no_dyn_no_ctx", func(t *testing.T) {
		t.Parallel()
		got := transpile(t, `package p
val : dyn utf8 = "default"
add : func(a : int64, b : int64) int64 = {
	return a + b
}
main : proc() = {}`)
		mustNotContain(t, got, "go_context.Context")
	})
}
