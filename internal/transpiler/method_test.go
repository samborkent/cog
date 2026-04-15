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
Foo.GetValue : func() utf8 = {
	return this.value
}
main : proc() = {}`)

		mustContain(t, got, "func (this _Foo) _GetValue() string")
		mustContain(t, got, "return this.value")
	})

	t.Run("this_multiple_fields", func(t *testing.T) {
		t.Parallel()

		got := transpile(t, `package p
Point ~ struct {
	x : int64
	y : int64
}
Point.Sum : func() int64 = {
	return this.x + this.y
}
main : proc() = {}`)

		mustContain(t, got, "func (this _Point) _Sum() int64")
		mustContain(t, got, "this.x")
		mustContain(t, got, "this.y")
	})

	t.Run("exported_method", func(t *testing.T) {
		t.Parallel()

		got := transpile(t, `package p
export Foo ~ struct {
	value : utf8
}
export Foo.GetValue : func() utf8 = {
	return this.value
}
main : proc() = {}`)

		mustContain(t, got, "func (this Foo) GetValue() string")
	})

	t.Run("reference_receiver", func(t *testing.T) {
		t.Parallel()

		got := transpile(t, `package p
Counter ~ struct {}
&Counter.Increment : proc() = {}
main : proc() = {}`)

		mustContain(t, got, "func (this *_Counter) _Increment(")
	})

	t.Run("method_with_params", func(t *testing.T) {
		t.Parallel()

		got := transpile(t, `package p
Adder ~ struct {
	base : int64
}
Adder.Add : func(n : int64) int64 = {
	return this.base + n
}
main : proc() = {}`)

		mustContain(t, got, "func (this _Adder) _Add(n int64) int64")
		mustContain(t, got, "this.base + n")
	})

	t.Run("method_proc_no_return", func(t *testing.T) {
		t.Parallel()

		got := transpile(t, `package p
Foo ~ struct {
	name : utf8
}
Foo.Greet : proc() = {
	@print(this.name)
}
main : proc() = {}`)

		mustContain(t, got, "func (this _Foo) _Greet(")
		mustContain(t, got, "this.name")
	})
}
