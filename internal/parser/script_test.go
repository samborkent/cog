package parser_test

import (
	"context"
	"strings"
	"testing"

	"github.com/samborkent/cog/internal/lexer"
	"github.com/samborkent/cog/internal/parser"
)

func parseScript(t *testing.T, src string) {
	t.Helper()

	l := lexer.NewLexer(strings.NewReader(src))

	toks, err := l.Parse(t.Context())
	if err != nil {
		t.Fatalf("lex error: %v", err)
	}

	p, err := parser.NewScriptParser(toks, false)
	if err != nil {
		t.Fatalf("parser init error: %v", err)
	}

	f, err := p.Parse(t.Context(), "test.cogs")
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	if f.Package == nil || f.Package.Identifier.Name != "main" {
		t.Fatal("expected synthesized package main")
	}
}

func parseScriptShouldError(t *testing.T, src string) {
	t.Helper()

	ctx, cancel := context.WithTimeout(t.Context(), 3e9)
	defer cancel()

	l := lexer.NewLexer(strings.NewReader(src))

	toks, err := l.Parse(ctx)
	if err != nil {
		return
	}

	p, err := parser.NewScriptParser(toks, false)
	if err != nil {
		return
	}

	_, err = p.Parse(ctx, "test.cogs")
	if err == nil {
		t.Fatal("expected parse error, got nil")
	}
}

func TestScriptBasic(t *testing.T) {
	t.Parallel()
	parseScript(t, `x := 42
@print(x)
`)
}

func TestScriptWithImport(t *testing.T) {
	t.Parallel()
	parseScript(t, `goimport (
	"strings"
)

x := @go.strings.ToUpper("hello")
@print(x)
`)
}

func TestScriptPackageNotAllowed(t *testing.T) {
	t.Parallel()
	parseScriptShouldError(t, `package main
x := 42
`)
}

func TestScriptExportNotAllowed(t *testing.T) {
	t.Parallel()
	parseScriptShouldError(t, `export x := 42
`)
}

func TestScriptForLoop(t *testing.T) {
	t.Parallel()
	parseScript(t, `xs := @slice<int64>(3)
for v in xs {
	@print(v)
}
`)
}

func TestScriptIfStatement(t *testing.T) {
	t.Parallel()
	parseScript(t, `x := 5
if x > 3 {
	@print("big")
}
`)
}

func TestScriptSwitchStatement(t *testing.T) {
	t.Parallel()
	parseScript(t, `x := 1
switch x {
case 1:
	@print("one")
case 2:
	@print("two")
}
`)
}

func TestScriptTypeAlias(t *testing.T) {
	t.Parallel()
	parseScript(t, `Name ~ utf8
n : Name = "hello"
@print(n)
`)
}

func TestScriptNoForwardReferences(t *testing.T) {
	t.Parallel()
	parseScriptShouldError(t, `@print(x)
x := 42
`)
}
