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

	t.Run("valid_in_proc", func(t *testing.T) {
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

	t.Run("inside_func_errors", func(t *testing.T) {
		t.Parallel()
		parseShouldError(t, `package p
goimport (
	"strings"
)
myFunc : func() utf8 = {
	result := @go.strings.ToUpper("hello")
	return result
}`)
	})

	t.Run("inside_nested_func_errors", func(t *testing.T) {
		t.Parallel()
		parseShouldError(t, `package p
goimport (
	"strings"
)
main : proc() = {
	helper := func() utf8 = {
		return @go.strings.ToUpper("hello")
	}
	@print(helper())
}`)
	})

	t.Run("inside_func_parameter_errors", func(t *testing.T) {
		t.Parallel()
		parseShouldError(t, `package p
goimport (
	"strings"
)
myFunc : func(s utf8) utf8 = {
	result := @go.strings.ToUpper(s)
	return result
}`)
	})

	t.Run("inside_proc_allowed", func(t *testing.T) {
		t.Parallel()

		parse(t, `package p
goimport (
	"strings"
)
main : proc() = {
	x := @go.strings.ToUpper("hello")
	@print(x)
}`)
	})

	t.Run("top_level_allowed", func(t *testing.T) {
		t.Parallel()

		parse(t, `package p
goimport (
	"strings"
)
x : utf8 = @go.strings.ToUpper("hello")
main : proc() = {
	@print(x)
}`)
	})
}
