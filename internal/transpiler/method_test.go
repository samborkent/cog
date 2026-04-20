package transpiler_test

import "testing"

func TestConvertMethod(t *testing.T) {
	t.Parallel()

	t.Run("this_field_access", func(t *testing.T) {
		t.Parallel()

		got := transpile(t, `package p
Foo ~ struct {
	value : utf8
}
(f : Foo).GetValue : func() utf8 = {
	return f.value
}
main : proc() = {}`)

		mustContain(t, got, "func (f _Foo) _GetValue() string")
		mustContain(t, got, "return f.value")
	})

	t.Run("this_multiple_fields", func(t *testing.T) {
		t.Parallel()

		got := transpile(t, `package p
Point ~ struct {
	x : int64
	y : int64
}
(p : Point).Sum : func() int64 = {
	return p.x + p.y
}
main : proc() = {}`)

		mustContain(t, got, "func (p _Point) _Sum() int64")
		mustContain(t, got, "p.x")
		mustContain(t, got, "p.y")
	})

	t.Run("exported_method", func(t *testing.T) {
		t.Parallel()

		got := transpile(t, `package p
export Foo ~ struct {
	value : utf8
}
export (f : Foo).GetValue : func() utf8 = {
	return f.value
}
main : proc() = {}`)

		mustContain(t, got, "func (f Foo) GetValue() string")
	})

	t.Run("reference_receiver", func(t *testing.T) {
		t.Parallel()

		got := transpile(t, `package p
Counter ~ struct {}
(c : &Counter).Increment : proc() = {}
main : proc() = {}`)

		mustContain(t, got, "func (_ *_Counter) _Increment(")
	})

	t.Run("method_with_params", func(t *testing.T) {
		t.Parallel()

		got := transpile(t, `package p
Adder ~ struct {
	base : int64
}
(a : Adder).Add : func(n : int64) int64 = {
	return a.base + n
}
main : proc() = {}`)

		mustContain(t, got, "func (a _Adder) _Add(n int64) int64")
		mustContain(t, got, "a.base + n")
	})

	t.Run("method_proc_no_return", func(t *testing.T) {
		t.Parallel()

		got := transpile(t, `package p
Foo ~ struct {
	name : utf8
}
(f : Foo).Greet : proc() = {
	@print(f.name)
}
main : proc() = {}`)

		mustContain(t, got, "func (f _Foo) _Greet(")
		mustContain(t, got, "f.name")
	})
}
