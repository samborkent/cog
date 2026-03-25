package transpiler_test

import "testing"

func TestConvertBuiltin(t *testing.T) {
	t.Parallel()

	t.Run("print", func(t *testing.T) {
		t.Parallel()
		got := transpile(t, `package p
main : proc() = {
	@print("hello")
}`)
		mustContain(t, got, "builtin.Print")
	})

	t.Run("if", func(t *testing.T) {
		t.Parallel()
		got := transpile(t, `package p
main : proc() = {
	x := @if(true, 1, 2)
	@print(x)
}`)
		mustContain(t, got, "builtin.If")
	})

	t.Run("slice", func(t *testing.T) {
		t.Parallel()
		got := transpile(t, `package p
main : proc() = {
	xs := @slice<int64>(3)
	@print(xs)
}`)
		mustContain(t, got, "make([]int64,")
	})

	t.Run("slice with capacity", func(t *testing.T) {
		t.Parallel()
		got := transpile(t, `package p
main : proc() = {
	xs := @slice<int64>(3, 10)
	@print(xs)
}`)
		mustContain(t, got, "make([]int64,")
	})

	t.Run("map", func(t *testing.T) {
		t.Parallel()
		got := transpile(t, `package p
main : proc() = {
	m := @map<utf8, int64>()
	@print(m)
}`)
		mustContain(t, got, "make(map[string]int64)")
	})

	t.Run("set", func(t *testing.T) {
		t.Parallel()
		got := transpile(t, `package p
main : proc() = {
	s := @set<int64>()
	@print(s)
}`)
		mustContain(t, got, "make(cog.Set[int64])")
	})

	t.Run("ptr", func(t *testing.T) {
		t.Parallel()
		got := transpile(t, `package p
main : proc() = {
	ptr := @ptr<utf8>()
	_ = ptr
}`)
		mustContain(t, got, "new(string)")
	})
}
