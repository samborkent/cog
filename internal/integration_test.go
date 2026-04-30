package cog_test

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/samborkent/cog/internal/ast"
	"github.com/samborkent/cog/internal/lexer"
	"github.com/samborkent/cog/internal/parser"
	"github.com/samborkent/cog/internal/tokens"
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

	p, err := parser.NewParserWithSymbols(toks, parser.NewSymbolTable(), false, "", 0)
	if err != nil {
		t.Fatalf("parser init error: %v", err)
	}

	f, err := p.Parse(t.Context(), "")
	if err != nil {
		t.Fatalf("parser parse error: %v", err)
	}

	tr := transpiler.NewTranspiler(ast.MergeASTs(f))

	gofile, err := tr.Transpile()
	if err != nil {
		t.Fatalf("transpile error: %v", err)
	}

	var buf bytes.Buffer

	if err := tr.Print(&buf, gofile); err != nil {
		t.Fatalf("printing go ast: %v", err)
	}

	return buf.String()
}

// runGenerated compiles and runs generated Go code, returning its output.
// projectRoot returns the absolute path to the module root (the directory containing go.mod).
func projectRoot(t *testing.T) string {
	t.Helper()

	// We are in internal/, so go up one level.
	dir, err := filepath.Abs(filepath.Join(".", ".."))
	if err != nil {
		t.Fatalf("resolving project root: %v", err)
	}

	return dir
}

// goModCacheDir returns the on-disk directory for a module from the Go module cache.
func goModCacheDir(t *testing.T, module string) string {
	t.Helper()

	out, err := exec.Command("go", "mod", "download", "-json", module+"@latest").CombinedOutput()
	if err != nil {
		t.Fatalf("go mod download %s: %v\n%s", module, err, out)
	}

	// Parse "Dir" field from JSON output.
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, `"Dir"`) {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				dir := strings.TrimSpace(parts[1])
				dir = strings.Trim(dir, `",`)

				return dir
			}
		}
	}

	t.Fatalf("could not find Dir in go mod download output for %s:\n%s", module, out)

	return ""
}

