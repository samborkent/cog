package transpiler_test

import (
	"bytes"
	"strings"
	"testing"

	goprinter "go/printer"
	gotoken "go/token"

	"github.com/samborkent/cog/internal/lexer"
	"github.com/samborkent/cog/internal/parser"
	"github.com/samborkent/cog/internal/transpiler"
)

// transpile runs the full pipeline and returns generated Go source.
func transpile(t *testing.T, src string) string {
	t.Helper()

	l := lexer.NewLexer(strings.NewReader(src))

	toks, err := l.Parse(t.Context())
	if err != nil {
		t.Fatalf("lex error: %v", err)
	}

	p, err := parser.NewParserWithSymbols(toks, parser.NewSymbolTable(), false, "")
	if err != nil {
		t.Fatalf("parser init error: %v", err)
	}

	f, err := p.Parse(t.Context(), "test.cog")
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	tr := transpiler.NewTranspiler(f)

	gofile, err := tr.Transpile()
	if err != nil {
		t.Fatalf("transpile error: %v", err)
	}

	var buf bytes.Buffer

	fset := gotoken.NewFileSet()
	if err := goprinter.Fprint(&buf, fset, gofile); err != nil {
		t.Fatalf("printing go ast: %v", err)
	}

	return buf.String()
}

// transpileWithPrint runs the full pipeline using Transpiler.Print,
// which includes post-processing (stripping placeholder declarations).
func transpileWithPrint(t *testing.T, src string) string {
	t.Helper()

	l := lexer.NewLexer(strings.NewReader(src))

	toks, err := l.Parse(t.Context())
	if err != nil {
		t.Fatalf("lex error: %v", err)
	}

	p, err := parser.NewParserWithSymbols(toks, parser.NewSymbolTable(), false, "")
	if err != nil {
		t.Fatalf("parser init error: %v", err)
	}

	f, err := p.Parse(t.Context(), "test.cog")
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	tr := transpiler.NewTranspiler(f)

	gofile, err := tr.Transpile()
	if err != nil {
		t.Fatalf("transpile error: %v", err)
	}

	var buf bytes.Buffer
	if err := tr.Print(&buf, gofile); err != nil {
		t.Fatalf("print error: %v", err)
	}

	return buf.String()
}

// mustContain asserts that got contains the substring want.
func mustContain(t *testing.T, got, want string) {
	t.Helper()

	if !strings.Contains(got, want) {
		t.Errorf("output missing %q\ngot:\n%s", want, got)
	}
}

// mustNotContain asserts that got does NOT contain the substring want.
func mustNotContain(t *testing.T, got, want string) {
	t.Helper()

	if strings.Contains(got, want) {
		t.Errorf("output should not contain %q\ngot:\n%s", want, got)
	}
}

// mustFailTranspile asserts that transpilation fails with an error containing want.
func mustFailTranspile(t *testing.T, src, want string) {
	t.Helper()

	l := lexer.NewLexer(strings.NewReader(src))

	toks, err := l.Parse(t.Context())
	if err != nil {
		t.Fatalf("lex error: %v", err)
	}

	p, err := parser.NewParserWithSymbols(toks, parser.NewSymbolTable(), false, "")
	if err != nil {
		t.Fatalf("parser init error: %v", err)
	}

	f, err := p.Parse(t.Context(), "test.cog")
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	tr := transpiler.NewTranspiler(f)

	_, err = tr.Transpile()
	if err == nil {
		t.Fatalf("expected transpile error containing %q, but got no error", want)
	}

	if !strings.Contains(err.Error(), want) {
		t.Errorf("expected error containing %q, got: %v", want, err)
	}
}
