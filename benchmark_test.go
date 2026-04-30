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

// lexedFile holds lexer output for a single .cog file.
type lexedFile struct {
	path   string
	tokens []tokens.Token
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
func lexPackage(ctx context.Context, t testing.TB, pkg packageFiles) []lexedFile {
	t.Helper()

	names := sortedKeys(pkg.files)
	lexed := make([]lexedFile, len(names))

	for i, name := range names {
		l := lexer.NewLexerWithFileID(strings.NewReader(pkg.files[name]), uint16(i))

		toks, err := l.Parse(ctx)
		if err != nil {
			t.Fatalf("lexer error (%s): %v", name, err)
		}

		lexed[i] = lexedFile{path: name, tokens: toks}
	}

	return lexed
}

// findGlobals runs FindGlobals on all files with a shared symbol table,
// returning the parsers for subsequent ParseOnly calls.
func findGlobals(ctx context.Context, t testing.TB, lexed []lexedFile, symbols *parser.SymbolTable) []*parser.Parser {
	t.Helper()

	parsers := make([]*parser.Parser, len(lexed))

	for i, lf := range lexed {
		p, err := parser.NewParserWithSymbols(lf.tokens, symbols, false, lf.path, uint16(i))
		if err != nil {
			t.Fatalf("parser init (%s): %v", lf.path, err)
		}

		p.FindGlobals(ctx)

		parsers[i] = p
	}

	return parsers
}

// compilePackage compiles a single package: FindGlobals + ParseOnly.
func compilePackage(ctx context.Context, t testing.TB, pkg packageFiles) ([]*ast.AST, *parser.SymbolTable) {
	t.Helper()

	lexed := lexPackage(ctx, t, pkg)
	symbols := parser.NewSymbolTable()
	parsers := findGlobals(ctx, t, lexed, symbols)

	astFiles := make([]*ast.AST, len(lexed))

	for i, lf := range lexed {
		f, err := parsers[i].ParseOnly(ctx, lf.path)
		if err != nil {
			t.Fatalf("parser error (%s): %v", lf.path, err)
		}

		astFiles[i] = f
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
// It mirrors the flow in cmd/main.go: lex all → FindGlobals → compile
// imports → populate exports → ParseOnly entry files.
func compileProject(ctx context.Context, t testing.TB) ([]*ast.AST, *parser.SymbolTable) {
	t.Helper()

	entry, imported := loadExamplePackages(t)

	// Phase 1: Lex and FindGlobals for the entry package.
	entryLexed := lexPackage(ctx, t, entry)
	entrySymbols := parser.NewSymbolTable()
	entryParsers := findGlobals(ctx, t, entryLexed, entrySymbols)

	// Phase 2: Compile imported packages and populate exports.
	for _, pkg := range imported {
		_, pkgSymbols := compilePackage(ctx, t, pkg)

		// Find the CogImport in the entry symbol table that matches this package.
		for _, imp := range entrySymbols.CogImports() {
			if imp.Path == pkg.dir || imp.Name == filepath.Base(pkg.dir) {
				populateImportExports(imp, pkgSymbols)
			}
		}
	}

	// Phase 3: Full parse the entry package.
	astFiles := make([]*ast.AST, len(entryLexed))

	for i, lf := range entryLexed {
		f, err := entryParsers[i].ParseOnly(ctx, lf.path)
		if err != nil {
			t.Fatalf("parser error (%s): %v", lf.path, err)
		}

		astFiles[i] = f
	}

	return astFiles, entrySymbols
}

// BenchmarkLexing benchmarks just the lexer phase across all example files.
func BenchmarkLexing(b *testing.B) {
	ctx := context.Background()
	entry, imported := loadExamplePackages(b)

	// Collect all files across all packages for lexing.
	allFiles := make(map[string]string)
	for name, src := range entry.files {
		allFiles[name] = src
	}

	for _, pkg := range imported {
		for name, src := range pkg.files {
			allFiles[name] = src
		}
	}

	names := sortedKeys(allFiles)

	b.ResetTimer()
	b.ReportAllocs()

	for b.Loop() {
		for i, name := range names {
			l := lexer.NewLexerWithFileID(strings.NewReader(allFiles[name]), uint16(i))

			toks, err := l.Parse(ctx)
			if err != nil {
				b.Fatalf("lexer error (%s): %v", name, err)
			}

			_ = toks
		}
	}
}

// BenchmarkParsing benchmarks the parser phase (lex + FindGlobals + ParseOnly)
// with proper multi-file symbol sharing and import resolution.
func BenchmarkParsing(b *testing.B) {
	ctx := context.Background()
	entry, imported := loadExamplePackages(b)

	b.ResetTimer()
	b.ReportAllocs()

	for b.Loop() {
		// Lex + FindGlobals for the entry package.
		entryLexed := lexPackage(ctx, b, entry)
		entrySymbols := parser.NewSymbolTable()
		entryParsers := findGlobals(ctx, b, entryLexed, entrySymbols)

		// Compile imported packages and populate exports.
		for _, pkg := range imported {
			_, pkgSymbols := compilePackage(ctx, b, pkg)

			for _, imp := range entrySymbols.CogImports() {
				if imp.Path == pkg.dir || imp.Name == filepath.Base(pkg.dir) {
					populateImportExports(imp, pkgSymbols)
				}
			}
		}

		// ParseOnly entry files.
		for i, lf := range entryLexed {
			f, err := entryParsers[i].ParseOnly(ctx, lf.path)
			if err != nil {
				b.Fatalf("parser error (%s): %v", lf.path, err)
			}

			_ = f
		}
	}
}

// BenchmarkTranspiling benchmarks the transpiler phase (AST -> Go AST).
func BenchmarkTranspiling(b *testing.B) {
	ctx := context.Background()
	astFiles, _ := compileProject(ctx, b)

	b.ResetTimer()
	b.ReportAllocs()

	for b.Loop() {
		tr := transpiler.NewTranspiler(ast.MergeASTs(astFiles...))

		gofiles, err := tr.TranspileFiles()
		if err != nil {
			b.Fatalf("transpile error: %v", err)
		}

		_ = gofiles
	}
}

// BenchmarkPrinting benchmarks the print phase (Go AST -> Go source code).
func BenchmarkPrinting(b *testing.B) {
	ctx := context.Background()
	astFiles, _ := compileProject(ctx, b)

	tr := transpiler.NewTranspiler(ast.MergeASTs(astFiles...))

	gofiles, err := tr.TranspileFiles()
	if err != nil {
		b.Fatalf("transpile error: %v", err)
	}

	b.ResetTimer()
	b.ReportAllocs()

	for b.Loop() {
		for _, gofile := range gofiles {
			var buf bytes.Buffer
			if err := tr.Print(&buf, gofile); err != nil {
				b.Fatalf("print error: %v", err)
			}

			_ = buf.String()
		}
	}
}

// BenchmarkTranspileAndPrint benchmarks transpile + print combined.
func BenchmarkTranspileAndPrint(b *testing.B) {
	ctx := context.Background()
	astFiles, _ := compileProject(ctx, b)

	b.ResetTimer()
	b.ReportAllocs()

	for b.Loop() {
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

			_ = buf.String()
		}
	}
}

// BenchmarkFullPipeline benchmarks the entire pipeline: lex + parse + transpile + print.
func BenchmarkFullPipeline(b *testing.B) {
	ctx := context.Background()
	entry, imported := loadExamplePackages(b)

	b.ResetTimer()
	b.ReportAllocs()

	for b.Loop() {
		// Lex + FindGlobals for the entry package.
		entryLexed := lexPackage(ctx, b, entry)
		entrySymbols := parser.NewSymbolTable()
		entryParsers := findGlobals(ctx, b, entryLexed, entrySymbols)

		// Compile imported packages and populate exports.
		for _, pkg := range imported {
			_, pkgSymbols := compilePackage(ctx, b, pkg)

			for _, imp := range entrySymbols.CogImports() {
				if imp.Path == pkg.dir || imp.Name == filepath.Base(pkg.dir) {
					populateImportExports(imp, pkgSymbols)
				}
			}
		}

		// ParseOnly entry files.
		astFiles := make([]*ast.AST, len(entryLexed))

		for i, lf := range entryLexed {
			f, err := entryParsers[i].ParseOnly(ctx, lf.path)
			if err != nil {
				b.Fatalf("parser error (%s): %v", lf.path, err)
			}

			astFiles[i] = f
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

			_ = buf.String()
		}
	}
}

// BenchmarkGoBuild benchmarks the Go build step for transpiled code.
func BenchmarkGoBuild(b *testing.B) {
	ctx := context.Background()
	tmpDir := b.TempDir()

	_, imported := loadExamplePackages(b)

	// Compile and write imported packages.
	for _, pkg := range imported {
		pkgASTs, _ := compilePackage(ctx, b, pkg)
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
	astFiles, _ := compileProject(ctx, b)
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

	b.ResetTimer()
	b.ReportAllocs()

	for b.Loop() {
		os.RemoveAll(filepath.Join(tmpDir, "bin"))

		buildCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
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

	t.Fatalf("could not find Dir in go mod download output for %s", module)

	return ""
}

// BenchmarkLargeFile benchmarks the full pipeline for the largest file (example.cog).
func BenchmarkLargeFile(b *testing.B) {
	ctx := context.Background()
	entry, imported := loadExamplePackages(b)

	b.ResetTimer()
	b.ReportAllocs()

	for b.Loop() {
		// Full project compile — the pipeline cost is dominated by example.cog.
		entryLexed := lexPackage(ctx, b, entry)
		entrySymbols := parser.NewSymbolTable()
		entryParsers := findGlobals(ctx, b, entryLexed, entrySymbols)

		for _, pkg := range imported {
			_, pkgSymbols := compilePackage(ctx, b, pkg)

			for _, imp := range entrySymbols.CogImports() {
				if imp.Path == pkg.dir || imp.Name == filepath.Base(pkg.dir) {
					populateImportExports(imp, pkgSymbols)
				}
			}
		}

		astFiles := make([]*ast.AST, len(entryLexed))

		for i, lf := range entryLexed {
			f, err := entryParsers[i].ParseOnly(ctx, lf.path)
			if err != nil {
				b.Fatalf("parser error (%s): %v", lf.path, err)
			}

			astFiles[i] = f
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

			_ = buf.String()
		}
	}
}

// BenchmarkMultiFileTranspile benchmarks transpiling multiple files together
// using TranspileFiles (one Go file per input file).
func BenchmarkMultiFileTranspile(b *testing.B) {
	ctx := context.Background()
	astFiles, _ := compileProject(ctx, b)

	b.ResetTimer()
	b.ReportAllocs()

	for b.Loop() {
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

			_ = buf.String()
		}
	}
}
