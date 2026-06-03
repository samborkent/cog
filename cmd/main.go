package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"slices"
	"sort"
	"strings"
	"sync"
	"syscall"

	"golang.org/x/sync/errgroup"

	"github.com/KimMachineGun/automemlimit/memlimit"
	"go.uber.org/automaxprocs/maxprocs"

	"github.com/samborkent/cog/internal/ast"
	"github.com/samborkent/cog/internal/lexer"
	"github.com/samborkent/cog/internal/parser"
	"github.com/samborkent/cog/internal/tokens"
	"github.com/samborkent/cog/internal/transpiler"
)

var (
	fileName        string
	debugMode       bool
	write           bool
	replaceLocalCog bool
)

// minParallelFiles is the minimum number of files needed to justify
// errgroup overhead. Below this threshold, sequential execution is faster.
const minParallelFiles = 4

func main() {
	flag.StringVar(&fileName, "file", "", "Name of .cog/.cogs file or directory containing .cog files.")
	flag.BoolVar(&debugMode, "debug", false, "Enable debug parser mode.")
	flag.BoolVar(&write, "write", false, "Write to file.")
	flag.BoolVar(&replaceLocalCog, "replace-local-cog", false, "Add replace directive for local cog module in generated go.mod.")
	flag.Parse()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	if fileName == "" {
		fmt.Println("missing file or directory name")
		return
	}

	// Set GOMAXPROCS.
	_, _ = maxprocs.Set()

	// Set GOMEMLIMIT based on 90% of available memory.
	_, _ = memlimit.SetGoMemLimitWithOpts(
		memlimit.WithProvider(memlimit.ApplyFallback(
			memlimit.FromCgroup,
			memlimit.FromSystem,
		)),
	)

	// Disable GC to improve performance of large projects.
	debug.SetGCPercent(-1)

	files, err := discoverFiles(fileName)
	if err != nil {
		fmt.Println(err.Error())
		return
	}

	// Script mode: single .cogs file.
	if strings.HasSuffix(files[0], ".cogs") {
		projectRoot := filepath.Dir(files[0])
		if err := runScript(ctx, projectRoot, files[0], ""); err != nil {
			fmt.Fprintln(os.Stderr, err.Error())
		}

		return
	}

	// Import paths are resolved relative to this root.
	projectRoot := filepath.Dir(files[0])

	if err := runProject(ctx, projectRoot, files); err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
	}
}

// discoverFiles resolves the input flag to a sorted list of .cog file paths.
// If a single .cog file is given, only that file is returned.
// If a directory is given, all .cog files in that directory are returned.
func discoverFiles(input string) ([]string, error) {
	input = filepath.Clean(input)

	info, err := os.Stat(input)
	if err != nil {
		return nil, fmt.Errorf("cannot access %q: %w", input, err)
	}

	// Single file: return just that file.
	if !info.IsDir() {
		if !strings.HasSuffix(input, ".cog") && !strings.HasSuffix(input, ".cogs") {
			return nil, fmt.Errorf("invalid file extension, must be .cog or .cogs")
		}

		return []string{input}, nil
	}

	// Directory: scan for all .cog files.
	entries, err := os.ReadDir(input)
	if err != nil {
		return nil, fmt.Errorf("reading directory %q: %w", input, err)
	}

	files := make([]string, 0, len(entries))

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".cog") {
			continue
		}

		files = append(files, filepath.Join(input, entry.Name()))
	}

	if len(files) == 0 {
		return nil, fmt.Errorf("no .cog files found in %q", input)
	}

	slices.Sort(files)

	return files, nil
}

// lexFile lexes a single .cog file and returns its token stream.
func lexFile(path string) (*lexer.Lexer, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("opening %q: %w", path, err)
	}

	fileInfo, err := file.Stat()
	if err != nil {
		return nil, fmt.Errorf("stat %q: %w", path, err)
	}

	return lexer.New(file, uint32(fileInfo.Size()), debugMode), nil
}

