package main

import (
	"context"
	"flag"
	"fmt"
	"os"
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
	runPackage(ctx, files)
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

// runPackage lexes, parses, and transpiles all .cog files as a single package.
func runPackage(ctx context.Context, files []string) {
	// Step 2: Lex each file independently.
	type lexedFile struct {
		path   string
		tokens []tokens.Token
	}

	lexed := make([]lexedFile, 0, len(files))

	for _, path := range files {
		toks, err := lexFile(ctx, path)
		if err != nil {
			fmt.Println(err.Error())
			return
		}

		lexed = append(lexed, lexedFile{path: path, tokens: toks})
	}

	// Step 3: Validate all files declare the same package name
	// and that it matches the directory name.
	dirName := filepath.Base(filepath.Dir(files[0]))
	var pkgName string

	for _, lf := range lexed {
		if len(lf.tokens) < 2 || lf.tokens[0].Type != tokens.Package {
			fmt.Printf("%s: missing package declaration\n", lf.path)
			return
		}

		name := lf.tokens[1].Literal

		if pkgName == "" {
			pkgName = name

			if pkgName != "main" && pkgName != dirName {
				fmt.Printf("%s: package %q does not match directory name %q\n", lf.path, pkgName, dirName)
				return
			}
		} else if name != pkgName {
			fmt.Printf("%s: declares package %q, but other files use %q\n", lf.path, name, pkgName)
			return
		}
	}

	// Step 4: Two-phase parse with shared symbol table.
	symbols := parser.NewSymbolTable()

	// Phase A: Pre-register globals from all files.
	parsers := make([]*parser.Parser, len(lexed))

	for i, lf := range lexed {
		p, err := parser.NewParserWithSymbols(lf.tokens, symbols, debug)
		if err != nil {
			fmt.Println(err.Error())
			return
		}

		p.FindGlobals(ctx)
		parsers[i] = p
	}

	// Phase B: Full parse each file using the shared symbol table.
	astFiles := make([]*ast.File, len(lexed))

	for i, lf := range lexed {
		f, err := parsers[i].ParseOnly(ctx, lf.path)
		if err != nil {
			fmt.Println(err.Error())
		}

		astFiles[i] = f

		if !write {
			fmt.Printf("--- %s ---\n%s\n\n", lf.path, f)

			if err != nil {
				return
			}
		}
	}

	// Step 5: Transpile — one Go file per cog file.
	t := transpiler.NewTranspiler(astFiles...)

	gofiles, err := t.TranspileFiles()
	if err != nil {
		fmt.Println(err.Error())
		return
	}

	if write {
		if err := os.MkdirAll("tmp", 0o700); err != nil {
			panic(fmt.Errorf("creating temp dir: %w", err))
		}
	}

	for i, lf := range lexed {
		outName := filepath.Base(lf.path)
		outName = strings.TrimSuffix(outName, ".cog") + ".go"

		if write {
			outFile, err := os.Create(filepath.Join("tmp", outName))
			if err != nil {
				panic(fmt.Errorf("creating output file: %w", err))
			}

			if err := t.Print(outFile, gofiles[i]); err != nil {
				_ = outFile.Close()
				panic(fmt.Errorf("printing output: %w", err))
			}

			_ = outFile.Close()
		} else {
			fmt.Printf("--- %s ---\n", outName)

			if err := t.Print(os.Stdout, gofiles[i]); err != nil {
				panic(fmt.Errorf("printing output: %w", err))
			}

			fmt.Println()
		}
	}
}
