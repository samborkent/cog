package cog_test

import (
	"bytes"
	"context"
	"fmt"
	goast "go/ast"
	"maps"
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

// lexedFile holds lexer output for a single .cog file.
type lexedFile struct {
	path  string
	lexer *lexer.Lexer
}

// packageFiles groups .cog source files by package directory.
type packageFiles struct {
	dir   string            // directory path relative to example root
	files map[string]string // relPath -> source content
}

// loadExamplePackages discovers all .cog files under example/ and groups them
// by package directory. Returns the entry package (example/) and any imported
// sub-packages in deterministic order.
func loadExamplePackages(t testing.TB) (entry packageFiles, imported []packageFiles) {
	t.Helper()

	exampleDir := "example"
	if _, err := os.Stat(exampleDir); os.IsNotExist(err) {
		exampleDir = filepath.Join("..", "example")
	}

	pkgMap := make(map[string]map[string]string) // dir -> (relPath -> content)

	err := filepath.Walk(exampleDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() || !strings.HasSuffix(info.Name(), ".cog") {
			return nil
		}

		relPath, err := filepath.Rel(exampleDir, path)
		if err != nil {
			return err
		}

		dir := filepath.Dir(relPath)

		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		if pkgMap[dir] == nil {
			pkgMap[dir] = make(map[string]string)
		}

		pkgMap[dir][relPath] = string(content)

		return nil
	})
	if err != nil {
		t.Fatalf("walking example dir: %v", err)
	}

	entryFiles, ok := pkgMap["."]
	if !ok || len(entryFiles) == 0 {
		t.Fatal("no entry package files found in example/")
	}

	entry = packageFiles{dir: ".", files: entryFiles}

	// Collect imported packages sorted by directory path.
	var importDirs []string
	for dir := range pkgMap {
		if dir != "." {
			importDirs = append(importDirs, dir)
		}
	}

	sort.Strings(importDirs)

	for _, dir := range importDirs {
		imported = append(imported, packageFiles{dir: dir, files: pkgMap[dir]})
	}

	return entry, imported
}

// sortedKeys returns the keys of a map in sorted order.
func sortedKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// lexPackage lexes all files in a package and returns tokens in sorted order.
func lexPackage(t testing.TB, pkg packageFiles) []lexedFile {
	t.Helper()

	names := sortedKeys(pkg.files)
	lexed := make([]lexedFile, len(names))

	for i, name := range names {
		file := pkg.files[name]
		l := lexer.New(strings.NewReader(file), uint32(len(file)), false)

		lexed[i] = lexedFile{path: name, lexer: l}
	}

	return lexed
}

// parseGlobals runs ParseGlobals on all files with a shared symbol table,
// returning the parsers (for subsequent ParseBodies) and the ASTs.
func parseGlobals(t testing.TB, lexed []lexedFile, symbols *parser.SymbolTable) ([]*parser.Parser, []*ast.AST) {
	t.Helper()

	parsers := make([]*parser.Parser, len(lexed))
	astFiles := make([]*ast.AST, len(lexed))

	for i, lf := range lexed {
		p, err := parser.NewParserWithSymbols(lf.lexer, symbols, lf.path, uint16(i), nil)
		if err != nil {
			t.Fatalf("parser init (%s): %v", lf.path, err)
		}

		f, err := p.ParseGlobals(t.Context(), lf.path)
		if err != nil {
			t.Fatalf("parser globals (%s): %v", lf.path, err)
		}

		parsers[i] = p
		astFiles[i] = f
	}

	return parsers, astFiles
}

// compilePackage compiles a single package: ParseGlobals + ParseBodies.
func compilePackage(t testing.TB, pkg packageFiles) ([]*ast.AST, *parser.SymbolTable) {
	t.Helper()

	lexed := lexPackage(t, pkg)
	symbols := parser.NewSymbolTable()
	parsers, astFiles := parseGlobals(t, lexed, symbols)

	for i, lf := range lexed {
		if err := parsers[i].ParseBodies(t.Context()); err != nil {
			t.Fatalf("parser bodies (%s): %v", lf.path, err)
		}
	}

	return astFiles, symbols
}

// populateImportExports fills a CogImport's Exports from the imported
// package's symbol table.
func populateImportExports(imp *parser.CogImport, symbols *parser.SymbolTable) {
	symbols.ForEachGlobal(func(name string, sym parser.Symbol) {
		if sym.Identifier.Exported {
			imp.Exports[name] = sym
		}
	})
}

