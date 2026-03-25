package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"sort"
	"strings"

	"github.com/samborkent/cog/internal/ast"
	"github.com/samborkent/cog/internal/lexer"
	"github.com/samborkent/cog/internal/parser"
	"github.com/samborkent/cog/internal/tokens"
	"github.com/samborkent/cog/internal/transpiler"
)

var (
	fileName string
	debug    bool
	write    bool
)

func main() {
	flag.StringVar(&fileName, "file", "", "Name of .cog file or directory containing .cog files.")
	flag.BoolVar(&debug, "debug", false, "Enable debug parser mode.")
	flag.BoolVar(&write, "write", false, "Write to file.")
	flag.Parse()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, os.Kill)
	defer stop()

	if fileName == "" {
		panic("missing file or directory name")
	}

	files := discoverFiles(fileName)

	// Determine project root: the directory of the entry package.
	// Import paths are resolved relative to this root.
	projectRoot := filepath.Dir(files[0])

	runProject(ctx, projectRoot, files)
}

// discoverFiles resolves the input flag to a sorted list of .cog file paths.
// If a single .cog file is given, only that file is returned.
// If a directory is given, all .cog files in that directory are returned.
func discoverFiles(input string) []string {
	input = filepath.Clean(input)

	info, err := os.Stat(input)
	if err != nil {
		panic(fmt.Errorf("cannot access %q: %w", input, err))
	}

	// Single file: return just that file.
	if !info.IsDir() {
		if !strings.HasSuffix(input, ".cog") {
			panic("invalid file extension, must be .cog")
		}
		return []string{input}
	}

	// Directory: scan for all .cog files.
	entries, err := os.ReadDir(input)
	if err != nil {
		panic(fmt.Errorf("reading directory %q: %w", input, err))
	}

	var files []string
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".cog") {
			continue
		}
		files = append(files, filepath.Join(input, entry.Name()))
	}

	if len(files) == 0 {
		panic(fmt.Errorf("no .cog files found in %q", input))
	}

	sort.Strings(files)

	return files
}

// lexFile lexes a single .cog file and returns its token stream.
func lexFile(ctx context.Context, path string) ([]tokens.Token, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("opening %q: %w", path, err)
	}
	defer func() { _ = file.Close() }()

	l := lexer.NewLexer(file)

	toks, err := l.Parse(ctx)
	if err != nil {
		return nil, fmt.Errorf("lexing %q: %w", path, err)
	}

	return toks, nil
}

// compiledPackage holds the output of compiling a single cog package.
type compiledPackage struct {
	importPath string      // relative import path (empty for the entry package)
	pkgName    string      // Go package name
	files      []lexedFile // original file paths
	astFiles   []*ast.File // parsed ASTs
	symbols    *parser.SymbolTable
}

type lexedFile struct {
	path   string
	tokens []tokens.Token
}

// runProject compiles the entry package and all its imported packages.
func runProject(ctx context.Context, projectRoot string, entryFiles []string) {
	// Step 1: Lex and validate the entry package.
	entryLexed, entryPkgName := lexAndValidate(ctx, entryFiles)
	if entryLexed == nil {
		return
	}

	// The Go module name for the transpiled project matches the entry package name.
	goModuleName := entryPkgName

	// Step 2: FindGlobals on the entry package (discovers globals + import paths).
	entrySymbols := parser.NewSymbolTable()
	entryParsers := findGlobals(ctx, entryLexed, entrySymbols)
	if entryParsers == nil {
		return
	}

	// Step 3: Process imported packages.
	importedPkgs := make(map[string]*compiledPackage) // key: import path

	for _, imp := range entrySymbols.CogImports() {
		pkg := compileImportedPackage(ctx, projectRoot, imp.Path)
		if pkg == nil {
			fmt.Printf("failed to compile imported package %q\n", imp.Path)
			return
		}

		importedPkgs[imp.Path] = pkg

		// Populate the entry package's import exports from the imported package.
		populateImportExports(imp, pkg.symbols)
	}

	// Step 4: Full parse the entry package (now with import exports available).
	entryASTs := make([]*ast.File, len(entryLexed))

	for i, lf := range entryLexed {
		f, err := entryParsers[i].ParseOnly(ctx, lf.path)
		if err != nil {
			fmt.Println(err.Error())
		}

		entryASTs[i] = f

		if !write {
			fmt.Printf("--- %s ---\n%s\n\n", lf.path, f)

			if err != nil {
				return
			}
		}
	}

	// Step 5: Transpile and output.
	entryPkg := &compiledPackage{
		pkgName:  entryPkgName,
		files:    entryLexed,
		astFiles: entryASTs,
		symbols:  entrySymbols,
	}

	outputProject(goModuleName, entryPkg, importedPkgs)
}

