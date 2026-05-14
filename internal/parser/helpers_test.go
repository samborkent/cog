package parser_test

import (
	"context"
	"strings"
	"testing"

	"github.com/samborkent/cog/internal/ast"
	"github.com/samborkent/cog/internal/lexer"
)

func parse(t *testing.T, src string) *ast.AST {
	t.Helper()

	l := lexer.New(strings.NewReader(src), uint32(len(src)), false)

	p, err := NewTestParser(t, l)
	if err != nil {
		t.Fatalf("parser init error: %v", err)
	}

	f, err := p.Parse(t.Context(), "test.cog")
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	return f
}

func parseShouldError(t *testing.T, src string) {
	t.Helper()

	ctx, cancel := context.WithTimeout(t.Context(), 3e9)
	defer cancel()

	l := lexer.New(strings.NewReader(src), uint32(len(src)), false)

	p, err := NewTestParser(t, l)
	if err != nil {
		t.Fatalf("parser creation error: %v", err)
	}

	_, err = p.Parse(ctx, "test.cog")
	if err == nil {
		t.Fatal("expected parse error, got nil")
	}
}

func stmtAs[T ast.Node](t *testing.T, f *ast.AST, i int) T {
	t.Helper()

	file := f.Node(1).(*ast.File)

	if i >= len(file.Statements) {
		t.Fatalf("expected at least %d statements, got %d", i+1, len(file.Statements))
	}

	s, ok := f.Node(file.Statements[i]).(T)
	if !ok {
		t.Fatalf("statement %d: expected %T, got %T", i, *new(T), f.Node(file.Statements[i]))
	}

	return s
}
