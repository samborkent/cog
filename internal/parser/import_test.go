package parser_test

import (
	"strings"
	"testing"

	"github.com/samborkent/cog/internal/ast"
	"github.com/samborkent/cog/internal/lexer"
	"github.com/samborkent/cog/internal/parser"
	"github.com/samborkent/cog/internal/types"
)

// parseWithSharedSymbols uses the FindGlobals + ParseOnly flow with a shared symbol table.
func parseWithSharedSymbols(t *testing.T, sources map[string]string) map[string]*ast.AST {
	t.Helper()

	symbols := parser.NewSymbolTable()

	type entry struct {
		name   string
		parser *parser.Parser
	}

	var entries []entry

	fileID := uint16(0)

	for name, src := range sources {
		l := lexer.NewLexer(strings.NewReader(src))

		toks, err := l.Parse(t.Context())
		if err != nil {
			t.Fatalf("lex error (%s): %v", name, err)
		}

		p, err := parser.NewParserWithSymbols(toks, symbols, false, src, fileID)
		if err != nil {
			t.Fatalf("parser init (%s): %v", name, err)
		}

		p.FindGlobals(t.Context())
		entries = append(entries, entry{name: name, parser: p})

		fileID++
	}

	result := make(map[string]*ast.AST, len(entries))

	for _, e := range entries {
		f, err := e.parser.ParseOnly(t.Context(), e.name)
		if err != nil {
			t.Fatalf("parse error (%s): %v", e.name, err)
		}

		result[e.name] = f
	}

	return result
}

