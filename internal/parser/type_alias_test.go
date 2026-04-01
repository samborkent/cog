package parser_test

import (
	"testing"

	"github.com/samborkent/cog/internal/ast"
	"github.com/samborkent/cog/internal/types"
)

func TestParseTypeAlias(t *testing.T) {
	t.Parallel()

	t.Run("simple", func(t *testing.T) {
		t.Parallel()
		f := parse(t, `package p
MyInt ~ int32
main : proc() = {}`)
		ta := stmtAs[*ast.Type](t, f, 0)
		if ta.Identifier.Name != "MyInt" {
			t.Errorf("expected name 'MyInt', got %q", ta.Identifier.Name)
		}
		if ta.Alias.Kind() != types.Int32 {
			t.Errorf("expected Int32, got %s", ta.Alias.Kind())
		}
	})

	t.Run("tuple", func(t *testing.T) {
		t.Parallel()
		f := parse(t, `package p
Pair ~ int32 & utf8
main : proc() = {}`)
		ta := stmtAs[*ast.Type](t, f, 0)
		if ta.Alias.Kind() != types.TupleKind {
			t.Errorf("expected TupleKind, got %s", ta.Alias.Kind())
		}
	})

	t.Run("union", func(t *testing.T) {
		t.Parallel()
		f := parse(t, `package p
Either ~ int32 | utf8
main : proc() = {}`)
		ta := stmtAs[*ast.Type](t, f, 0)
		if ta.Alias.Kind() != types.UnionKind {
			t.Errorf("expected UnionKind, got %s", ta.Alias.Kind())
		}
	})

	t.Run("option", func(t *testing.T) {
		t.Parallel()
		f := parse(t, `package p
MaybeInt ~ int32?
main : proc() = {}`)
		ta := stmtAs[*ast.Type](t, f, 0)
		if ta.Alias.Kind() != types.OptionKind {
			t.Errorf("expected OptionKind, got %s", ta.Alias.Kind())
		}
	})

	t.Run("slice", func(t *testing.T) {
		t.Parallel()
		f := parse(t, `package p
Ints ~ []int32
main : proc() = {}`)
		ta := stmtAs[*ast.Type](t, f, 0)
		if ta.Alias.Kind() != types.SliceKind {
			t.Errorf("expected SliceKind, got %s", ta.Alias.Kind())
		}
	})

	t.Run("array", func(t *testing.T) {
		t.Parallel()
		f := parse(t, `package p
Triple ~ [3]int32
main : proc() = {}`)
		ta := stmtAs[*ast.Type](t, f, 0)
		if ta.Alias.Kind() != types.ArrayKind {
			t.Errorf("expected ArrayKind, got %s", ta.Alias.Kind())
		}
	})

	t.Run("map", func(t *testing.T) {
		t.Parallel()
		f := parse(t, `package p
Dict ~ map<utf8, int32>
main : proc() = {}`)
		ta := stmtAs[*ast.Type](t, f, 0)
		if ta.Alias.Kind() != types.MapKind {
			t.Errorf("expected MapKind, got %s", ta.Alias.Kind())
		}
	})

	t.Run("set", func(t *testing.T) {
		t.Parallel()
		f := parse(t, `package p
UniqueInts ~ set<int32>
main : proc() = {}`)
		ta := stmtAs[*ast.Type](t, f, 0)
		if ta.Alias.Kind() != types.SetKind {
			t.Errorf("expected SetKind, got %s", ta.Alias.Kind())
		}
	})

	t.Run("struct", func(t *testing.T) {
		t.Parallel()
		f := parse(t, `package p
Point ~ struct {
	x : int32
	y : int32
}
main : proc() = {}`)
		ta := stmtAs[*ast.Type](t, f, 0)
		if ta.Alias.Kind() != types.StructKind {
			t.Errorf("expected StructKind, got %s", ta.Alias.Kind())
		}
	})

	t.Run("enum", func(t *testing.T) {
		t.Parallel()
		f := parse(t, `package p
Color ~ enum<utf8> {
	Red := "red",
	Green := "green",
	Blue := "blue",
}
main : proc() = {}`)
		ta := stmtAs[*ast.Type](t, f, 0)
		if ta.Alias.Kind() != types.EnumKind {
			t.Errorf("expected EnumKind, got %s", ta.Alias.Kind())
		}
	})

	t.Run("typed_error", func(t *testing.T) {
		t.Parallel()
		f := parse(t, `package p
MyError ~ error<utf8> {
	NotFound := "not found",
	Timeout := "timeout",
}
main : proc() = {}`)
		ta := stmtAs[*ast.Type](t, f, 0)
		if ta.Alias.Kind() != types.ErrorKind {
			t.Errorf("expected ErrorKind, got %s", ta.Alias.Kind())
		}
	})

	t.Run("typeless_error", func(t *testing.T) {
		t.Parallel()
		f := parse(t, `package p
MyError ~ error {
	NotFound,
	Timeout,
}
main : proc() = {}`)
		ta := stmtAs[*ast.Type](t, f, 0)
		if ta.Alias.Kind() != types.ErrorKind {
			t.Errorf("expected ErrorKind, got %s", ta.Alias.Kind())
		}
	})

	t.Run("ascii_error", func(t *testing.T) {
		t.Parallel()
		f := parse(t, `package p
MyError ~ error<ascii> {
	NotFound := "not found",
}
main : proc() = {}`)
		ta := stmtAs[*ast.Type](t, f, 0)
		if ta.Alias.Kind() != types.ErrorKind {
			t.Errorf("expected ErrorKind, got %s", ta.Alias.Kind())
		}
	})

	t.Run("error_invalid_type_param", func(t *testing.T) {
		t.Parallel()
		parseShouldError(t, `package p
MyError ~ error<int32> {
	Fail := 1,
}
main : proc() = {}`)
	})

	t.Run("result_requires_error_type", func(t *testing.T) {
		t.Parallel()
		parseShouldError(t, `package p
NotAnError ~ int32
main : proc() = {
	var r : int64 ! NotAnError
}`)
	})

	t.Run("result_value_cannot_be_error", func(t *testing.T) {
		t.Parallel()
		parseShouldError(t, `package p
MyErr ~ error { Fail }
OtherErr ~ error { Bad }
main : proc() = {
	var r : MyErr ! OtherErr
}`)
	})
}