// compileProject compiles the full example project: entry package + imports.
// It mirrors the flow in cmd/main.go: lex all → ParseGlobals → compile
// imports → populate exports → ParseBodies entry files.
func compileProject(t testing.TB) ([]*ast.AST, *parser.SymbolTable) {
	t.Helper()

	entry, imported := loadExamplePackages(t)

	// Phase 1: Lex and ParseGlobals for the entry package.
	entryLexed := lexPackage(t, entry)
	entrySymbols := parser.NewSymbolTable()
	entryParsers, astFiles := parseGlobals(t, entryLexed, entrySymbols)

	// Phase 2: Compile imported packages and populate exports.
	for _, pkg := range imported {
		_, pkgSymbols := compilePackage(t, pkg)

		// Find the CogImport in the entry symbol table that matches this package.
		for _, imp := range entrySymbols.CogImports().All() {
			if imp.Path == pkg.dir || imp.Name == filepath.Base(pkg.dir) {
				populateImportExports(imp, pkgSymbols)
			}
		}
	}

	// Phase 3: ParseBodies for the entry package.
	for i, lf := range entryLexed {
		if err := entryParsers[i].ParseBodies(t.Context()); err != nil {
			t.Fatalf("parser bodies (%s): %v", lf.path, err)
		}
	}

	return astFiles, entrySymbols
}

var lexingRangeTok tokens.Token

func BenchmarkLexing(b *testing.B) {
	b.ReportAllocs()

	for b.Loop() {
		b.StopTimer()
		entry, imported := loadExamplePackages(b)

		allFiles := make(map[string]string)
		maps.Copy(allFiles, entry.files)

		for _, pkg := range imported {
			maps.Copy(allFiles, pkg.files)
		}

		names := sortedKeys(allFiles)
		b.StartTimer()

		for _, name := range names {
			file := allFiles[name]
			l := lexer.New(strings.NewReader(file), uint32(len(file)), false)

			for {
				if b.Context().Err() != nil {
					return
				}

				tok := l.Peek(0)

				lexingRangeTok = tok

				if tok.Type == tokens.EOF {
					break
				}

				l.Step()
			}
		}
	}
}

var parsingAST *ast.AST

// BenchmarkParsing benchmarks the parser phase (lex + ParseGlobals + ParseBodies)
// with proper multi-file symbol sharing and import resolution.
func BenchmarkParsing(b *testing.B) {
	b.ReportAllocs()

	for b.Loop() {
		b.StopTimer()
		entry, imported := loadExamplePackages(b)
		b.StartTimer()

		// Lex + ParseGlobals for the entry package.
		entryLexed := lexPackage(b, entry)
		entrySymbols := parser.NewSymbolTable()
		entryParsers, astFiles := parseGlobals(b, entryLexed, entrySymbols)

		// Compile imported packages and populate exports.
		for _, pkg := range imported {
			_, pkgSymbols := compilePackage(b, pkg)

			for _, imp := range entrySymbols.CogImports().All() {
				if imp.Path == pkg.dir || imp.Name == filepath.Base(pkg.dir) {
					populateImportExports(imp, pkgSymbols)
				}
			}
		}

		// ParseBodies for entry files.
		for i, lf := range entryLexed {
			if err := entryParsers[i].ParseBodies(b.Context()); err != nil {
				b.Fatalf("parser bodies (%s): %v", lf.path, err)
			}

			parsingAST = astFiles[i]
		}
	}
}

var transpiledFiles []*goast.File

// BenchmarkTranspiling benchmarks the transpiler phase (AST -> Go AST).
func BenchmarkTranspiling(b *testing.B) {
	b.ReportAllocs()

	for b.Loop() {
		b.StopTimer()
		astFiles, _ := compileProject(b)
		b.StartTimer()

		tr := transpiler.NewTranspiler(ast.MergeASTs(astFiles...))

		gofiles, err := tr.TranspileFiles()
		if err != nil {
			b.Fatalf("transpile error: %v", err)
		}

		transpiledFiles = gofiles
	}
}

var printingOutput string

// BenchmarkPrinting benchmarks the print phase (Go AST -> Go source code).
func BenchmarkPrinting(b *testing.B) {
	b.ReportAllocs()

	for b.Loop() {
		b.StopTimer()
		astFiles, _ := compileProject(b)

		tr := transpiler.NewTranspiler(ast.MergeASTs(astFiles...))

		gofiles, err := tr.TranspileFiles()
		if err != nil {
			b.Fatalf("transpile error: %v", err)
		}
		b.StartTimer()

		for _, gofile := range gofiles {
			var buf bytes.Buffer

			if err := tr.Print(&buf, gofile); err != nil {
				b.Fatalf("print error: %v", err)
			}

			printingOutput = buf.String()
		}
	}
}

