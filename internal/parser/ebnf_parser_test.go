package parser_test

import (
	"testing"

	"github.com/samborkent/cog/internal/ast"
	"github.com/samborkent/cog/internal/types"
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

	file := f.Node(1).(*ast.File)

	if len(file.Statements) != 2 {
		t.Fatalf("expected 2 statements, got %d", len(file.Statements))
	}

	declaration := stmtAs[*ast.Declaration](t, f, 1)

	// Check that the value is a struct literal
	structLiteral, ok := f.Expr(declaration.Assignment.Expr).(*ast.StructLiteral)
	if !ok {
		t.Fatalf("expected struct literal, got %T", f.Expr(declaration.Assignment.Expr))
	}

	aliasType, ok := structLiteral.StructType.(*types.Alias)
	if !ok {
		t.Fatalf("expected alias type for struct literal, got %T", structLiteral.StructType)
	}

	if aliasType.Name != "Struct" {
		t.Errorf("expected struct type with Struct, got %s", aliasType.Name)
	}

	structType, ok := aliasType.Derived.(*types.Struct)
	if !ok {
		t.Fatalf("expected struct type for alias, got %T", aliasType.Derived)
	}

	if structType.Fields[0].Name != "Field" {
		t.Errorf("expected field name 'Field', got %s", structType.Fields[0].Name)
	}

	if structType.Fields[0].Type.Kind() != types.Int64 {
		t.Errorf("expected field type int64, got %s", structType.Fields[0].Type.String())
	}

	if len(structLiteral.Values) != 1 {
		t.Fatalf("expected 1 field value, got %d", len(structLiteral.Values))
	}

	if structLiteral.Values[0].Name != "Field" {
		t.Errorf("expected field name 'Field', got %s", structLiteral.Values[0].Name)
	}

	intLiteral := f.Expr(structLiteral.Values[0].Value).(*ast.Int64Literal)

	if intLiteral.Value != 42 {
		t.Errorf("expected field value 42, got %d", intLiteral.Value)
	}
}
