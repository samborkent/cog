package parser_test

import (
	"strings"
	"testing"

	"github.com/samborkent/cog/internal/ast"
)

func TestExpression(t *testing.T) {
	t.Parallel()

	t.Run("infix", func(t *testing.T) {
		t.Parallel()
		f := parse(t, `package p
x := 1 + 2
main : proc() = {}`)

		d := stmtAs[*ast.Declaration](t, f, 0)

		if d.Assignment.Expr == ast.ZeroExprIndex {
			t.Fatal("expected expression")
		}

		if _, ok := f.Expr(d.Assignment.Expr).(*ast.Infix); !ok {
			t.Errorf("expected Infix expression, got %T", d.Assignment.Expr)
		}
	})

	t.Run("prefix", func(t *testing.T) {
		t.Parallel()
		f := parse(t, `package p
x := -1
main : proc() = {}`)

		d := stmtAs[*ast.Declaration](t, f, 0)

		if _, ok := f.Expr(d.Assignment.Expr).(*ast.Prefix); !ok {
			t.Errorf("expected Prefix expression, got %T", d.Assignment.Expr)
		}
	})

	t.Run("comparison", func(t *testing.T) {
		t.Parallel()

		f := parse(t, `package p
main : proc() = {
	x := 1
	if x == 1 {
		@print("yes")
	}
}`)
		if f.LenNodes() == 0 {
			t.Fatal("expected statements")
		}
	})

	t.Run("boolean", func(t *testing.T) {
		t.Parallel()

		f := parse(t, `package p
main : proc() = {
	if true && false {
		@print("both")
	}
}`)
		if f.LenNodes() == 0 {
			t.Fatal("expected statements")
		}
	})
}

func TestStructLiteralWithTypeAlias(t *testing.T) {
	t.Parallel()

	// Test case for parsing struct literals with type aliases like Struct{...}
	src := `package main

Struct ~ struct {
	Field : int64
}

broken := Struct{
	Field = 42,
}`

	f := parse(t, src)

	// Check that we have the expected statements (type alias + variable declaration)
	if f.LenNodes() != 2 {
		t.Fatalf("expected 2 statements, got %d", f.LenNodes())
	}

	// The second statement should be the variable declaration with struct literal
	declaration := stmtAs[*ast.Declaration](t, f, 1)

	// Check that the value is a struct literal
	structLiteral, ok := f.Expr(declaration.Assignment.Expr).(*ast.StructLiteral)
	if !ok {
		t.Fatalf("expected struct literal, got %T", f.Expr(declaration.Assignment.Expr))
	}

	if !strings.Contains(structLiteral.StructType.String(), "Struct") {
		t.Errorf("expected struct type with Struct, got %s", structLiteral.StructType.String())
	}

	if !strings.Contains(structLiteral.StructType.Underlying().String(), "Field : int64") {
		t.Errorf("expected struct type with Field : int64, got %s", structLiteral.StructType.Underlying().String())
	}

	// Check that the struct literal has the expected field
	if len(structLiteral.Values) != 1 {
		t.Fatalf("expected 1 field value, got %d", len(structLiteral.Values))
	}

	if structLiteral.Values[0].Name != "Field" {
		t.Errorf("expected field name 'Field', got %s", structLiteral.Values[0].Name)
	}

	// Check that the field value is correct (it includes type annotation)
	if !strings.Contains(f.Expr(structLiteral.Values[0].Value).String(), "42") {
		t.Errorf("expected field value containing '42', got %s", f.Expr(structLiteral.Values[0].Value).String())
	}
}
