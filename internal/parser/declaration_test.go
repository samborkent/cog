package parser_test

import (
	"testing"

	"github.com/samborkent/cog/internal/ast"
	"github.com/samborkent/cog/internal/types"
)

func TestParseDeclaration(t *testing.T) {
	t.Parallel()

	t.Run("inferred", func(t *testing.T) {
		t.Parallel()
		f := parse(t, `package p
x := 1
main : proc() = {}`)

		d := stmtAs[*ast.Declaration](t, f, 0)
		if d.Assignment.Identifier.Name != "x" {
			t.Errorf("expected name 'x', got %q", d.Assignment.Identifier.Name)
		}

		if d.Assignment.Expr == ast.ZeroExpr {
			t.Error("expected expression in declaration")
		}
	})

	t.Run("bool", func(t *testing.T) {
		t.Parallel()
		f := parse(t, `package p
b := true
main : proc() = {}`)

		d := stmtAs[*ast.Declaration](t, f, 0)
		if d.Assignment.Identifier.ValueType.Kind() != types.Bool {
			t.Errorf("expected Bool, got %s", d.Assignment.Identifier.ValueType.Kind())
		}
	})

	t.Run("var", func(t *testing.T) {
		t.Parallel()
		f := parse(t, `package p
main : proc() = {
	var x := 1
	@print(x)
}`)
		d := stmtAs[*ast.Declaration](t, f, 0)

		if d.Assignment.Expr.NodeKind != ast.KindProcedureLiteral {
			t.Fatalf("expected ProcedureLiteral, got %T", d.Assignment.Expr)
		}

		proc := d.Assignment.Expr.AsProcedureLiteral()

		if proc.Body.Statements[0].NodeKind != ast.KindDeclaration {
			t.Fatalf("expected Declaration, got %T", proc.Body.Statements[0])
		}

		varDecl := proc.Body.Statements[0].AsDeclaration()

		if varDecl.Assignment.Identifier.Qualifier != ast.QualifierVariable {
			t.Errorf("expected QualifierVariable, got %d", varDecl.Assignment.Identifier.Qualifier)
		}
	})

	t.Run("export", func(t *testing.T) {
		t.Parallel()
		f := parse(t, `package p
export x := 1
main : proc() = {}`)

		d := stmtAs[*ast.Declaration](t, f, 0)
		if !d.Assignment.Identifier.Exported {
			t.Error("expected exported flag to be true")
		}
	})

	t.Run("duplicate_error", func(t *testing.T) {
		t.Parallel()
		parseShouldError(t, `package p
a := 1
a := 2
main : proc() = {}`)
	})

	t.Run("dyn", func(t *testing.T) {
		t.Parallel()
		f := parse(t, `package p
dyn val : utf8 = "default"
main : proc() = {}`)

		d := stmtAs[*ast.Declaration](t, f, 0)
		if d.Assignment.Identifier.Qualifier != ast.QualifierDynamic {
			t.Errorf("expected QualifierDynamic, got %d", d.Assignment.Identifier.Qualifier)
		}

		if d.Assignment.Identifier.ValueType.Kind() != types.UTF8 {
			t.Errorf("expected UTF8, got %s", d.Assignment.Identifier.ValueType.Kind())
		}
	})

	t.Run("dyn_inside_proc_error", func(t *testing.T) {
		t.Parallel()
		parseShouldError(t, `package p
main : proc() = {
	dyn inner : utf8 = "nope"
}`)
	})
}

func TestParseTypedDeclaration(t *testing.T) {
	t.Parallel()

	t.Run("int64", func(t *testing.T) {
		t.Parallel()
		f := parse(t, `package p
x : int64 = 42
main : proc() = {}`)

		d := stmtAs[*ast.Declaration](t, f, 0)
		if d.Assignment.Identifier.Name != "x" {
			t.Errorf("expected name 'x', got %q", d.Assignment.Identifier.Name)
		}

		if d.Assignment.Identifier.ValueType.Kind() != types.Int64 {
			t.Errorf("expected Int64, got %s", d.Assignment.Identifier.ValueType.Kind())
		}
	})

	t.Run("string", func(t *testing.T) {
		t.Parallel()
		f := parse(t, `package p
s : utf8 = "hello"
main : proc() = {}`)

		d := stmtAs[*ast.Declaration](t, f, 0)
		if d.Assignment.Identifier.ValueType.Kind() != types.UTF8 {
			t.Errorf("expected UTF8, got %s", d.Assignment.Identifier.ValueType.Kind())
		}
	})
}