var transpiledOutput string

// BenchmarkTranspileAndPrint benchmarks transpile + print combined.
func BenchmarkTranspileAndPrint(b *testing.B) {
	b.ReportAllocs()

	for b.Loop() {
		b.StopTimer()
		astFiles, _ := compileProject(b)
		b.StartTimer()

		tr := transpiler.NewTranspiler(ast.MergeASTs(astFiles...))

		gofiles, err := tr.TranspileFiles()
		if err != nil {
			b.Fatalf("transpile error: %v", err)
		}

		for _, gofile := range gofiles {
			var buf bytes.Buffer
			if err := tr.Print(&buf, gofile); err != nil {
				b.Fatalf("print error: %v", err)
			}

			transpiledOutput = buf.String()
		}
	}
}

var pipelineOutput string

// BenchmarkFullPipeline benchmarks the entire pipeline: lex + parse + transpile + print.
func BenchmarkFullPipeline(b *testing.B) {
	b.ReportAllocs()

	for b.Loop() {
		b.StopTimer()
		entry, imported := loadExamplePackages(b)
		b.StartTimer()

		// Lex + ParseGlobals for the entry package.
		entryLexed := lexPackage(b, entry)
		entrySymbols := parser.NewSymbolTable()
		entryParsers, astFiles := parseGlobals(b, entryLexed, entrySymbols)

		// Compile imported packages and populate exports.
		for _, pkg := range imported {
			_, pkgSymbols := compilePackage(b, pkg)

			for _, imp := range entrySymbols.CogImports().All() {
				if imp.Path == pkg.dir || imp.Name == filepath.Base(pkg.dir) {
					populateImportExports(imp, pkgSymbols)
				}
			}
		}

		// ParseBodies for entry files.
		for i, lf := range entryLexed {
			if err := entryParsers[i].ParseBodies(b.Context()); err != nil {
				b.Fatalf("parser bodies (%s): %v", lf.path, err)
			}

			_ = astFiles[i]
		}

		// Transpile + print.
		tr := transpiler.NewTranspiler(ast.MergeASTs(astFiles...))

		gofiles, err := tr.TranspileFiles()
		if err != nil {
			b.Fatalf("transpile error: %v", err)
		}

		for _, gofile := range gofiles {
			var buf bytes.Buffer
			if err := tr.Print(&buf, gofile); err != nil {
				b.Fatalf("print error: %v", err)
			}

			pipelineOutput = buf.String()
		}
	}
}

// BenchmarkGoBuild benchmarks the Go build step for transpiled code.
func BenchmarkGoBuild(b *testing.B) {
	b.ReportAllocs()

	for b.Loop() {
		b.StopTimer()

		// b.TempDir() returns a unique directory per call, so capture it once.
		tmpDir := b.TempDir()

		_, imported := loadExamplePackages(b)

		// Compile and write imported packages.
		for _, pkg := range imported {
			pkgASTs, _ := compilePackage(b, pkg)
			pkgTr := transpiler.NewTranspilerWithModule("main", ast.MergeASTs(pkgASTs...))

			pkgGoFiles, err := pkgTr.TranspileFiles()
			if err != nil {
				b.Fatalf("transpile import %s: %v", pkg.dir, err)
			}

			outDir := filepath.Join(tmpDir, filepath.FromSlash(pkg.dir))
			if err := os.MkdirAll(outDir, 0o700); err != nil {
				b.Fatalf("mkdir %s: %v", outDir, err)
			}

			for i, gofile := range pkgGoFiles {
				var buf bytes.Buffer
				if err := pkgTr.Print(&buf, gofile); err != nil {
					b.Fatalf("print import error: %v", err)
				}

				if err := os.WriteFile(filepath.Join(outDir, fmt.Sprintf("file%d.go", i)), buf.Bytes(), 0o600); err != nil {
					b.Fatalf("writing import file: %v", err)
				}
			}
		}

		// Compile the full project (entry package with imports resolved).
		astFiles, _ := compileProject(b)
		tr := transpiler.NewTranspilerWithModule("main", ast.MergeASTs(astFiles...))

		gofiles, err := tr.TranspileFiles()
		if err != nil {
			b.Fatalf("transpile error: %v", err)
		}

		for i, gofile := range gofiles {
			var buf bytes.Buffer
			if err := tr.Print(&buf, gofile); err != nil {
				b.Fatalf("print error: %v", err)
			}

			if err := os.WriteFile(filepath.Join(tmpDir, fmt.Sprintf("file%d.go", i)), buf.Bytes(), 0o600); err != nil {
				b.Fatalf("writing file: %v", err)
			}
		}

		// Write go.mod with proper dependencies.
		root := projectRoot(b)

		goMod := fmt.Sprintf(`module main

go 1.26.3

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
`, root, filepath.Join(root, "..", "adaptive-gc"), goModCacheDir(b, "github.com/pbnjay/memory"))

		if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goMod), 0o600); err != nil {
			b.Fatalf("writing go.mod: %v", err)
		}

		// Run go mod tidy to resolve transitive dependencies.
		tidy := exec.Command("go", "mod", "tidy")
		tidy.Dir = tmpDir

		if out, err := tidy.CombinedOutput(); err != nil {
			b.Fatalf("go mod tidy failed: %v\n%s", err, out)
		}
		b.StartTimer()

		os.RemoveAll(filepath.Join(tmpDir, "bin"))

		buildCtx, cancel := context.WithTimeout(b.Context(), 30*time.Second)
		cmd := exec.CommandContext(buildCtx, "go", "build", "-o", "bin", ".")
		cmd.Dir = tmpDir

		out, err := cmd.CombinedOutput()
		cancel()

		if err != nil {
			b.Fatalf("go build failed: %v\n%s", err, out)
		}
	}
}

