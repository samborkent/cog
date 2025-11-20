package cog_test

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	goprinter "go/printer"
	gotoken "go/token"

	"github.com/samborkent/cog/internal/lexer"
	"github.com/samborkent/cog/internal/parser"
	"github.com/samborkent/cog/internal/transpiler"
)

// transpileSource runs the lexer, parser and transpiler and returns generated go source
func transpileSource(t *testing.T, src string) string {
	t.Helper()

	l := lexer.NewLexer(strings.NewReader(src))
	toks, err := l.Parse(t.Context())
	if err != nil {
		t.Fatalf("lexer error: %v", err)
	}

	p, err := parser.NewParser(toks, false)
	if err != nil {
		t.Fatalf("parser init error: %v", err)
	}

	f, err := p.Parse(t.Context(), "")
	if err != nil {
		t.Fatalf("parser parse error: %v", err)
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

// runGenerated writes the generated code into ./tmp/ and runs `go run` on it (cwd is repo root)
func runGenerated(t *testing.T, code string) (string, error) {
	t.Helper()

	// Use testing-managed temp dir to avoid file collisions and ensure cleanup by the testing framework.
	tmpDir := t.TempDir()

	path := filepath.Join(tmpDir, "integration_test_gen.go")
	if err := os.WriteFile(path, []byte(code), 0o600); err != nil {
		t.Fatalf("write generated file: %v", err)
	}

	ctx, cancel := context.WithTimeout(t.Context(), 8*time.Second)
	defer cancel()

	// run `go run <tmpdir>/integration_test_gen.go` with cwd = repo root (test runs from repo root)
	cmd := exec.CommandContext(ctx, "go", "run", path)
	cmd.Dir = "."
	out, err := cmd.CombinedOutput()

	return string(out), err
}

// tryTranspile attempts to run the full pipeline and returns generated code or an error.
func tryTranspile(ctx context.Context, src string) (string, error) {
	l := lexer.NewLexer(strings.NewReader(src))

	toks, err := l.Parse(ctx)
	if err != nil {
		return "", fmt.Errorf("lexer: %w", err)
	}

	p, err := parser.NewParser(toks, false)
	if err != nil {
		return "", fmt.Errorf("parser init: %w", err)
	}

	f, err := p.Parse(ctx, "")
	if err != nil {
		return "", fmt.Errorf("parser parse: %w", err)
	}

	tr := transpiler.NewTranspiler(f)

	gofile, err := tr.Transpile()
	if err != nil {
		return "", fmt.Errorf("transpile: %w", err)
	}

	var buf bytes.Buffer

	fset := gotoken.NewFileSet()
	if err := goprinter.Fprint(&buf, fset, gofile); err != nil {
		return "", fmt.Errorf("printing go ast: %w", err)
	}

	return buf.String(), nil
}

func TestHelloPrint(t *testing.T) {
	src := `package main

main : proc() = {
    @print("hello")
}`

	code := transpileSource(t, src)

	out, err := runGenerated(t, code)
	if err != nil {
		t.Fatalf("running generated program failed: %v\noutput:\n%s", err, out)
	}

	if !strings.Contains(out, "hello") {
		t.Fatalf("expected output to contain 'hello', got:\n%s", out)
	}
}

func TestIfBuiltinTypeMismatch(t *testing.T) {
	src := `package main

main : proc() = {
	@print(@if(true, "str", 10))
}`

	_, err := tryTranspile(t.Context(), src)
	if err == nil {
		t.Fatalf("expected parser/transpile error for @if type mismatch, got nil")
	}

	t.Logf("error as expected: %v", err)
}

func TestDynDeclarationInsideProcShouldError(t *testing.T) {
	src := `package main

main : proc() = {
	dyn inner : utf8 = "nope"
}`

	_, err := tryTranspile(t.Context(), src)
	if err == nil {
		t.Fatalf("expected parser error for dyn inside proc, got nil")
	}

	t.Logf("error as expected: %v", err)
}

func TestEnumMissingAssignmentShouldError(t *testing.T) {
	src := `package main

Status ~ enum[utf8] {
	Open,
}

main : proc() = {}
`

	_, err := tryTranspile(t.Context(), src)
	if err == nil {
		t.Fatalf("expected parser error for malformed enum literal (missing :=), got nil")
	}

	t.Logf("error as expected: %v", err)
}

func TestMissingPackageProducesError(t *testing.T) {
	// No package declaration -> parser should return an error
	src := `main : proc() = {}`

	l := lexer.NewLexer(strings.NewReader(src))
	toks, err := l.Parse(t.Context())
	if err != nil {
		// If lexer fails that's also acceptable for this malformed input
		t.Logf("lexer error (expected for malformed input): %v", err)
		return
	}

	p, err := parser.NewParser(toks, false)
	if err != nil {
		t.Fatalf("parser init error: %v", err)
	}

	_, err = p.Parse(t.Context(), "")
	if err == nil {
		t.Fatalf("expected parser error for missing package, got nil")
	}
}

func TestEnumPrintsUnderlyingValue(t *testing.T) {
	src := `package main

Status ~ enum[utf8] {
    Open := "open",
}

main : proc() = {
    v := Status.Open
    @print(v)
}`

	code := transpileSource(t, src)

	out, err := runGenerated(t, code)
	if err != nil {
		t.Fatalf("running generated program failed: %v\noutput:\n%s", err, out)
	}

	if !strings.Contains(out, "open") {
		t.Fatalf("expected output to contain 'open', got:\n%s", out)
	}
}

func TestDynamicVarDefaultAndOverwrite(t *testing.T) {
	src := `package main

dyn val : utf8 = "default"

main : proc() = {
    @print(val)
    val = "x"
    @print(val)
}`

	code := transpileSource(t, src)

	out, err := runGenerated(t, code)
	if err != nil {
		t.Fatalf("running generated program failed: %v\noutput:\n%s", err, out)
	}

	if !strings.Contains(out, "default") || !strings.Contains(out, "x") {
		t.Fatalf("expected output to contain 'default' and 'x', got:\n%s", out)
	}
}

func TestDuplicateGlobalDeclarationShouldError(t *testing.T) {
	src := `package main

a := 1
a := 2

main : proc() = {}`

	// Run with parser debug enabled to surface where the parser may hang.
	l := lexer.NewLexer(strings.NewReader(src))
	toks, err := l.Parse(t.Context())
	if err != nil {
		t.Fatalf("lexer error: %v", err)
	}

	p, err := parser.NewParser(toks, true)
	if err != nil {
		t.Fatalf("parser init error: %v", err)
	}

	done := make(chan struct{})
	var perr error
	go func() {
		_, perr = p.Parse(t.Context(), "")
		close(done)
	}()

	select {
	case <-done:
		if perr == nil {
			t.Fatalf("expected error for duplicate global declaration, got nil")
		}
		t.Logf("error as expected: %v", perr)
	case <-time.After(5 * time.Second):
		t.Fatalf("parser hung while parsing duplicate globals — observed possible infinite loop")
	}
}

func TestMissingParenInIfShouldError(t *testing.T) {
	src := `package main

main : proc() = {
	@print(@if(true, 1, 2)
}`

	_, err := tryTranspile(t.Context(), src)
	if err == nil {
		t.Fatalf("expected parser error for missing paren in @if, got nil")
	}

	t.Logf("error as expected: %v", err)
}

func TestUndefinedIdentifierShouldError(t *testing.T) {
	src := `package main

main : proc() = {
	x := y
}`

	_, err := tryTranspile(t.Context(), src)
	if err == nil {
		t.Fatalf("expected parser error for undefined identifier, got nil")
	}

	t.Logf("error as expected: %v", err)
}

func TestFuncReferencingDynShouldError(t *testing.T) {
	src := `package main

dyn val : utf8 = "def"

upper : func() utf8 = {
	return val
}

main : proc() = {}
`

	_, err := tryTranspile(t.Context(), src)
	if err == nil {
		t.Fatalf("expected transpile error for func referencing dyn variable, got nil")
	}

	t.Logf("error as expected: %v", err)
}
