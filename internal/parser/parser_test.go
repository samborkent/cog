package parser_test

import (
	"strings"
	"testing"

	"github.com/samborkent/cog/internal/ast"
	"github.com/samborkent/cog/internal/lexer"
	"github.com/samborkent/cog/internal/parser"
)

func NewTestParser(t *testing.T, lex *lexer.Lexer) (*parser.Parser, error) {
	t.Helper()

	return parser.NewParserWithSymbols(lex, parser.NewSymbolTable(), "", 0, nil)
}

func TestParse(t *testing.T) {
	t.Parallel()

	t.Run("parse_only_after_findglobals_starts_at_package", func(t *testing.T) {
		t.Parallel()

		src := `package p
main : proc() = {}`

		l := lexer.New(strings.NewReader(src), uint32(len(src)), false)

		p, err := NewTestParser(t, l)
		if err != nil {
			t.Fatalf("parser init error: %v", err)
		}

		if _, err := p.ParseGlobals(t.Context(), "test.cog"); err != nil {
			t.Fatalf("ParseGlobals error: %v", err)
		}

		if err := p.ParseBodies(t.Context()); err != nil {
			t.Fatalf("ParseBodies error: %v", err)
		}
	})

	t.Run("file_name", func(t *testing.T) {
		t.Parallel()

		f := parse(t, `package p
main : proc() = {}`)

		file := f.Node(1).(*ast.File)

		if file.Name != "test.cog" {
			t.Errorf("expected file name 'test.cog', got %q", file.Name)
		}
	})

	t.Run("forward_type_reference", func(t *testing.T) {
		t.Parallel()
		f := parse(t, `package p
A ~ B
B ~ int32
main : proc() = {}`)

		ta := stmtAs[*ast.Type](t, f, 0)
		if ta.Identifier.Token.Literal != "A" {
			t.Errorf("expected name 'A', got %q", ta.Identifier.Token.Literal)
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