// lexAndValidate lexes all files and validates they declare the same package.
func lexAndValidate(ctx context.Context, files []string) ([]lexedFile, string) {
	lexed := make([]lexedFile, 0, len(files))

	for _, path := range files {
		toks, err := lexFile(ctx, path)
		if err != nil {
			fmt.Println(err.Error())
			return nil, ""
		}

		lexed = append(lexed, lexedFile{path: path, tokens: toks})
	}

	dirName := filepath.Base(filepath.Dir(files[0]))
	var pkgName string

	for _, lf := range lexed {
		if len(lf.tokens) < 2 || lf.tokens[0].Type != tokens.Package {
			fmt.Printf("%s: missing package declaration\n", lf.path)
			return nil, ""
		}

		name := lf.tokens[1].Literal

		if pkgName == "" {
			pkgName = name

			if pkgName != "main" && pkgName != dirName {
				fmt.Printf("%s: package %q does not match directory name %q\n", lf.path, pkgName, dirName)
				return nil, ""
			}
		} else if name != pkgName {
			fmt.Printf("%s: declares package %q, but other files use %q\n", lf.path, name, pkgName)
			return nil, ""
		}
	}

	return lexed, pkgName
}

// findGlobals runs FindGlobals on all files with a shared symbol table.
func findGlobals(ctx context.Context, lexed []lexedFile, symbols *parser.SymbolTable) []*parser.Parser {
	parsers := make([]*parser.Parser, len(lexed))

	for i, lf := range lexed {
		p, err := parser.NewParserWithSymbols(lf.tokens, symbols, debug)
		if err != nil {
			fmt.Println(err.Error())
			return nil
		}

		p.FindGlobals(ctx)
		parsers[i] = p
	}

	return parsers
}

// compileImportedPackage discovers, lexes, parses, and validates an imported package.
func compileImportedPackage(ctx context.Context, projectRoot, importPath string) *compiledPackage {
	pkgDir := filepath.Join(projectRoot, filepath.FromSlash(importPath))
	files := discoverFiles(pkgDir)

	lexed, pkgName := lexAndValidate(ctx, files)
	if lexed == nil {
		return nil
	}

	symbols := parser.NewSymbolTable()
	parsers := findGlobals(ctx, lexed, symbols)
	if parsers == nil {
		return nil
	}

	// TODO: handle transitive imports for imported packages.

	astFiles := make([]*ast.File, len(lexed))
	for i, lf := range lexed {
		f, err := parsers[i].ParseOnly(ctx, lf.path)
		if err != nil {
			fmt.Println(err.Error())
			return nil
		}

		astFiles[i] = f
	}

	return &compiledPackage{
		importPath: importPath,
		pkgName:    pkgName,
		files:      lexed,
		astFiles:   astFiles,
		symbols:    symbols,
	}
}

// populateImportExports fills a CogImport's Exports map from the imported package's symbol table.
func populateImportExports(imp *parser.CogImport, symbols *parser.SymbolTable) {
	symbols.ForEachGlobal(func(name string, sym parser.Symbol) {
		if sym.Identifier.Exported {
			imp.Exports[name] = sym
		}
	})
}

// outputProject transpiles and writes all packages.
func outputProject(goModuleName string, entry *compiledPackage, imported map[string]*compiledPackage) {
	if write {
		if err := os.MkdirAll("tmp", 0o700); err != nil {
			panic(fmt.Errorf("creating temp dir: %w", err))
		}
	}

	// Transpile and output imported packages first.
	for _, pkg := range imported {
		transpileAndOutput(goModuleName, pkg)
	}

	// Transpile and output the entry package.
	transpileAndOutput(goModuleName, entry)

	if write {
		// Write go.mod so `go run .` works from tmp/.
		// Only declare the module and Go version; `go mod tidy` resolves all dependencies.
		gomod := fmt.Sprintf("module %s\n\ngo 1.26.1\n", goModuleName)
		if err := os.WriteFile(filepath.Join("tmp", "go.mod"), []byte(gomod), 0o600); err != nil {
			panic(fmt.Errorf("writing go.mod: %w", err))
		}

		// Run go mod tidy to resolve all dependencies.
		tidy := exec.Command("go", "mod", "tidy")
		tidy.Dir = "tmp"
		if out, err := tidy.CombinedOutput(); err != nil {
			panic(fmt.Errorf("go mod tidy: %s\n%w", out, err))
		}
	}
}

// transpileAndOutput transpiles a single package and writes/prints its Go files.
func transpileAndOutput(goModuleName string, pkg *compiledPackage) {
	t := transpiler.NewTranspilerWithModule(goModuleName, pkg.astFiles...)

	gofiles, err := t.TranspileFiles()
	if err != nil {
		fmt.Println(err.Error())
		return
	}

	// Determine output directory.
	outDir := "tmp"
	if pkg.importPath != "" {
		outDir = filepath.Join("tmp", filepath.FromSlash(pkg.importPath))
	}

	if write {
		if err := os.MkdirAll(outDir, 0o700); err != nil {
			panic(fmt.Errorf("creating output dir: %w", err))
		}
	}

	for i, lf := range pkg.files {
		outName := filepath.Base(lf.path)
		outName = strings.TrimSuffix(outName, ".cog") + ".go"

		if write {
			outFile, err := os.Create(filepath.Join(outDir, outName))
			if err != nil {
				panic(fmt.Errorf("creating output file: %w", err))
			}

			if err := t.Print(outFile, gofiles[i]); err != nil {
				_ = outFile.Close()
				panic(fmt.Errorf("printing output: %w", err))
			}

			_ = outFile.Close()
		} else {
			fmt.Printf("--- %s ---\n", filepath.Join(outDir, outName))

			if err := t.Print(os.Stdout, gofiles[i]); err != nil {
				panic(fmt.Errorf("printing output: %w", err))
			}

			fmt.Println()
		}
	}
}
