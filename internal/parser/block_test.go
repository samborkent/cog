package parser_test

import "testing"

func TestParseBlockStatement(t *testing.T) {
	t.Parallel()

	t.Run("scoping", func(t *testing.T) {
		t.Parallel()

		f := parse(t, `package p
main : proc() = {
	if true {
		x : var = 1
		x = 2
		_ = x
	}
}`)
		if f.LenNodes() == 0 {
			t.Fatal("expected statements")
		}
	})
}
