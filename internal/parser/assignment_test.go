package parser_test

import "testing"

func TestParseAssignment(t *testing.T) {
	t.Parallel()

	t.Run("valid", func(t *testing.T) {
		t.Parallel()
		f := parse(t, `package p
main : proc() = {
	var x := 1
	x = 2
}`)
		if len(f.Statements) == 0 {
			t.Fatal("expected statements")
		}
	})

	t.Run("immutable_error", func(t *testing.T) {
		t.Parallel()
		parseShouldError(t, `package p
main : proc() = {
	x := 1
	x = 2
}`)
	})

	t.Run("undefined_error", func(t *testing.T) {
		t.Parallel()
		parseShouldError(t, `package p
main : proc() = {
	x = 1
}`)
	})
}
