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

// runGenerated compiles and runs generated Go code, returning its output.
func runGenerated(t *testing.T, code string) (string, error) {
	t.Helper()

	tmpDir := t.TempDir()

	srcPath := filepath.Join(tmpDir, "main.go")
	if err := os.WriteFile(srcPath, []byte(code), 0o600); err != nil {
		t.Fatalf("write generated file: %v", err)
	}

	ctx, cancel := context.WithTimeout(t.Context(), 8*time.Second)
	defer cancel()

	binPath := filepath.Join(tmpDir, "bin")

	build := exec.CommandContext(ctx, "go", "build", "-ldflags=-s -w", "-o", binPath, srcPath)
	if out, err := build.CombinedOutput(); err != nil {
		return string(out), err
	}

	run := exec.CommandContext(ctx, binPath)
	out, err := run.CombinedOutput()

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

	t.Parallel()

	out, err := runGenerated(t, code)
	if err != nil {
		t.Fatalf("running generated program failed: %v\noutput:\n%s", err, out)
	}

	if !strings.Contains(out, "hello") {
		t.Fatalf("expected output to contain 'hello', got:\n%s", out)
	}
}

func TestIfBuiltinTypeMismatch(t *testing.T) {
	t.Parallel()

	src := `package main

main : proc() = {
	@print(@if(true, "str", 10))
}`

	_, err := tryTranspile(t.Context(), src)
	if err == nil {
		t.Fatalf("expected parser/transpile error for @if type mismatch, got nil")
	}
}

func TestDynDeclarationInsideProcShouldError(t *testing.T) {
	t.Parallel()

	src := `package main

main : proc() = {
	dyn inner : utf8 = "nope"
}`

	_, err := tryTranspile(t.Context(), src)
	if err == nil {
		t.Fatalf("expected parser error for dyn inside proc, got nil")
	}
}

func TestEnumMissingAssignmentShouldError(t *testing.T) {
	t.Parallel()

	src := `package main

Status ~ enum<utf8> {
	Open,
}

main : proc() = {}
`

	_, err := tryTranspile(t.Context(), src)
	if err == nil {
		t.Fatalf("expected parser error for malformed enum literal (missing :=), got nil")
	}
}

