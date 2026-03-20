package parser_test

import (
	"testing"

	"github.com/samborkent/cog/internal/ast"
	"github.com/samborkent/cog/internal/types"
)

func TestParseLiteral(t *testing.T) {
	t.Parallel()

	t.Run("struct", func(t *testing.T) {
		t.Parallel()
		f := parse(t, `package p
Point ~ struct {
	x : int32
	y : int32
}
main : proc() = {
	p : Point = {
		x = 1,
		y = 2,
	}
	@print(p)
}`)
		if len(f.Statements) == 0 {
			t.Fatal("expected statements")
		}
	})

	t.Run("array", func(t *testing.T) {
		t.Parallel()
		f := parse(t, `package p
arr : [3]int64 = {1, 2, 3}
main : proc() = {}`)
		d := stmtAs[*ast.Declaration](t, f, 0)
		if d.Assignment.Identifier.ValueType.Kind() != types.ArrayKind {
			t.Errorf("expected ArrayKind, got %s", d.Assignment.Identifier.ValueType.Kind())
		}
	})

	t.Run("slice", func(t *testing.T) {
		t.Parallel()
		f := parse(t, `package p
xs : []utf8 = {"foo", "bar"}
main : proc() = {}`)
		d := stmtAs[*ast.Declaration](t, f, 0)
		if d.Assignment.Identifier.ValueType.Kind() != types.SliceKind {
			t.Errorf("expected SliceKind, got %s", d.Assignment.Identifier.ValueType.Kind())
		}
	})

	t.Run("map", func(t *testing.T) {
		t.Parallel()
		f := parse(t, `package p
M ~ map<utf8, int64>
m : M = {"a": 1, "b": 2}
main : proc() = {}`)
		if len(f.Statements) < 2 {
			t.Fatal("expected at least 2 statements")
		}
	})
}
