package parser_test

import "testing"

func TestParseIfStatement(t *testing.T) {
	t.Parallel()

	t.Run("simple", func(t *testing.T) {
		t.Parallel()
		f := parse(t, `package p
main : proc() = {
	if true {
		@print("yes")
	}
}`)
		if len(f.Statements) == 0 {
			t.Fatal("expected statements")
		}
	})

	t.Run("if_else", func(t *testing.T) {
		t.Parallel()
		f := parse(t, `package p
main : proc() = {
	if true {
		@print("yes")
	} else {
		@print("no")
	}
}`)
		if len(f.Statements) == 0 {
			t.Fatal("expected statements")
		}
	})
}