func TestMissingPackageProducesError(t *testing.T) {
	t.Parallel()

	// No package declaration -> parser should return an error
	src := `main : proc() = {}`

	l := lexer.NewLexer(strings.NewReader(src))
	toks, err := l.Parse(t.Context())
	if err != nil {
		t.Fatalf("lexer error: %v", err)
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

Status ~ enum<utf8> {
    Open := "open",
}

main : proc() = {
    v := Status.Open
    @print(v)
}`

	code := transpileSource(t, src)

	t.Parallel()

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

	t.Parallel()

	out, err := runGenerated(t, code)
	if err != nil {
		t.Fatalf("running generated program failed: %v\noutput:\n%s", err, out)
	}

	if !strings.Contains(out, "default") || !strings.Contains(out, "x") {
		t.Fatalf("expected output to contain 'default' and 'x', got:\n%s", out)
	}
}

func TestDuplicateGlobalDeclarationShouldError(t *testing.T) {
	t.Parallel()

	src := `package main

a := 1
a := 2

main : proc() = {}`
	l := lexer.NewLexer(strings.NewReader(src))
	toks, err := l.Parse(t.Context())
	if err != nil {
		t.Fatalf("lexer error: %v", err)
	}

	p, err := parser.NewParser(toks, true)
	if err != nil {
		t.Fatalf("parser init error: %v", err)
	}

	_, err = p.Parse(t.Context(), "")
	if err == nil {
		t.Fatalf("expected error for duplicate global declaration, got nil")
	}
}

func TestMissingParenInIfShouldError(t *testing.T) {
	t.Parallel()

	src := `package main

main : proc() = {
	@print(@if(true, 1, 2)
}`

	_, err := tryTranspile(t.Context(), src)
	if err == nil {
		t.Fatalf("expected parser error for missing paren in @if, got nil")
	}
}

func TestUndefinedIdentifierShouldError(t *testing.T) {
	t.Parallel()

	src := `package main

main : proc() = {
	x := y
}`

	_, err := tryTranspile(t.Context(), src)
	if err == nil {
		t.Fatalf("expected parser error for undefined identifier, got nil")
	}
}

func TestFuncReferencingDynShouldError(t *testing.T) {
	t.Parallel()

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
}

func TestMainAsIntShouldError(t *testing.T) {
	t.Parallel()

	src := `package main

main : int64 = 5
`

	_, err := tryTranspile(t.Context(), src)
	if err == nil {
		t.Fatalf("expected parser error for main declared as int64, got nil")
	}

	if !strings.Contains(err.Error(), `"main" can only be declared as proc()`) {
		t.Fatalf("expected error about main declaration, got: %v", err)
	}
}

func TestMainAsShortDeclShouldError(t *testing.T) {
	t.Parallel()

	src := `package main

main := 5
`

	_, err := tryTranspile(t.Context(), src)
	if err == nil {
		t.Fatalf("expected parser error for main := 5, got nil")
	}

	if !strings.Contains(err.Error(), `"main" can only be declared as proc()`) {
		t.Fatalf("expected error about main declaration, got: %v", err)
	}
}

func TestMainAsFuncShouldError(t *testing.T) {
	t.Parallel()

	src := `package main

main : func() utf8 = {
	return "hello"
}
`

	_, err := tryTranspile(t.Context(), src)
	if err == nil {
		t.Fatalf("expected parser error for main declared as func, got nil")
	}

	if !strings.Contains(err.Error(), `"main" can only be declared as proc()`) {
		t.Fatalf("expected error about main declaration, got: %v", err)
	}
}

func TestMainAsProcWithParamsShouldError(t *testing.T) {
	t.Parallel()

	src := `package main

main : proc(x : utf8) = {
	@print(x)
}
`

	_, err := tryTranspile(t.Context(), src)
	if err == nil {
		t.Fatalf("expected parser error for main declared with parameters, got nil")
	}

	if !strings.Contains(err.Error(), `"main" can only be declared as proc()`) {
		t.Fatalf("expected error about main declaration, got: %v", err)
	}
}

func TestMainAsProcIsValid(t *testing.T) {
	src := `package main

main : proc() = {
	@print("valid main")
}
`

	code := transpileSource(t, src)

	t.Parallel()

	out, err := runGenerated(t, code)
	if err != nil {
		t.Fatalf("running generated program failed: %v\noutput:\n%s", err, out)
	}

	if !strings.Contains(out, "valid main") {
		t.Fatalf("expected output to contain 'valid main', got:\n%s", out)
	}
}

func TestNoContextWithoutProcsOrDyn(t *testing.T) {
	src := `package main

main : proc() = {
	@print("no context needed")
}
`

	code := transpileSource(t, src)

	if strings.Contains(code, "\"context\"") {
		t.Fatalf("expected no context import for simple main, got:\n%s", code)
	}

	t.Parallel()

	out, err := runGenerated(t, code)
	if err != nil {
		t.Fatalf("running generated program failed: %v\noutput:\n%s", err, out)
	}

	if !strings.Contains(out, "no context needed") {
		t.Fatalf("expected output to contain 'no context needed', got:\n%s", out)
	}
}

func TestNoContextWithFuncOnly(t *testing.T) {
	src := `package main

add : func(a : int64, b : int64) int64 = {
	return a + b
}

main : proc() = {
	@print(add(1, 2))
}
`

	code := transpileSource(t, src)

	if strings.Contains(code, "\"context\"") {
		t.Fatalf("expected no context for program with only func (no proc), got:\n%s", code)
	}

	t.Parallel()

	out, err := runGenerated(t, code)
	if err != nil {
		t.Fatalf("running generated program failed: %v\noutput:\n%s", err, out)
	}

	if !strings.Contains(out, "3") {
		t.Fatalf("expected output to contain '3', got:\n%s", out)
	}
}

func TestDynVarWithoutProcs(t *testing.T) {
	src := `package main

dyn val : utf8 = "hello"

main : proc() = {
	@print(val)
}
`

	code := transpileSource(t, src)

	if strings.Contains(code, "\"context\"") {
		t.Fatalf("expected no context import for dyn var without procs, got:\n%s", code)
	}

	if !strings.Contains(code, "cogDyn") {
		t.Fatalf("expected cogDyn struct in generated code, got:\n%s", code)
	}

	t.Parallel()

	out, err := runGenerated(t, code)
	if err != nil {
		t.Fatalf("running generated program failed: %v\noutput:\n%s", err, out)
	}

	if !strings.Contains(out, "hello") {
		t.Fatalf("expected output to contain 'hello', got:\n%s", out)
	}
}

func TestContextWithProcDecl(t *testing.T) {
	src := `package main

greet : proc(name : utf8) = {
	@print(name)
}

main : proc() = {
	greet("world")
}
`

	code := transpileSource(t, src)

	if !strings.Contains(code, "context") {
		t.Fatalf("expected context import for program with proc declaration, got:\n%s", code)
	}

	t.Parallel()

	out, err := runGenerated(t, code)
	if err != nil {
		t.Fatalf("running generated program failed: %v\noutput:\n%s", err, out)
	}

	if !strings.Contains(out, "world") {
		t.Fatalf("expected output to contain 'world', got:\n%s", out)
	}
}

func TestDynVarWithProc(t *testing.T) {
	src := `package main

dyn val : utf8 = "initial"

setter : proc(s : utf8) = {
	val = s
	@print(val)
}

main : proc() = {
	@print(val)
	setter("changed")
	@print(val)
}
`

	code := transpileSource(t, src)

	if !strings.Contains(code, "\"context\"") {
		t.Fatalf("expected context import for dyn var with proc, got:\n%s", code)
	}

	if !strings.Contains(code, "cogDyn") {
		t.Fatalf("expected cogDyn struct in generated code, got:\n%s", code)
	}

	t.Parallel()

	out, err := runGenerated(t, code)
	if err != nil {
		t.Fatalf("running generated program failed: %v\noutput:\n%s", err, out)
	}

	// Dynamic scoping: setter's changes to val are isolated (copy-on-entry),
	// so main sees "initial" both before and after calling setter.
	lines := strings.Split(strings.TrimSpace(out), "\n")
	if len(lines) != 3 {
		t.Fatalf("expected 3 lines of output, got %d:\n%s", len(lines), out)
	}

	if lines[0] != "initial" {
		t.Fatalf("expected first line 'initial', got %q", lines[0])
	}

	if lines[1] != "changed" {
		t.Fatalf("expected second line 'changed', got %q", lines[1])
	}

	if lines[2] != "initial" {
		t.Fatalf("expected third line 'initial' (dynamic scoping isolation), got %q", lines[2])
	}
}

func TestUndefinedGoImportInGlobal(t *testing.T) {
	t.Parallel()

	src := `package main

result := @go.strings.ToUpper("hello")

main : proc() = {
    @print(result)
}`
	ctx, cancel := context.WithTimeout(t.Context(), 3*time.Second)
	defer cancel()

	_, err := tryTranspile(ctx, src)
	if err == nil {
		t.Fatal("expected error for undefined go import, got nil")
	}
}

func TestUndefinedGoImportInProc(t *testing.T) {
	t.Parallel()

	src := `package main

main : proc() = {
    str := @go.strings.ToUpper("hello")
    @print(str)
}`
	ctx, cancel := context.WithTimeout(t.Context(), 3*time.Second)
	defer cancel()

	_, err := tryTranspile(ctx, src)
	if err == nil {
		t.Fatal("expected error for undefined go import, got nil")
	}
}

func TestDefinedGoImportButWrongName(t *testing.T) {
	t.Parallel()

	src := `package main

goimport (
    "fmt"
)

main : proc() = {
    str := @go.strings.ToUpper("hello")
    @print(str)
}`
	ctx, cancel := context.WithTimeout(t.Context(), 3*time.Second)
	defer cancel()

	_, err := tryTranspile(ctx, src)
	if err == nil {
		t.Fatal("expected error for undefined go import, got nil")
	}
}

func TestEnumBeforeStructType(t *testing.T) {
	src := `package main

Planets ~ enum<planet> {
    Earth := {
        radius = 0.5,
        mass = 0.1,
    },
}

planet ~ struct {
    export (
        radius : float64
        mass : float64
    )
}

main : proc() = {
    @print("ok")
}`

	code := transpileSource(t, src)

	t.Parallel()

	out, err := runGenerated(t, code)
	if err != nil {
		t.Fatalf("running generated program failed: %v\noutput:\n%s", err, out)
	}

	if !strings.Contains(out, "ok") {
		t.Fatalf("expected 'ok', got:\n%s", out)
	}
}

func TestTypeAliasBeforeTarget(t *testing.T) {
	src := `package main

MyString ~ BaseString
BaseString ~ utf8

main : proc() = {
    s : MyString = "hello"
    @print(s)
}`

	code := transpileSource(t, src)

	t.Parallel()

	out, err := runGenerated(t, code)
	if err != nil {
		t.Fatalf("running generated program failed: %v\noutput:\n%s", err, out)
	}

	if !strings.Contains(out, "hello") {
		t.Fatalf("expected 'hello', got:\n%s", out)
	}
}

func TestGlobalStructLiteral(t *testing.T) {
	src := `package main

Point ~ struct {
    export (
        x : float64
        y : float64
    )
}

val : Point = {
    x = 1.5,
    y = 2.5,
}

main : proc() = {
    @print(val.x)
}`

	code := transpileSource(t, src)

	t.Parallel()

	out, err := runGenerated(t, code)
	if err != nil {
		t.Fatalf("running generated program failed: %v\noutput:\n%s", err, out)
	}

	if !strings.Contains(out, "1.5") {
		t.Fatalf("expected '1.5', got:\n%s", out)
	}
}
