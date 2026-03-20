package parser_test

import (
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
		if d.Assignment.Expression == nil {
			t.Fatal("expected expression")
		}
		if _, ok := d.Assignment.Expression.(*ast.Infix); !ok {
			t.Errorf("expected Infix expression, got %T", d.Assignment.Expression)
		}
	})

	t.Run("prefix", func(t *testing.T) {
		t.Parallel()
		f := parse(t, `package p
x := -1
main : proc() = {}`)
		d := stmtAs[*ast.Declaration](t, f, 0)
		if _, ok := d.Assignment.Expression.(*ast.Prefix); !ok {
			t.Errorf("expected Prefix expression, got %T", d.Assignment.Expression)
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
		if len(f.Statements) == 0 {
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
		if len(f.Statements) == 0 {
			t.Fatal("expected statements")
		}
	})
}
