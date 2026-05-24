package transpiler_test

import (
	"bytes"
	"sort"
	"strings"
	"testing"

	goprinter "go/printer"
	gotoken "go/token"

	"github.com/samborkent/cog/internal/ast"
	"github.com/samborkent/cog/internal/lexer"
	"github.com/samborkent/cog/internal/parser"
	"github.com/samborkent/cog/internal/transpiler"
)

// transpileMultiFile parses multiple files with a shared symbol table
// and transpiles using TranspileFiles, returning a map of filename -> Go source.
func transpileMultiFile(t *testing.T, files map[string]string) map[string]string {
	t.Helper()

	names := make([]string, 0, len(files))
	for name := range files {
		names = append(names, name)
	}

	sort.Strings(names)

	type lf struct {
		name  string
		lexer *lexer.Lexer
	}

	lexed := make([]lf, 0, len(files))
	for _, name := range names {
		file := files[name]
		l := lexer.New(strings.NewReader(file), uint32(len(file)), false)

		lexed = append(lexed, lf{name: name, lexer: l})
	}

	symbols := parser.NewSymbolTable()

	parsers := make([]*parser.Parser, len(lexed))
	astFiles := make([]*ast.AST, len(lexed))
	for i, f := range lexed {
		p, err := parser.NewParserWithSymbols(f.lexer, symbols, f.name, uint16(i), nil)
		if err != nil {
			t.Fatalf("parser init (%s): %v", f.name, err)
		}

		af, err := p.ParseGlobals(t.Context(), f.name)
		if err != nil {
			t.Fatalf("parser globals (%s): %v", f.name, err)
		}

		parsers[i] = p
		astFiles[i] = af
	}

	for i, f := range lexed {
		if err := parsers[i].ParseBodies(t.Context()); err != nil {
			t.Fatalf("parse bodies (%s): %v", f.name, err)
		}
	}

	tr := transpiler.NewTranspilerWithModule("testmod", ast.MergeASTs(astFiles...))

	gofiles, err := tr.TranspileFiles(t.Context())
	if err != nil {
		t.Fatalf("TranspileFiles error: %v", err)
	}

	result := make(map[string]string, len(gofiles))
	fset := gotoken.NewFileSet()

	for i, gf := range gofiles {
		var buf bytes.Buffer
		if err := goprinter.Fprint(&buf, fset, gf); err != nil {
			t.Fatalf("print error (%s): %v", lexed[i].name, err)
		}

		result[lexed[i].name] = buf.String()
	}

	return result
}

// transpileWithModule parses a single file and transpiles it using NewTranspilerWithModule.
func transpileWithModule(t *testing.T, moduleName, src string) string {
	t.Helper()

	l := lexer.New(strings.NewReader(src), uint32(len(src)), false)

	p, err := parser.NewParserWithSymbols(l, parser.NewSymbolTable(), "", 0, nil)
	if err != nil {
		t.Fatalf("parser init error: %v", err)
	}

	f, err := p.Parse(t.Context(), "test.cog")
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	tr := transpiler.NewTranspilerWithModule(moduleName, ast.MergeASTs(f))

	gofile, err := tr.Transpile()
	if err != nil {
		t.Fatalf("transpile error: %v", err)
	}

	var buf bytes.Buffer

	fset := gotoken.NewFileSet()
	if err := goprinter.Fprint(&buf, fset, gofile); err != nil {
		t.Fatalf("print error: %v", err)
	}

	return buf.String()
}