// runScript compiles a single .cogs script file.
// Script files have no package declaration; the transpiled output is placed
// in cmd/{scriptName}/ with package main and a func main() wrapping the body.
// If goModuleName is empty, the script name is used and go.mod is written.
func runScript(ctx context.Context, projectRoot string, scriptPath string, goModuleName string) error {
	toks, err := lexFile(scriptPath)
	if err != nil {
		return err
	}

	symbols := parser.NewSymbolTableAuto(1, minParallelFiles)

	p, err := parser.NewScriptParserWithSymbols(toks, symbols, nil, scriptPath)
	if err != nil {
		return err
	}

	f, err := p.ParseGlobals(ctx, scriptPath)
	if err != nil {
		return err
	}

	// Process imported packages.
	importedPkgs := make(map[string]*compiledPackage)
	var importLock sync.Mutex

	group, errCtx := errgroup.WithContext(ctx)
	group.SetLimit(runtime.GOMAXPROCS(-1))

	for _, imp := range symbols.CogImports().All() {
		group.Go(func() error {
			pkg, err := compileImportedPackage(errCtx, projectRoot, imp.Path)
			if err != nil {
				return fmt.Errorf("failed to compile imported package %q: %w", imp.Path, err)
			}

			importLock.Lock()
			importedPkgs[imp.Path] = pkg
			importLock.Unlock()

			populateImportExports(imp, pkg.symbols)

			return nil
		})
	}

	if err := group.Wait(); err != nil {
		return err
	}

	if err := p.ParseBodies(ctx); err != nil {
		return err
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
			return fmt.Errorf("creating temp dir: %w", err)
		}
	}

	for _, pkg := range importedPkgs {
		transpileAndOutput(ctx, goModuleName, pkg)
	}

	// Transpile the script file.
	t := transpiler.NewTranspilerWithModule(goModuleName, ast.MergeASTs(f))

	gofile, err := t.TranspileScript()
	if err != nil {
		return err
	}

	outDir := filepath.Join("tmp", "cmd", scriptName)
	outName := "main.go"

	if write {
		if err := os.MkdirAll(outDir, 0o700); err != nil {
			return fmt.Errorf("creating output dir: %w", err)
		}

		outFile, err := os.Create(filepath.Join(outDir, outName))
		if err != nil {
			return fmt.Errorf("creating output file: %w", err)
		}

		if err := t.Print(outFile, gofile); err != nil {
			_ = outFile.Close()

			return fmt.Errorf("printing output: %w", err)
		}

		_ = outFile.Close()

		// Only write go.mod when running as a standalone script (not part of a project).
		if standalone {
			gomod := fmt.Sprintf("module %s\n\ngo 1.26.3\n", goModuleName)
			if replaceLocalCog {
				gomod += "\nreplace github.com/samborkent/cog => ./..\n"
			}

			if err := os.WriteFile(filepath.Join("tmp", "go.mod"), []byte(gomod), 0o600); err != nil {
				return fmt.Errorf("writing go.mod: %w", err)
			}

			tidy := exec.Command("go", "mod", "tidy")

			tidy.Dir = "tmp"
			if out, err := tidy.CombinedOutput(); err != nil {
				return fmt.Errorf("go mod tidy: %s\n%w", out, err)
			}
		}
	}

	return nil
}

// compiledPackage holds the output of compiling a single cog package.
type compiledPackage struct {
	importPath string        // relative import path (empty for the entry package)
	pkgName    string        // Go package name
	files      []lexedFile   // original file paths
	astFiles   ast.MergedAST // parsed ASTs
	symbols    *parser.SymbolTable
}

