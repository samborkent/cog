package parser_test

import "testing"

func TestParseBlockStatement(t *testing.T) {
	t.Parallel()

	t.Run("scoping", func(t *testing.T) {
		t.Parallel()
		f := parse(t, `package p
main : proc() = {
	if true {
		var x := 1
		x = 2
	}
}`)
		if len(f.Statements) == 0 {
			t.Fatal("expected statements")
		}
	})
}
