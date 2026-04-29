package parser_test

import "testing"

func TestParseForStatement(t *testing.T) {
	t.Parallel()

	t.Run("infinite", func(t *testing.T) {
		t.Parallel()

		f := parse(t, `package p
main : proc() = {
	for {
		break
	}
}`)
		if f.LenNodes() == 0 {
			t.Fatal("expected statements")
		}
	})

	t.Run("for_in", func(t *testing.T) {
		t.Parallel()

		f := parse(t, `package p
main : proc() = {
	xs := @slice<int64>(3)
	for v in xs {
		@print(v)
	}
}`)
		if f.LenNodes() == 0 {
			t.Fatal("expected statements")
		}
	})
}
