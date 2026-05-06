package parser_test

import "testing"

func TestParseGoImport(t *testing.T) {
	t.Parallel()

	t.Run("valid", func(t *testing.T) {
		t.Parallel()

		f := parse(t, `package p
goimport (
	"strings"
)
main : proc() = {}`)
		if f.LenNodes() < 1 {
			t.Fatal("expected at least goimport + main")
		}
	})
}

func TestParseGoCallExpression(t *testing.T) {
	t.Parallel()

	t.Run("valid", func(t *testing.T) {
		t.Parallel()

		f := parse(t, `package p
goimport (
	"strings"
)
main : proc() = {
	x := @go.strings.ToUpper("hello")
	@print(x)
}`)
		if f.LenNodes() == 0 {
			t.Fatal("expected statements")
		}
	})

	t.Run("undefined_import_error", func(t *testing.T) {
		t.Parallel()
		parseShouldError(t, `package p
main : proc() = {
	x := @go.strings.ToUpper("hello")
}`)
	})
}
