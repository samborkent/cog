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
	fileName        string
	debug           bool
	write           bool
	replaceLocalCog bool
)

func main() {
	flag.StringVar(&fileName, "file", "", "Name of .cog/.cogs file or directory containing .cog files.")
	flag.BoolVar(&debug, "debug", false, "Enable debug parser mode.")
	flag.BoolVar(&write, "write", false, "Write to file.")
	flag.BoolVar(&replaceLocalCog, "replace-local-cog", false, "Add replace directive for local cog module in generated go.mod.")
	flag.Parse()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, os.Kill)
	defer stop()

	if fileName == "" {
		panic("missing file or directory name")
	}

	files := discoverFiles(fileName)

	// Script mode: single .cogs file.
	if strings.HasSuffix(files[0], ".cogs") {
		projectRoot := filepath.Dir(files[0])
		runScript(ctx, projectRoot, files[0], "")

		return
	}

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
		if !strings.HasSuffix(input, ".cog") && !strings.HasSuffix(input, ".cogs") {
			panic("invalid file extension, must be .cog or .cogs")
		}

		return []string{input}
	}

	// Directory: scan for all .cog files.
	entries, err := os.ReadDir(input)
	if err != nil {
		panic(fmt.Errorf("reading directory %q: %w", input, err))
	}

	files := make([]string, 0, len(entries))

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
func lexFile(ctx context.Context, path string, fileID uint16) ([]tokens.Token, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("opening %q: %w", path, err)
	}

	defer func() { _ = file.Close() }()

	l := lexer.NewLexerWithFileID(file, fileID)

	toks, err := l.Parse(ctx)
	if err != nil {
		return nil, fmt.Errorf("lexing %q: %w", path, err)
	}

	return toks, nil
}

// runScript compiles a single .cogs script file.
// Script files have no package declaration; the transpiled output is placed
// in cmd/{scriptName}/ with package main and a func main() wrapping the body.
// If goModuleName is empty, the script name is used and go.mod is written.
func runScript(ctx context.Context, projectRoot string, scriptPath string, goModuleName string) {
	toks, err := lexFile(ctx, scriptPath, 0)
	if err != nil {
		fmt.Println(err.Error())
		return
	}

	symbols := parser.NewSymbolTable()

	p, err := parser.NewScriptParserWithSymbols(toks, symbols, debug)
	if err != nil {
		fmt.Println(err.Error())
		return
	}

	p.FindGlobals(ctx)

	// Process imported packages.
	importedPkgs := make(map[string]*compiledPackage)

	for _, imp := range symbols.CogImports() {
		pkg := compileImportedPackage(ctx, projectRoot, imp.Path)
		if pkg == nil {
			fmt.Printf("failed to compile imported package %q\n", imp.Path)
			return
		}

		importedPkgs[imp.Path] = pkg
		populateImportExports(imp, pkg.symbols)
	}

	f, err := p.ParseOnly(ctx, scriptPath)
	if err != nil {
		fmt.Println(err.Error())
		return
	}

	// Determine script name from file name (without extension).
	scriptName := strings.TrimSuffix(filepath.Base(scriptPath), ".cogs")

	standalone := goModuleName == ""
	if standalone {
		goModuleName = scriptName
	}

	// Transpile imported packages first.
	if write {
		if err := os.MkdirAll("tmp", 0o700); err != nil {
			panic(fmt.Errorf("creating temp dir: %w", err))
		}
	}

	for _, pkg := range importedPkgs {
		transpileAndOutput(goModuleName, pkg)
	}

	// Transpile the script file.
	t := transpiler.NewTranspilerWithModule(goModuleName, []*ast.File{f})

	gofile, err := t.TranspileScript()
	if err != nil {
		fmt.Println(err.Error())
		return
	}

	outDir := filepath.Join("tmp", "cmd", scriptName)
	outName := "main.go"

	if write {
		if err := os.MkdirAll(outDir, 0o700); err != nil {
			panic(fmt.Errorf("creating output dir: %w", err))
		}

		outFile, err := os.Create(filepath.Join(outDir, outName))
		if err != nil {
			panic(fmt.Errorf("creating output file: %w", err))
		}

		if err := t.Print(outFile, gofile); err != nil {
			_ = outFile.Close()

			panic(fmt.Errorf("printing output: %w", err))
		}

		_ = outFile.Close()

		// Only write go.mod when running as a standalone script (not part of a project).
		if standalone {
			gomod := fmt.Sprintf("module %s\n\ngo 1.26.2\n", goModuleName)
			if replaceLocalCog {
				gomod += "\nreplace github.com/samborkent/cog => ./..\n"
			}

			if err := os.WriteFile(filepath.Join("tmp", "go.mod"), []byte(gomod), 0o600); err != nil {
				panic(fmt.Errorf("writing go.mod: %w", err))
			}

			tidy := exec.Command("go", "mod", "tidy")

			tidy.Dir = "tmp"
			if out, err := tidy.CombinedOutput(); err != nil {
				panic(fmt.Errorf("go mod tidy: %s\n%w", out, err))
			}
		}
	}

	// fmt.Printf("--- %s ---\n", filepath.Join(outDir, outName))

	// if err := t.Print(os.Stdout, gofile); err != nil {
	// 	panic(fmt.Errorf("printing output: %w", err))
	// }

	// fmt.Println()
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
	fileID uint16
}

