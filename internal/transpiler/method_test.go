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
}