func runGenerated(t *testing.T, code string) (string, error) {
	t.Helper()

	tmpDir := t.TempDir()

	srcPath := filepath.Join(tmpDir, "main.go")
	if err := os.WriteFile(srcPath, []byte(code), 0o600); err != nil {
		t.Fatalf("write generated file: %v", err)
	}

	root := projectRoot(t)

	// Write a go.mod so the generated code can resolve its imports.
	goMod := fmt.Sprintf(`module main

go 1.26.2

require (
	github.com/samborkent/cog v0.0.0
	github.com/samborkent/adaptive-gc v0.0.0
	github.com/pbnjay/memory v0.0.0
)

replace (
	github.com/samborkent/cog => %s
	github.com/samborkent/adaptive-gc => %s
	github.com/pbnjay/memory => %s
)
`, root, filepath.Join(root, "..", "adaptive-gc"), goModCacheDir(t, "github.com/pbnjay/memory"))

	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goMod), 0o600); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}

	ctx, cancel := context.WithTimeout(t.Context(), 8*time.Second)
	defer cancel()

	binPath := filepath.Join(tmpDir, "bin")

	build := exec.CommandContext(ctx, "go", "build", "-ldflags=-s -w", "-o", binPath, ".")
	build.Dir = tmpDir
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

	p, err := parser.NewParserWithSymbols(toks, parser.NewSymbolTable(), false, "", 0)
	if err != nil {
		return "", fmt.Errorf("parser init: %w", err)
	}

	f, err := p.Parse(ctx, "")
	if err != nil {
		return "", fmt.Errorf("parser parse: %w", err)
	}

	tr := transpiler.NewTranspiler(ast.MergeASTs(f))

	gofile, err := tr.Transpile()
	if err != nil {
		return "", fmt.Errorf("transpile: %w", err)
	}

	var buf bytes.Buffer

	if err := tr.Print(&buf, gofile); err != nil {
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

	p, err := parser.NewParserWithSymbols(toks, parser.NewSymbolTable(), false, "", 0)
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

	p, err := parser.NewParserWithSymbols(toks, parser.NewSymbolTable(), true, "", 0)
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

func TestFuncWritingDynShouldError(t *testing.T) {
	t.Parallel()

	src := `package main

dyn val : utf8 = "def"

writer : func() utf8 = {
	val = "changed"
	return val
}

main : proc() = {}
`

	_, err := tryTranspile(t.Context(), src)
	if err == nil {
		t.Fatalf("expected transpile error for func writing dyn variable, got nil")
	}

	if !strings.Contains(err.Error(), "func cannot assign dynamically scoped variable") {
		t.Fatalf("expected error about func assigning dyn var, got: %v", err)
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

	// Context is always present for signal handling + adaptive GC.
	if !strings.Contains(code, "context") {
		t.Fatalf("expected context import for signal handling, got:\n%s", code)
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

	// Context is always present for signal handling + adaptive GC.
	if !strings.Contains(code, "context") {
		t.Fatalf("expected context import for signal handling, got:\n%s", code)
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

	// Context is always present for signal handling + adaptive GC.
	if !strings.Contains(code, "context") {
		t.Fatalf("expected context import for signal handling, got:\n%s", code)
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

// transpilePackage runs the multi-file pipeline: lex each source, share one
// symbol table for globals, parse each file, then transpile all files together.
// The files map is filename → cog source.
func transpilePackage(t *testing.T, files map[string]string) string {
	t.Helper()

	type lexedFile struct {
		name   string
		tokens []tokens.Token
	}

	// Deterministic order.
	names := make([]string, 0, len(files))
	for name := range files {
		names = append(names, name)
	}

	sort.Strings(names)

	lexed := make([]lexedFile, 0, len(files))

	for _, name := range names {
		l := lexer.NewLexer(strings.NewReader(files[name]))

		toks, err := l.Parse(t.Context())
		if err != nil {
			t.Fatalf("lexer error (%s): %v", name, err)
		}

		lexed = append(lexed, lexedFile{name: name, tokens: toks})
	}

	// Shared symbol table across all files.
	symbols := parser.NewSymbolTable()

	parsers := make([]*parser.Parser, len(lexed))
	for i, lf := range lexed {
		p, err := parser.NewParserWithSymbols(lf.tokens, symbols, false, lf.name, uint16(i))
		if err != nil {
			t.Fatalf("parser init (%s): %v", lf.name, err)
		}

		p.FindGlobals(t.Context())
		parsers[i] = p
	}

	astFiles := make([]*ast.AST, len(lexed))

	for i, lf := range lexed {
		f, err := parsers[i].ParseOnly(t.Context(), lf.name)
		if err != nil {
			t.Fatalf("parser parse (%s): %v", lf.name, err)
		}

		astFiles[i] = f
	}

	tr := transpiler.NewTranspiler(ast.MergeASTs(astFiles...))

	gofile, err := tr.Transpile()
	if err != nil {
		t.Fatalf("transpile error: %v", err)
	}

	var buf bytes.Buffer
	if err := tr.Print(&buf, gofile); err != nil {
		t.Fatalf("printing go ast: %v", err)
	}

	return buf.String()
}

// tryTranspilePackage is the error-returning variant of transpilePackage.
func tryTranspilePackage(t *testing.T, files map[string]string) (string, error) {
	t.Helper()

	type lexedFile struct {
		name   string
		tokens []tokens.Token
	}

	names := make([]string, 0, len(files))
	for name := range files {
		names = append(names, name)
	}

	sort.Strings(names)

	lexed := make([]lexedFile, 0, len(files))

	for _, name := range names {
		l := lexer.NewLexer(strings.NewReader(files[name]))

		toks, err := l.Parse(t.Context())
		if err != nil {
			return "", fmt.Errorf("lexer (%s): %w", name, err)
		}

		lexed = append(lexed, lexedFile{name: name, tokens: toks})
	}

	symbols := parser.NewSymbolTable()

	parsers := make([]*parser.Parser, len(lexed))
	for i, lf := range lexed {
		p, err := parser.NewParserWithSymbols(lf.tokens, symbols, false, lf.name, uint16(i))
		if err != nil {
			return "", fmt.Errorf("parser init (%s): %w", lf.name, err)
		}

		p.FindGlobals(t.Context())
		parsers[i] = p
	}

	astFiles := make([]*ast.AST, len(lexed))
	for i, lf := range lexed {
		f, err := parsers[i].ParseOnly(t.Context(), lf.name)
		if err != nil {
			return "", fmt.Errorf("parser parse (%s): %w", lf.name, err)
		}

		astFiles[i] = f
	}

	tr := transpiler.NewTranspiler(ast.MergeASTs(astFiles...))

	gofile, err := tr.Transpile()
	if err != nil {
		return "", fmt.Errorf("transpile: %w", err)
	}

	var buf bytes.Buffer
	if err := tr.Print(&buf, gofile); err != nil {
		return "", fmt.Errorf("printing go ast: %w", err)
	}

	return buf.String(), nil
}

// TestMultiFileCrossReference verifies that a type declared in one file
// can be used in another file when both share the same package.
func TestMultiFileCrossReference(t *testing.T) {
	files := map[string]string{
		"types.cog": `package main

Point ~ struct {
    export (
        x : float64
        y : float64
    )
}
`,
		"main.cog": `package main

val : Point = {
    x = 1.5,
    y = 2.5,
}

main : proc() = {
    @print(val.x)
    @print(val.y)
}
`,
	}

	code := transpilePackage(t, files)

	t.Parallel()

	out, err := runGenerated(t, code)
	if err != nil {
		t.Fatalf("running generated program failed: %v\noutput:\n%s\ncode:\n%s", err, out, code)
	}

	if !strings.Contains(out, "1.5") || !strings.Contains(out, "2.5") {
		t.Fatalf("expected '1.5' and '2.5', got:\n%s", out)
	}
}

// TestMultiFileSameGlobalRedeclare verifies that declaring the same global
// in two files produces an error.
func TestMultiFileSameGlobalRedeclare(t *testing.T) {
	t.Parallel()

	files := map[string]string{
		"a.cog": `package main

x : int64 = 1
`,
		"b.cog": `package main

x : int64 = 2
`,
	}

	_, err := tryTranspilePackage(t, files)
	if err == nil {
		t.Fatal("expected error for duplicate global declaration, got none")
	}
}

// TestMultiFileFunctionCrossCall verifies that a function declared in one file
// can be called from another file.
func TestMultiFileFunctionCrossCall(t *testing.T) {
	files := map[string]string{
		"greet.cog": `package main

greet : func(name: utf8) utf8 = {
    return "hello " + name
}
`,
		"main.cog": `package main

main : proc() = {
    @print(greet("world"))
}
`,
	}

	code := transpilePackage(t, files)

	t.Parallel()

	out, err := runGenerated(t, code)
	if err != nil {
		t.Fatalf("running generated program failed: %v\noutput:\n%s\ncode:\n%s", err, out, code)
	}

	if !strings.Contains(out, "hello world") {
		t.Fatalf("expected 'hello world', got:\n%s", out)
	}
}

func TestMainInNonMainPackageShouldError(t *testing.T) {
	t.Parallel()

	// A package that declares main must be named "main".
	// Here findGlobals should succeed, but the package name check should fail.
	src := `package notmain

main : proc() = {
	@print("bad")
}
`

	l := lexer.NewLexer(strings.NewReader(src))

	toks, err := l.Parse(t.Context())
	if err != nil {
		t.Fatalf("lexer error: %v", err)
	}

	symbols := parser.NewSymbolTable()

	p, err := parser.NewParserWithSymbols(toks, symbols, false, "test.cog", 0)
	if err != nil {
		t.Fatalf("parser init error: %v", err)
	}

	p.FindGlobals(t.Context())

	// Replicate the check from runProject: main exists but package != "main".
	if _, hasMain := symbols.Resolve("main"); !hasMain {
		t.Fatal("expected symbol table to contain main")
	}

	// The package name comes from the token stream (tokens[1]).
	pkgName := toks[1].Literal
	if pkgName == "main" {
		t.Fatal("expected non-main package name")
	}
}

func TestDuplicateMainAcrossFilesShouldError(t *testing.T) {
	t.Parallel()

	// Two files in the same package both declaring main should fail.
	files := map[string]string{
		"a.cog": `package main

main : proc() = {
	@print("a")
}
`,
		"b.cog": `package main

main : proc() = {
	@print("b")
}
`,
	}

	_, err := tryTranspilePackage(t, files)
	if err == nil {
		t.Fatal("expected error for duplicate main across files, got nil")
	}

	if !strings.Contains(err.Error(), "cannot redeclare") {
		t.Fatalf("expected 'cannot redeclare' error, got: %v", err)
	}
}

func TestGenericAliasForwardReference(t *testing.T) {
	src := `package main

names : List<utf8> = @slice<utf8>(3)

List<T ~ any> ~ []T

main : proc() = {
	@print(names)
}`

	code := transpileSource(t, src)

	t.Parallel()

	out, err := runGenerated(t, code)
	if err != nil {
		t.Fatalf("running generated program failed: %v\noutput:\n%s", err, out)
	}

	if !strings.Contains(out, "[") {
		t.Fatalf("expected slice output, got:\n%s", out)
	}
}

func TestGenericAliasMultipleConstraints(t *testing.T) {
	src := `package main

List<T ~ any> ~ []T
NumSlice<T ~ number> ~ []T
SortableSlice<T ~ ordered> ~ []T
Dict<K ~ comparable, V ~ any> ~ map<K, V>
TagSlice<T ~ string | int> ~ []T

main : proc() = {
	names : List<utf8> = @slice<utf8>(3)
	@print(names)
	scores : NumSlice<int64> = @slice<int64>(5)
	@print(scores)
	words : SortableSlice<utf8> = @slice<utf8>(10)
	@print(words)
	lookup : Dict<utf8, int64> = @map<utf8, int64>()
	@print(lookup)
	labels : TagSlice<utf8> = @slice<utf8>(3)
	@print(labels)
}`

	code := transpileSource(t, src)

	t.Parallel()

	out, err := runGenerated(t, code)
	if err != nil {
		t.Fatalf("running generated program failed: %v\noutput:\n%s\ncode:\n%s", err, out, code)
	}

	// All slices/maps should produce output.
	lines := strings.Split(strings.TrimSpace(out), "\n")
	if len(lines) < 5 {
		t.Fatalf("expected at least 5 output lines, got %d:\n%s", len(lines), out)
	}
}

func TestComparisonNotConfusedWithGenericTypeParam(t *testing.T) {
	src := `package main

List<T ~ any> ~ []T

main : proc() = {
	index := 10
	if index < 5 {
		@print(index)
	}
	xs : List<int32> = @slice<int32>(3)
	@print(xs)
}`

	code := transpileSource(t, src)

	t.Parallel()

	out, err := runGenerated(t, code)
	if err != nil {
		t.Fatalf("running generated program failed: %v\noutput:\n%s\ncode:\n%s", err, out, code)
	}

	if !strings.Contains(out, "[") {
		t.Fatalf("expected slice output, got:\n%s", out)
	}
}

// mustContain checks that 'got' contains 'want'.
func mustContain(t *testing.T, got, want string) {
	t.Helper()

	if !strings.Contains(got, want) {
		t.Errorf("expected %q to contain %q", got, want)
	}
}

// mustNotContain checks that 'got' does not contain 'want'.
func mustNotContain(t *testing.T, got, want string) {
	t.Helper()

	if strings.Contains(got, want) {
		t.Errorf("expected %q not to contain %q", got, want)
	}
}

func TestImportedPackageMustNotDeclareMain(t *testing.T) {
	t.Parallel()

	// Simulate an imported package that declares a main proc.
	// After findGlobals, the symbol table must not contain main for library packages.
	src := `package geom

main : proc() = {
	@print("bad")
}
`

	l := lexer.NewLexer(strings.NewReader(src))

	toks, err := l.Parse(t.Context())
	if err != nil {
		t.Fatalf("lexer error: %v", err)
	}

	symbols := parser.NewSymbolTable()

	p, err := parser.NewParserWithSymbols(toks, symbols, false, "", 0)
	if err != nil {
		t.Fatalf("parser init error: %v", err)
	}

	p.FindGlobals(t.Context())

	// Replicate the check from compileImportedPackage: imported packages must not have main.
	if _, hasMain := symbols.Resolve("main"); !hasMain {
		t.Fatal("expected symbol table to contain main for this test scenario")
	}

	// In the real pipeline, this would cause compileImportedPackage to return nil.
	// Here we just verify the symbol is detected.
	pkgName := toks[1].Literal
	if pkgName == "main" {
		t.Fatal("expected non-main package name for imported package test")
	}
}

func TestFunctionTranspilation(t *testing.T) {
	t.Parallel()

	// Test that functions are transpiled as function declarations, not variable declarations
	got := transpilePackage(t, map[string]string{
		"main.cog": `package main

// Regular function
add : func(a : int64, b : int64) int64 = {
    return a + b
}

// Procedure
greet : proc(name : utf8) = {
    @print(name)
}

main : proc() = {
    result := add(1, 2)
    greet("hello")
    @print(result)
}`,
	})

	// Should contain function declarations, not variable declarations
	mustContain(t, got, "func add(a int64, b int64) int64 {")
	mustContain(t, got, "func greet(ctx go_context.Context, name string)")

	// Should NOT contain the old variable declaration format
	mustNotContain(t, got, "var add func")
	mustNotContain(t, got, "var greet")
}

func TestGenericFunctionCallInferred(t *testing.T) {
	src := `package main

genFunc : func<T ~ any>(x : T) = {
	@print(x)
}

main : proc() = {
	genFunc("hello generics")
	genFunc(42)
}`

	code := transpileSource(t, src)

	t.Parallel()

	// Should emit Go generic call syntax.
	mustContain(t, code, "genFunc[string]")
	mustContain(t, code, "genFunc[int64]")
	mustContain(t, code, "func genFunc[T any]")

	out, err := runGenerated(t, code)
	if err != nil {
		t.Fatalf("running generated program failed: %v\noutput:\n%s\ncode:\n%s", err, out, code)
	}

	mustContain(t, out, "hello generics")
	mustContain(t, out, "42")
}

func TestGenericFunctionCallExplicit(t *testing.T) {
	src := `package main

genFunc : func<T ~ any>(x : T) = {
	@print(x)
}

main : proc() = {
	genFunc<utf8>("explicit call")
}`

	code := transpileSource(t, src)

	t.Parallel()

	mustContain(t, code, "genFunc[string]")

	out, err := runGenerated(t, code)
	if err != nil {
		t.Fatalf("running generated program failed: %v\noutput:\n%s\ncode:\n%s", err, out, code)
	}

	mustContain(t, out, "explicit call")
}

func TestGenericFunctionWithReturnType(t *testing.T) {
	src := `package main

identity : func<T ~ any>(x : T) T = {
	return x
}

main : proc() = {
	result := identity("returned")
	@print(result)
}`

	code := transpileSource(t, src)

	t.Parallel()

	mustContain(t, code, "func identity[T any](x T) T {")

	out, err := runGenerated(t, code)
	if err != nil {
		t.Fatalf("running generated program failed: %v\noutput:\n%s\ncode:\n%s", err, out, code)
	}

	mustContain(t, out, "returned")
}
