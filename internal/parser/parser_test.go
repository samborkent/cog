package parser_test

import (
	"testing"

	"github.com/samborkent/cog/internal/ast"
	"github.com/samborkent/cog/internal/parser"
	"github.com/samborkent/cog/internal/tokens"
)

func NewTestParser(t *testing.T, tokens []tokens.Token, debug bool) (*parser.Parser, error) {
	t.Helper()

	return parser.NewParserWithSymbols(tokens, parser.NewSymbolTable(), debug, "")
}

func TestParse(t *testing.T) {
	t.Parallel()

	t.Run("file_name", func(t *testing.T) {
		t.Parallel()

		f := parse(t, `package p
main : proc() = {}`)
		if f.Name != "test.cog" {
			t.Errorf("expected file name 'test.cog', got %q", f.Name)
		}
	})

	t.Run("forward_type_reference", func(t *testing.T) {
		t.Parallel()
		f := parse(t, `package p
A ~ B
B ~ int32
main : proc() = {}`)

		ta := stmtAs[*ast.Type](t, f, 0)
		if ta.Identifier.Name != "A" {
			t.Errorf("expected name 'A', got %q", ta.Identifier.Name)
		}
	})

	t.Run("missing_brace_error", func(t *testing.T) {
		t.Parallel()
		parseShouldError(t, `package p
main : proc() = {
	@print("unclosed")
`)
	})

	t.Run("main_as_int_error", func(t *testing.T) {
		t.Parallel()
		parseShouldError(t, `package p
main := 1`)
	})

	t.Run("main_as_short_decl_error", func(t *testing.T) {
		t.Parallel()
		parseShouldError(t, `package p
main := proc() = {}`)
	})

	t.Run("multiple_errors", func(t *testing.T) {
		t.Parallel()
		parseShouldError(t, `package p
a := 1
a := 2
main : proc() = {}`)
	})
}
