package parser_test

import (
	"strings"
	"testing"

	"github.com/samborkent/cog/internal/ast"
	"github.com/samborkent/cog/internal/lexer"
	"github.com/samborkent/cog/internal/parser"
	"github.com/samborkent/cog/internal/tokens"
	"github.com/samborkent/cog/internal/types"
)

// parseWithSharedSymbols uses the ParseGlobals + ParseBodies flow with a shared symbol table.
func parseWithSharedSymbols(t *testing.T, sources map[string]string) map[string]*ast.AST {
	t.Helper()

	symbols := parser.NewSymbolTable()

	type entry struct {
		name   string
		parser *parser.Parser
		ast    *ast.AST
	}

	var entries []entry

	fileID := uint16(0)

	for name, src := range sources {
		l := lexer.New(strings.NewReader(src), uint32(len(src)), false)

		p, err := parser.NewParserWithSymbols(l, symbols, src, fileID, nil)
		if err != nil {
			t.Fatalf("parser init (%s): %v", name, err)
		}

		f, err := p.ParseGlobals(t.Context(), name)
		if err != nil {
			t.Fatalf("parser globals (%s): %v", name, err)
		}

		entries = append(entries, entry{name: name, parser: p, ast: f})

		fileID++
	}

	result := make(map[string]*ast.AST, len(entries))

	for _, e := range entries {
		if err := e.parser.ParseBodies(t.Context()); err != nil {
			t.Fatalf("parse bodies error (%s): %v", e.name, err)
		}

		result[e.name] = e.ast
	}

	return result
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
		l := lexer.New(strings.NewReader(src), uint32(len(src)), false)

		symbols := parser.NewSymbolTable()

		p, err := parser.NewParserWithSymbols(l, symbols, src, 0, nil)
		if err != nil {
			t.Fatalf("parser init: %v", err)
		}

		if _, err := p.ParseGlobals(t.Context(), "test.cog"); err != nil {
			t.Fatalf("ParseGlobals: %v", err)
		}

		imports := symbols.CogImports()
		if imports.Len() != 1 {
			t.Fatalf("expected 1 import, got %d", imports.Len())
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
		l := lexer.New(strings.NewReader(src), uint32(len(src)), false)
		symbols := parser.NewSymbolTable()

		p, err := parser.NewParserWithSymbols(l, symbols, "", 0, nil)
		if err != nil {
			t.Fatalf("parser init: %v", err)
		}

		if _, err := p.ParseGlobals(t.Context(), "test.cog"); err != nil {
			t.Fatalf("ParseGlobals: %v", err)
		}

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
		l := lexer.New(strings.NewReader(src), uint32(len(src)), false)
		symbols := parser.NewSymbolTable()

		p, err := parser.NewParserWithSymbols(l, symbols, src, 0, nil)
		if err != nil {
			t.Fatalf("parser init: %v", err)
		}

		if _, err := p.ParseGlobals(t.Context(), "test.cog"); err != nil {
			t.Fatalf("ParseGlobals: %v", err)
		}

		imports := symbols.CogImports()
		if imports.Len() != 2 {
			t.Fatalf("expected 2 imports, got %d", imports.Len())
		}
	})

	t.Run("import_invalid_parent_traversal", func(t *testing.T) {
		t.Parallel()
		parseShouldError(t, `package main

import (
	"../escape"
)

main : proc() = {}
`)
	})

	t.Run("import_invalid_absolute", func(t *testing.T) {
		t.Parallel()
		parseShouldError(t, `package main

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
		l1 := lexer.New(strings.NewReader(src1), uint32(len(src1)), false)
		p1, _ := parser.NewParserWithSymbols(l1, syms, src1, 0, nil)
		if _, err := p1.ParseGlobals(t.Context(), "val.cog"); err != nil {
			t.Fatalf("ParseGlobals val.cog: %v", err)
		}

		src2 := `package main

main : proc() = {
	@print(val)
}
`
		l2 := lexer.New(strings.NewReader(src2), uint32(len(src2)), false)
		p2, _ := parser.NewParserWithSymbols(l2, syms, src2, 0, nil)
		if _, err := p2.ParseGlobals(t.Context(), "main.cog"); err != nil {
			t.Fatalf("ParseGlobals main.cog: %v", err)
		}

		// val should be resolvable by the shared symbol table.
		_, ok := syms.ResolveIdent("val")
		if !ok {
			t.Fatal("expected 'val' to be in shared symbol table")
		}

		// Both files should have their bodies parsed successfully.
		if err := p1.ParseBodies(t.Context()); err != nil {
			t.Fatalf("ParseBodies val.cog: %v", err)
		}

		if err := p2.ParseBodies(t.Context()); err != nil {
			t.Fatalf("ParseBodies main.cog: %v", err)
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
	l := lexer.New(strings.NewReader(src), uint32(len(src)), false)
	symbols := parser.NewSymbolTable()

	p, err := parser.NewParserWithSymbols(l, symbols, src, 0, nil)
	if err != nil {
		t.Fatalf("parser init: %v", err)
	}

	if _, err := p.ParseGlobals(t.Context(), "test.cog"); err != nil {
		t.Fatalf("ParseGlobals: %v", err)
	}

	// Simulate populating exports (as cmd/main.go does).
	imp, ok := symbols.ResolveCogImport("geom")
	if !ok {
		t.Fatal("expected cog import 'geom'")
	}

	imp.ExportValues["Distance"] = parser.Symbol{
		Identifier: &ast.Identifier{
			Token: tokens.Token{
				Type:    tokens.Identifier,
				Literal: "Distance",
			},
			ValueType: &types.Procedure{Function: true, Parameters: []*types.Parameter{{Name: "a", Type: types.Basics[types.Float64]}, {Name: "b", Type: types.Basics[types.Float64]}}, ReturnType: types.Basics[types.Float64]},
			Exported:  true,
		},
	}
	imp.ExportValues["Pi"] = parser.Symbol{
		Identifier: &ast.Identifier{
			Token: tokens.Token{
				Type:    tokens.Identifier,
				Literal: "Pi",
			},
			ValueType: types.Basics[types.Float64],
			Exported:  true,
		},
	}

	// Verify exports are accessible.
	if len(imp.ExportValues) != 2 {
		t.Fatalf("expected 2 exports, got %d", len(imp.ExportValues))
	}

	if _, ok := imp.ExportValues["Distance"]; !ok {
		t.Error("missing export Distance")
	}

	if _, ok := imp.ExportValues["Pi"]; !ok {
		t.Error("missing export Pi")
	}
}