// projectRoot returns the absolute path to the repository root.
func projectRoot(t testing.TB) string {
	t.Helper()

	root, err := filepath.Abs(".")
	if err != nil {
		t.Fatalf("getting project root: %v", err)
	}

	return root
}

// goModCacheDir returns the on-disk directory for a module from the Go module cache.
func goModCacheDir(t testing.TB, module string) string {
	t.Helper()

	out, err := exec.Command("go", "mod", "download", "-json", module+"@latest").CombinedOutput()
	if err != nil {
		t.Fatalf("go mod download %s: %v\n%s", module, err, out)
	}

	// Extract "Dir" from the JSON output.
	for line := range strings.SplitSeq(string(out), "\n") {
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

	t.Fatalf("could not find Dir in go mod download output for %s", module)

	return ""
}

var largeFileOutput string

// BenchmarkLargeFile benchmarks the full pipeline for the largest file (example.cog).
func BenchmarkLargeFile(b *testing.B) {
	b.ReportAllocs()

	for b.Loop() {
		b.StopTimer()
		entry, imported := loadExamplePackages(b)
		b.StartTimer()

		// Full project compile — the pipeline cost is dominated by example.cog.
		entryLexed := lexPackage(b, entry)
		entrySymbols := parser.NewSymbolTable()
		entryParsers, astFiles := parseGlobals(b, entryLexed, entrySymbols)

		for _, pkg := range imported {
			_, pkgSymbols := compilePackage(b, pkg)

			for _, imp := range entrySymbols.CogImports().All() {
				if imp.Path == pkg.dir || imp.Name == filepath.Base(pkg.dir) {
					populateImportExports(imp, pkgSymbols)
				}
			}
		}

		for i, lf := range entryLexed {
			if err := entryParsers[i].ParseBodies(b.Context()); err != nil {
				b.Fatalf("parser bodies (%s): %v", lf.path, err)
			}

			_ = astFiles[i]
		}

		tr := transpiler.NewTranspiler(ast.MergeASTs(astFiles...))

		gofiles, err := tr.TranspileFiles()
		if err != nil {
			b.Fatalf("transpile error: %v", err)
		}

		for _, gofile := range gofiles {
			var buf bytes.Buffer

			if err := tr.Print(&buf, gofile); err != nil {
				b.Fatalf("print error: %v", err)
			}

			largeFileOutput = buf.String()
		}
	}
}

var multiFileOutput string

// BenchmarkMultiFileTranspile benchmarks transpiling multiple files together
// using TranspileFiles (one Go file per input file).
func BenchmarkMultiFileTranspile(b *testing.B) {
	b.ReportAllocs()

	for b.Loop() {
		b.StopTimer()
		astFiles, _ := compileProject(b)
		b.StartTimer()

		tr := transpiler.NewTranspiler(ast.MergeASTs(astFiles...))

		gofiles, err := tr.TranspileFiles()
		if err != nil {
			b.Fatalf("transpile error: %v", err)
		}

		for _, gofile := range gofiles {
			var buf bytes.Buffer
			if err := tr.Print(&buf, gofile); err != nil {
				b.Fatalf("print error: %v", err)
			}

			multiFileOutput = buf.String()
		}
	}
}