type lexedFile struct {
	path   string
	lexer  *lexer.Lexer
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

	// Step 2: ParseGlobals on the entry package (full parse with deferred bodies).
	entrySymbols := parser.NewSymbolTableAuto(len(entryLexed), minParallelFiles)

	entryParsers, entryASTs, err := parseGlobals(ctx, entryLexed, entrySymbols)
	if err != nil {
		return err
	}

	// Validate unresolved forward stubs.
	// TODO: this semantics doesn't make sense. A single symbol table should be created at the start,
	// and then this check should be done on that symbol table.
	if err := entryParsers[0].ValidateGlobals(); err != nil {
		return err
	}

	// A package that declares a main proc must be named "main".
	if _, hasMain := entrySymbols.Resolve("main"); hasMain && entryPkgName != "main" {
		return fmt.Errorf("package %q declares a main proc but is not named \"main\"", entryPkgName)
	}

	// Step 3: Process imported packages.
	importedPkgs := make(map[string]*compiledPackage) // key: import path
	var importedLock sync.Mutex

	group, errCtx := errgroup.WithContext(ctx)
	group.SetLimit(runtime.GOMAXPROCS(-1))

	for _, imp := range entrySymbols.CogImports().All() {
		group.Go(func() error {
			pkg, err := compileImportedPackage(errCtx, projectRoot, imp.Path)
			if err != nil {
				return fmt.Errorf("failed to compile imported package %q: %w", imp.Path, err)
			}

			importedLock.Lock()
			importedPkgs[imp.Path] = pkg
			importedLock.Unlock()

			// Populate the entry package's import exports from the imported package.
			populateImportExports(imp, pkg.symbols)

			return nil
		})
	}

	if err := group.Wait(); err != nil {
		return err
	}

	// Step 4: ParseBodies (now with import exports available).
	if len(entryLexed) < minParallelFiles {
		for i, lf := range entryLexed {
			if err := entryParsers[i].ParseBodies(ctx); err != nil {
				return err
			}

			if !write {
				fmt.Printf("--- %s ---\n%s\n\n", lf.path, entryASTs[i].Node(1))
			}
		}
	} else {
		group, _ = errgroup.WithContext(ctx)
		group.SetLimit(runtime.GOMAXPROCS(-1))

		for i, lf := range entryLexed {
			group.Go(func() error {
				if err := entryParsers[i].ParseBodies(ctx); err != nil {
					return err
				}

				if !write {
					fmt.Printf("--- %s ---\n%s\n\n", lf.path, entryASTs[i].Node(1))
				}

				return nil
			})
		}

		if err := group.Wait(); err != nil {
			return err
		}
	}

	// Step 5: Transpile and output.
	entryPkg := &compiledPackage{
		pkgName:  entryPkgName,
		files:    entryLexed,
		astFiles: ast.MergeASTs(entryASTs...),
		symbols:  entrySymbols,
	}

	outputProject(ctx, goModuleName, entryPkg, importedPkgs)

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

	if len(files) < minParallelFiles {
		for i, path := range files {
			lex, err := lexFile(path)
			if err != nil {
				return nil, "", fmt.Errorf("lexing file %q: %w", path, err)
			}

			lexed = append(lexed, lexedFile{path: path, lexer: lex, fileID: uint16(i)})
		}
	} else {
		var lexedLock sync.Mutex

		group, _ := errgroup.WithContext(ctx)
		group.SetLimit(runtime.GOMAXPROCS(-1))

		for i, path := range files {
			group.Go(func() error {
				lex, err := lexFile(path)
				if err != nil {
					return fmt.Errorf("lexing file %q: %w", path, err)
				}

				lexedLock.Lock()
				lexed = append(lexed, lexedFile{path: path, lexer: lex, fileID: uint16(i)})
				lexedLock.Unlock()

				return nil
			})
		}

		if err := group.Wait(); err != nil {
			return nil, "", err
		}
	}

	dirName := filepath.Base(filepath.Dir(files[0]))

	var pkgName string

	for _, lf := range lexed {
		if lf.lexer.Len < 2 || lf.lexer.Peek(0).Type != tokens.Package {
			return nil, "", fmt.Errorf("%s: missing package declaration", lf.path)
		}

		name := lf.lexer.Peek(1).Literal

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

// parseGlobals runs ParseGlobals on all files with a shared symbol table.
// Returns the parsers (for subsequent ParseBodies) and the ASTs.
func parseGlobals(ctx context.Context, lexed []lexedFile, symbols *parser.SymbolTable) ([]*parser.Parser, []*ast.AST, error) {
	parsers := make([]*parser.Parser, len(lexed))
	asts := make([]*ast.AST, len(lexed))

	if len(lexed) < minParallelFiles {
		for i, lf := range lexed {
			p, err := parser.NewParserWithSymbols(lf.lexer, symbols, lf.path, uint16(i), nil)
			if err != nil {
				return nil, nil, fmt.Errorf("creating parser for %q: %w", lf.path, err)
			}

			f, err := p.ParseGlobals(ctx, lf.path)
			if err != nil {
				return nil, nil, fmt.Errorf("parsing globals for %q: %w", lf.path, err)
			}

			parsers[i] = p
			asts[i] = f
		}
	} else {
		group, errCtx := errgroup.WithContext(ctx)
		group.SetLimit(runtime.GOMAXPROCS(-1))

		for i, lf := range lexed {
			group.Go(func() error {
				p, err := parser.NewParserWithSymbols(lf.lexer, symbols, lf.path, uint16(i), nil)
				if err != nil {
					return fmt.Errorf("creating parser for %q: %w", lf.path, err)
				}

				f, err := p.ParseGlobals(errCtx, lf.path)
				if err != nil {
					return fmt.Errorf("parsing globals for %q: %w", lf.path, err)
				}

				parsers[i] = p
				asts[i] = f

				return nil
			})
		}

		if err := group.Wait(); err != nil {
			return nil, nil, err
		}
	}

	return parsers, asts, nil
}

// compileImportedPackage discovers, lexes, parses, and validates an imported package.
func compileImportedPackage(ctx context.Context, projectRoot, importPath string) (*compiledPackage, error) {
	pkgDir := filepath.Join(projectRoot, filepath.FromSlash(importPath))

	files, err := discoverFiles(pkgDir)
	if err != nil {
		return nil, fmt.Errorf("finding .cog files: %w", err)
	}

	lexed, pkgName, err := lexAndValidate(ctx, files)
	if err != nil {
		return nil, fmt.Errorf("lexing files: %w", err)
	}

	symbols := parser.NewSymbolTableAuto(len(lexed), minParallelFiles)

	parsers, astFiles, err := parseGlobals(ctx, lexed, symbols)
	if err != nil {
		return nil, fmt.Errorf("parsing globals: %w", err)
	}

	// Validate unresolved forward stubs.
	if err := parsers[0].ValidateGlobals(); err != nil {
		return nil, err
	}

	// Imported packages must not declare a main proc.
	if sym, hasMain := symbols.Resolve("main"); hasMain {
		ln, col := sym.Identifier.Token.Ln, sym.Identifier.Token.Col

		return nil, fmt.Errorf("%s:%d:%d: imported package %q must not declare a main proc",
			files[0], ln, col, pkgName)
	}

	// ParseBodies to fill in deferred procedure bodies.
	for i, lf := range lexed {
		if err := parsers[i].ParseBodies(ctx); err != nil {
			return nil, fmt.Errorf("parsing bodies in %q: %w", lf.path, err)
		}
	}

	return &compiledPackage{
		importPath: importPath,
		pkgName:    pkgName,
		files:      lexed,
		astFiles:   ast.MergeASTs(astFiles...),
		symbols:    symbols,
	}, nil
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
func outputProject(ctx context.Context, goModuleName string, entry *compiledPackage, imported map[string]*compiledPackage) {
	if write {
		if err := os.MkdirAll("tmp", 0o700); err != nil {
			panic(fmt.Errorf("creating temp dir: %w", err))
		}
	}

	// Transpile and output imported packages first.
	for _, pkg := range imported {
		transpileAndOutput(ctx, goModuleName, pkg)
	}

	// Transpile and output the entry package.
	transpileAndOutput(ctx, goModuleName, entry)

	if write {
		// Write go.mod so `go run .` works from tmp/.
		// Only declare the module and Go version; `go mod tidy` resolves all dependencies.
		gomod := fmt.Sprintf("module %s\n\ngo 1.26.3\n", goModuleName)
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
func transpileAndOutput(ctx context.Context, goModuleName string, pkg *compiledPackage) {
	t := transpiler.NewTranspilerWithModule(goModuleName, pkg.astFiles)

	gofiles, err := t.TranspileFiles(ctx)
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
		}
	}
}
