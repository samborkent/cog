package parser_test

import "testing"

func TestParseBoolSwitch(t *testing.T) {
	t.Parallel()

	t.Run("with_default", func(t *testing.T) {
		t.Parallel()

		f := parse(t, `package p
main : proc() = {
	switch {
	case true:
		@print("yes")
	default:
		@print("no")
	}
}`)
		if len(f.Statements) == 0 {
			t.Fatal("expected statements")
		}
	})
}

func TestParseIdentSwitch(t *testing.T) {
	t.Parallel()

	t.Run("with_default", func(t *testing.T) {
		t.Parallel()

		f := parse(t, `package p
main : proc() = {
	x := 1
	switch x {
	case 1:
		@print("one")
	default:
		@print("other")
	}
}`)
		if len(f.Statements) == 0 {
			t.Fatal("expected statements")
		}
	})
}