func TestTranspileFiles(t *testing.T) {
	t.Parallel()

	t.Run("single_file", func(t *testing.T) {
		t.Parallel()
		result := transpileMultiFile(t, map[string]string{
			"main.cog": `package main

main : proc() = {
	@print("hello")
}
`,
		})

		got, ok := result["main.cog"]
		if !ok {
			t.Fatal("expected main.cog in output")
		}

		mustContain(t, got, "package main")
		mustContain(t, got, "builtin.Print")
	})

	t.Run("two_files", func(t *testing.T) {
		t.Parallel()
		result := transpileMultiFile(t, map[string]string{
			"greet.cog": `package main

greet : func(name : utf8) utf8 = {
	return "hello " + name
}
`,
			"main.cog": `package main

main : proc() = {
	@print(greet("world"))
}
`,
		})

		if len(result) != 2 {
			t.Fatalf("expected 2 output files, got %d", len(result))
		}

		main := result["main.cog"]
		mustContain(t, main, "package main")
		mustContain(t, main, "greet")
	})

	t.Run("cross_file_variable_reference_same_package", func(t *testing.T) {
		t.Parallel()
		result := transpileMultiFile(t, map[string]string{
			"other.cog": `package main

Coordinate ~ struct {
	lat : float64
	lon : float64
}

formatCoord : func(c : Coordinate) utf8 = {
	return "lat"
}
`,
			"main.cog": `package main

Planet ~ struct {
	radius : float64
	mass : float64
}

main : proc() = {
	loc : Coordinate = {
		lat = 52.37,
		lon = 4.89,
	}
	@print(formatCoord(loc))

	earth : Planet = {
		radius = 10,
		mass = 20,
	}
	_ = earth.radius
}
`,
		})

		if len(result) != 2 {
			t.Fatalf("expected 2 output files, got %d", len(result))
		}

		main := result["main.cog"]

		// main.cog should contain the declaration of earth and reference it
		mustContain(t, main, "earth")
		mustContain(t, main, "radius")
	})
}

func TestTranspileFilesDoesNotLeakLocalSymbolsAcrossFiles(t *testing.T) {
	t.Parallel()

	result := transpileMultiFile(t, map[string]string{
		"a.cog": `package main

main : proc() = {
	shared := 1
	_ = shared
}
`,
		"b.cog": `package main

other : proc() = {
	shared := 2
	@print(shared)
}
`,
		"c.cog": `package main

third : proc() = {
	@print("ok")
}
`,
		"d.cog": `package main

fourth : proc() = {
	@print("ok")
}
`,
	})

	a := result["a.cog"]
	mustContain(t, a, "shared")
}

func TestTranspileFilesWithModule(t *testing.T) {
	t.Parallel()

	got := transpileWithModule(t, "myproject", `package main

main : proc() = {
	@print("test")
}
`)

	mustContain(t, got, "package main")
}

func TestTranspileGoImportAlias(t *testing.T) {
	t.Parallel()

	got := transpile(t, `package main

goimport (
	"strings"
)

main : proc() = {
	x := @go.strings.ToUpper("hello")
	@print(x)
}
`)

	mustContain(t, got, "go_strings")
	mustContain(t, got, `"strings"`)
}

func TestTranspileMultiFileGoImport(t *testing.T) {
	t.Parallel()

	result := transpileMultiFile(t, map[string]string{
		"main.cog": `package main

goimport (
	"strings"
)

main : proc() = {
	x := @go.strings.ToUpper("hello")
	@print(x)
}
`,
	})

	got := result["main.cog"]
	mustContain(t, got, "go_strings")
}

func TestTranspileCogImport(t *testing.T) {
	t.Parallel()

	// Test that cog import generates proper Go import with module prefix.

	src := `package main

import (
	"geom"
)

main : proc() = {}
`
	l := lexer.New(strings.NewReader(src), uint32(len(src)), false)

	p, err := parser.NewParserWithSymbols(l, parser.NewSymbolTable(), "", 0, nil)
	if err != nil {
		t.Fatalf("parser init: %v", err)
	}

	f, err := p.Parse(t.Context(), "test.cog")
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	tr := transpiler.NewTranspilerWithModule("mymod", ast.MergeASTs(f))

	gofile, err := tr.Transpile()
	if err != nil {
		t.Fatalf("transpile error: %v", err)
	}

	var buf bytes.Buffer

	fset := gotoken.NewFileSet()
	if err := goprinter.Fprint(&buf, fset, gofile); err != nil {
		t.Fatalf("print error: %v", err)
	}

	got := buf.String()
	mustContain(t, got, `"mymod/geom"`)
}

func TestTranspileExportedName(t *testing.T) {
	t.Parallel()

	got := transpile(t, `package main

export val := 42

main : proc() = {
	@print(val)
}
`)

	mustContain(t, got, "Val")
}
