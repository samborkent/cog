package transpiler_test

import (
	"bytes"
	"strings"
	"testing"

	goprinter "go/printer"
	gotoken "go/token"

	"github.com/samborkent/cog/internal/ast"
	"github.com/samborkent/cog/internal/lexer"
	"github.com/samborkent/cog/internal/parser"
	"github.com/samborkent/cog/internal/transpiler"
)

func transpileScript(t *testing.T, src string) string {
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

	tr := transpiler.NewTranspiler([]*ast.File{f})

	gofile, err := tr.TranspileScript()
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

func TestScriptTranspileBasic(t *testing.T) {
	t.Parallel()
	got := transpileScript(t, `x := 42
@print(x)
`)
	mustContain(t, got, "package main")
	mustContain(t, got, "func main()")
	mustContain(t, got, "42")
}

func TestScriptTranspileGoImport(t *testing.T) {
	t.Parallel()
	got := transpileScript(t, `goimport (
	"strings"
)

x := @go.strings.ToUpper("hello")
@print(x)
`)
	mustContain(t, got, "package main")
	mustContain(t, got, "func main()")
	mustContain(t, got, "go_strings")
	mustContain(t, got, `"strings"`)
}

func TestScriptTranspileTypeAlias(t *testing.T) {
	t.Parallel()
	got := transpileScript(t, `Name ~ utf8
n : Name = "hello"
@print(n)
`)
	mustContain(t, got, "package main")
	mustContain(t, got, "func main()")
	// Type alias should be top-level, not inside main.
	mustContain(t, got, "type _Name")
}

func TestScriptTranspileForLoop(t *testing.T) {
	t.Parallel()
	got := transpileScript(t, `xs := @slice<int64>(3)
for v in xs {
	@print(v)
}
`)
	mustContain(t, got, "func main()")
	mustContain(t, got, "range")
}

func TestScriptTranspileSwitch(t *testing.T) {
	t.Parallel()
	got := transpileScript(t, `x := 1
switch x {
case 1:
	@print("one")
default:
	@print("other")
}
`)
	mustContain(t, got, "func main()")
	mustContain(t, got, "switch")
	mustContain(t, got, "case")
}
