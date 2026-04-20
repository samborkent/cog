package parser_test

import "testing"

func TestParseStatement(t *testing.T) {
	t.Parallel()

	t.Run("return", func(t *testing.T) {
		t.Parallel()

		f := parse(t, `package p
add : func(a : int64, b : int64) int64 = {
	return a + b
}
main : proc() = {}`)
		if len(f.Statements) < 2 {
			t.Fatal("expected at least 2 statements")
		}
	})

	t.Run("selector_method_call", func(t *testing.T) {
		t.Parallel()

		// Selector method calls like obj.Method() must not be
		// confused with selector assignment (obj.field = val).
		f := parse(t, `package p
Foo ~ struct { value : utf8 }
Foo.Greet : proc() = {
	@print("hi")
}
main : proc() = {
	var f := Foo{ value = "hello" }
	f.Greet()
}`)
		if len(f.Statements) < 3 {
			t.Fatal("expected at least 3 statements")
		}
	})
}
