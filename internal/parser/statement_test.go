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
}
