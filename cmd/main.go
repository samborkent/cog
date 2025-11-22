package main

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"strings"

	"github.com/samborkent/cog/internal/ast"
	"github.com/samborkent/cog/internal/lexer"
	"github.com/samborkent/cog/internal/parser"
	"github.com/samborkent/cog/internal/transpiler"
)

var (
	fileName string
	debug    bool
	write    bool
)

func main() {
	flag.StringVar(&fileName, "file", "", "Name of file to execute.")
	flag.BoolVar(&debug, "debug", false, "Enable debug parser mode.")
	flag.BoolVar(&write, "write", false, "Write to file.")
	flag.Parse()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, os.Kill)
	defer stop()

	if fileName == "" {
		runPromt(ctx)
	} else {
		runFile(ctx, fileName)
	}
}

func runPromt(ctx context.Context) {
	scanner := bufio.NewScanner(os.Stdin)

	lines := make(chan []byte, 1)

	go func() {
		for scanner.Scan() {
			if ctx.Err() != nil {
				return
			}

			_, err := fmt.Print("> ")
			if err != nil {
				panic(err)
			}

			line := scanner.Bytes()
			if len(line) == 0 {
				continue
			}

			lines <- line
		}
	}()

	select {
	case <-ctx.Done():
		close(lines)
		break
	case line := <-lines:
		run(ctx, bytes.NewReader(line))
	}

	if err := scanner.Err(); err != nil {
		panic(err)
	}
}

func runFile(ctx context.Context, fileName string) {
	fileName = filepath.Clean(fileName)

	if !strings.HasSuffix(fileName, ".cog") {
		panic("invalid file extension, must be .cog")
	}

	file, err := os.Open(fileName)
	if err != nil {
		panic(err)
	}

	run(ctx, file)
}

func run(ctx context.Context, r io.Reader) {
	l := lexer.NewLexer(r)

	tokens, err := l.Parse(ctx)
	if err != nil {
		fmt.Println(err.Error())
		return
	}

	p, err := parser.NewParser(tokens, debug)
	if err != nil {
		fmt.Println(err.Error())
		return
	}

	done := make(chan struct{})

	var f *ast.File

	go func() {
		defer func() {
			if r := recover(); r != nil {
				if len(p.Errs) > 0 {
					fmt.Println("parser errors:")
					fmt.Println(errors.Join(p.Errs...).Error())
				}

				panic(r)
			}
		}()

		f, err = p.Parse(ctx)
		if err != nil {
			fmt.Println(err.Error())
		}

		close(done)
	}()

	<-done

	if !write {
		fmt.Printf("\nparsed nodes:\n\n")

		ln, col := f.Package.Pos()
		fmt.Printf("%d - ln %d, col %d: %s\n", 0, ln, col, f.Package)

		for i, n := range f.Statements {
			ln, col := n.Pos()
			fmt.Printf("%d - ln %d, col %d: %s\n", i+1, ln, col, n)
		}

		if err != nil {
			return
		}
	}

	t := transpiler.NewTranspiler(f, fileName)

	src, err := t.Transpile()
	if err != nil {
		fmt.Println(err.Error())
		return
	}

	// TODO: implement multi file projects

	out := os.Stdout
	if write {
		if err := os.MkdirAll("tmp", 0o700); err != nil {
			panic(fmt.Errorf("creating temp dir: %w", err))
		}

		outFile, err := os.Create("tmp/" + strings.TrimSuffix(fileName, ".cog") + ".go")
		if err != nil {
			panic(fmt.Errorf("creating output file: %w", err))
		}
		defer func() { _ = outFile.Close() }()

		out = outFile
	} else {
		fmt.Printf("\ntranspiled nodes:\n\n")
	}

	if _, err := out.Write([]byte(src)); err != nil {
		panic(fmt.Errorf("writing output: %w", err))
	}
}