// runProject compiles the entry package and all its imported packages.
func runProject(ctx context.Context, projectRoot string, entryFiles []string) error {
	// Step 1: Lex and validate the entry package.
	entryLexed, entryPkgName, err := lexAndValidate(ctx, entryFiles)
	if err != nil {
		return err
	}

	// The Go module name for the transpiled project matches the entry package name.
	goModuleName := entryPkgName

	// Step 2: FindGlobals on the entry package (discovers globals + import paths).
	entrySymbols := parser.NewSymbolTable()

	entryParsers := findGlobals(ctx, entryLexed, entrySymbols)
	if entryParsers == nil {
		return fmt.Errorf("failed to find globals")
	}

	// A package that declares a main proc must be named "main".
	if _, hasMain := entrySymbols.Resolve("main"); hasMain && entryPkgName != "main" {
		return fmt.Errorf("package %q declares a main proc but is not named \"main\"", entryPkgName)
	}

	// Step 3: Process imported packages.
	importedPkgs := make(map[string]*compiledPackage) // key: import path

	for _, imp := range entrySymbols.CogImports() {
		pkg := compileImportedPackage(ctx, projectRoot, imp.Path)
		if pkg == nil {
			return fmt.Errorf("failed to compile imported package %q", imp.Path)
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
				return err
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

	// Step 6: Discover and compile any .cogs script files in the project root.
	scriptFiles := discoverScripts(projectRoot)
	for _, sf := range scriptFiles {
		runScript(ctx, projectRoot, sf, goModuleName)
	}

	return nil
}

// discoverScripts finds all .cogs files in the given directory.
func discoverScripts(dir string) []string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}

	var scripts []string

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".cogs") {
			continue
		}

		scripts = append(scripts, filepath.Join(dir, entry.Name()))
	}

	sort.Strings(scripts)

	return scripts
}

// lexAndValidate lexes all files and validates they declare the same package.
func lexAndValidate(ctx context.Context, files []string) ([]lexedFile, string, error) {
	lexed := make([]lexedFile, 0, len(files))

	for i, path := range files {
		toks, err := lexFile(ctx, path, uint16(i))
		if err != nil {
			return nil, "", err
		}

		lexed = append(lexed, lexedFile{path: path, tokens: toks, fileID: uint16(i)})
	}

	dirName := filepath.Base(filepath.Dir(files[0]))

	var pkgName string

	for _, lf := range lexed {
		if len(lf.tokens) < 2 || lf.tokens[0].Type != tokens.Package {
			return nil, "", fmt.Errorf("%s: missing package declaration", lf.path)
		}

		name := lf.tokens[1].Literal

		if pkgName == "" {
			pkgName = name

			if pkgName != "main" && dirName != "." && pkgName != dirName {
				return nil, "", fmt.Errorf("%s: package %q does not match directory name %q", lf.path, pkgName, dirName)
			}
		} else if name != pkgName {
			return nil, "", fmt.Errorf("%s: declares package %q, but other files use %q", lf.path, name, pkgName)
		}
	}

	return lexed, pkgName, nil
}

// findGlobals runs FindGlobals on all files with a shared symbol table.
func findGlobals(ctx context.Context, lexed []lexedFile, symbols *parser.SymbolTable) []*parser.Parser {
	parsers := make([]*parser.Parser, len(lexed))

	for i, lf := range lexed {
		p, err := parser.NewParserWithSymbols(lf.tokens, symbols, debug, lf.path)
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

	lexed, pkgName, err := lexAndValidate(ctx, files)
	if err != nil {
		fmt.Println(err.Error())
		return nil
	}

	symbols := parser.NewSymbolTable()

	parsers := findGlobals(ctx, lexed, symbols)
	if parsers == nil {
		return nil
	}

	// Imported packages must not declare a main proc.
	if sym, hasMain := symbols.Resolve("main"); hasMain {
		ln, col := sym.Identifier.Token.Ln, sym.Identifier.Token.Col
		fmt.Printf("%s:%d:%d: imported package %q must not declare a main proc\n",
			files[0], ln, col, pkgName)

		return nil
	}

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
		gomod := fmt.Sprintf("module %s\n\ngo 1.26.2\n", goModuleName)
		if replaceLocalCog {
			gomod += "\nreplace github.com/samborkent/cog => ./..\n"
		}

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
	t := transpiler.NewTranspilerWithModule(goModuleName, pkg.astFiles)

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
			// fmt.Printf("--- %s ---\n", filepath.Join(outDir, outName))

			// if err := t.Print(os.Stdout, gofiles[i]); err != nil {
			// 	panic(fmt.Errorf("printing output: %w", err))
			// }

			// fmt.Println()
		}
	}
}