// findGlobalsShouldError runs FindGlobals + ParseOnly and expects an error.
func findGlobalsShouldError(t *testing.T, src string) {
	t.Helper()

	l := lexer.NewLexer(strings.NewReader(src))

	toks, err := l.Parse(t.Context())
	if err != nil {
		return // lexer error is fine
	}

	symbols := parser.NewSymbolTable()

	p, err := parser.NewParserWithSymbols(toks, symbols, false, "test.cog", 0)
	if err != nil {
		return
	}

	p.FindGlobals(t.Context())

	_, err = p.ParseOnly(t.Context(), "test.cog")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestFindGlobalsImport(t *testing.T) {
	t.Parallel()

	t.Run("import_registers_cog_import", func(t *testing.T) {
		t.Parallel()

		src := `package main

import (
	"geom"
)

main : proc() = {}
`
		l := lexer.NewLexer(strings.NewReader(src))

		toks, err := l.Parse(t.Context())
		if err != nil {
			t.Fatalf("lex error: %v", err)
		}

		symbols := parser.NewSymbolTable()

		p, err := parser.NewParserWithSymbols(toks, symbols, false, "", 0)
		if err != nil {
			t.Fatalf("parser init: %v", err)
		}

		p.FindGlobals(t.Context())

		imports := symbols.CogImports()
		if len(imports) != 1 {
			t.Fatalf("expected 1 import, got %d", len(imports))
		}

		imp, ok := symbols.ResolveCogImport("geom")
		if !ok {
			t.Fatal("expected to resolve cog import 'geom'")
		}

		if imp.Path != "geom" {
			t.Errorf("import path = %q, want geom", imp.Path)
		}
	})

	t.Run("import_subpackage", func(t *testing.T) {
		t.Parallel()

		src := `package main

import (
	"geom/metric"
)

main : proc() = {}
`
		l := lexer.NewLexer(strings.NewReader(src))

		toks, err := l.Parse(t.Context())
		if err != nil {
			t.Fatalf("lex error: %v", err)
		}

		symbols := parser.NewSymbolTable()

		p, err := parser.NewParserWithSymbols(toks, symbols, false, "", 0)
		if err != nil {
			t.Fatalf("parser init: %v", err)
		}

		p.FindGlobals(t.Context())

		imp, ok := symbols.ResolveCogImport("metric")
		if !ok {
			t.Fatal("expected to resolve cog import 'metric'")
		}

		if imp.Path != "geom/metric" {
			t.Errorf("import path = %q, want geom/metric", imp.Path)
		}

		if imp.Name != "metric" {
			t.Errorf("import name = %q, want metric", imp.Name)
		}
	})

	t.Run("import_multiple", func(t *testing.T) {
		t.Parallel()

		src := `package main

import (
	"alpha"
	"beta"
)

main : proc() = {}
`
		l := lexer.NewLexer(strings.NewReader(src))

		toks, err := l.Parse(t.Context())
		if err != nil {
			t.Fatalf("lex error: %v", err)
		}

		symbols := parser.NewSymbolTable()

		p, err := parser.NewParserWithSymbols(toks, symbols, false, "", 0)
		if err != nil {
			t.Fatalf("parser init: %v", err)
		}

		p.FindGlobals(t.Context())

		imports := symbols.CogImports()
		if len(imports) != 2 {
			t.Fatalf("expected 2 imports, got %d", len(imports))
		}
	})

	t.Run("import_invalid_parent_traversal", func(t *testing.T) {
		t.Parallel()
		findGlobalsShouldError(t, `package main

import (
	"../escape"
)

main : proc() = {}
`)
	})

	t.Run("import_invalid_absolute", func(t *testing.T) {
		t.Parallel()
		findGlobalsShouldError(t, `package main

import (
	"/absolute"
)

main : proc() = {}
`)
	})
}

func TestFindGlobalsMultiFile(t *testing.T) {
	t.Parallel()

	t.Run("shared_symbols", func(t *testing.T) {
		t.Parallel()
		files := parseWithSharedSymbols(t, map[string]string{
			"types.cog": `package main

Point ~ struct {
	x : float64
	y : float64
}
`,
			"main.cog": `package main

main : proc() = {
	@print("hello")
}
`,
		})

		if len(files) != 2 {
			t.Fatalf("expected 2 files, got %d", len(files))
		}
	})

	t.Run("globals_across_files", func(t *testing.T) {
		t.Parallel()

		syms := parser.NewSymbolTable()

		src1 := `package main

val := 42
`
		l1 := lexer.NewLexer(strings.NewReader(src1))
		toks1, _ := l1.Parse(t.Context())
		p1, _ := parser.NewParserWithSymbols(toks1, syms, false, "", 0)
		p1.FindGlobals(t.Context())

		src2 := `package main

main : proc() = {
	@print(val)
}
`
		l2 := lexer.NewLexer(strings.NewReader(src2))
		toks2, _ := l2.Parse(t.Context())
		p2, _ := parser.NewParserWithSymbols(toks2, syms, false, "", 0)
		p2.FindGlobals(t.Context())

		// val should be resolvable by the shared symbol table.
		_, ok := syms.Resolve("val")
		if !ok {
			t.Fatal("expected 'val' to be in shared symbol table")
		}

		// Both files should parse successfully.
		if _, err := p1.ParseOnly(t.Context(), "types.cog"); err != nil {
			t.Fatalf("parse types.cog: %v", err)
		}

		if _, err := p2.ParseOnly(t.Context(), "main.cog"); err != nil {
			t.Fatalf("parse main.cog: %v", err)
		}
	})
}

func TestFindGlobalsImportWithExports(t *testing.T) {
	t.Parallel()

	src := `package main

import (
	"geom"
)

main : proc() = {}
`
	l := lexer.NewLexer(strings.NewReader(src))

	toks, err := l.Parse(t.Context())
	if err != nil {
		t.Fatalf("lex error: %v", err)
	}

	symbols := parser.NewSymbolTable()

	p, err := parser.NewParserWithSymbols(toks, symbols, false, "", 0)
	if err != nil {
		t.Fatalf("parser init: %v", err)
	}

	p.FindGlobals(t.Context())

	// Simulate populating exports (as cmd/main.go does).
	imp, ok := symbols.ResolveCogImport("geom")
	if !ok {
		t.Fatal("expected cog import 'geom'")
	}

	imp.Exports["Distance"] = parser.Symbol{
		Identifier: &ast.Identifier{
			Name:      "Distance",
			ValueType: &types.Procedure{Function: true, Parameters: []*types.Parameter{{Name: "a", Type: types.Basics[types.Float64]}, {Name: "b", Type: types.Basics[types.Float64]}}, ReturnType: types.Basics[types.Float64]},
			Exported:  true,
		},
	}
	imp.Exports["Pi"] = parser.Symbol{
		Identifier: &ast.Identifier{
			Name:      "Pi",
			ValueType: types.Basics[types.Float64],
			Exported:  true,
		},
	}

	// Verify exports are accessible.
	if len(imp.Exports) != 2 {
		t.Fatalf("expected 2 exports, got %d", len(imp.Exports))
	}

	if _, ok := imp.Exports["Distance"]; !ok {
		t.Error("missing export Distance")
	}

	if _, ok := imp.Exports["Pi"]; !ok {
		t.Error("missing export Pi")
	}
}
