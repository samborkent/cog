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
		if d.Assignment.Identifier.Token.Literal != "x" {
			t.Errorf("expected name 'x', got %q", d.Assignment.Identifier.Token.Literal)
		}

		if d.Assignment.Expr == ast.ZeroExprIndex {
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

		proc, ok := f.Expr(d.Assignment.Expr).(*ast.ProcedureLiteral)
		if !ok {
			t.Fatalf("expected ProcedureLiteral, got %T", d.Assignment.Expr)
		}

		block := f.Node(proc.Body).(*ast.Block)

		varDecl, ok := f.Node(block.Statements[0]).(*ast.Declaration)
		if !ok {
			t.Fatalf("expected Declaration, got %T", f.Node(block.Statements[0]))
		}

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
		if d.Assignment.Identifier.Token.Literal != "x" {
			t.Errorf("expected name 'x', got %q", d.Assignment.Identifier.Token.Literal)
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

func TestForwardValueReference(t *testing.T) {
	t.Parallel()

	t.Run("later_declared_global", func(t *testing.T) {
		t.Parallel()
		f := parse(t, `package p
foo : int64 = bar
bar : int64 = 42
main : proc() = {}`)

		d := stmtAs[*ast.Declaration](t, f, 0)
		if d.Assignment.Identifier.Token.Literal != "foo" {
			t.Errorf("expected name 'foo', got %q", d.Assignment.Identifier.Token.Literal)
		}

		if d.Assignment.Identifier.ValueType.Kind() != types.Int64 {
			t.Errorf("expected Int64, got %s", d.Assignment.Identifier.ValueType.Kind())
		}

		// The expression referencing 'bar' should resolve to int64.
		expr := f.Expr(d.Assignment.Expr)
		ident, ok := expr.(*ast.Identifier)
		if !ok {
			t.Fatalf("expected *ast.Identifier, got %T", expr)
		}

		if ident.Token.Literal != "bar" {
			t.Errorf("expected reference to 'bar', got %q", ident.Token.Literal)
		}

		if types.IsNone(ident.ValueType) {
			t.Error("forward value reference 'bar' was not resolved")
		}

		if ident.ValueType.Kind() != types.Int64 {
			t.Errorf("expected bar type Int64, got %s", ident.ValueType.Kind())
		}
	})

	t.Run("undefined_identifier_errors", func(t *testing.T) {
		t.Parallel()
		parseShouldError(t, `package p
foo : int64 = nonexistent
main : proc() = {}`)
	})

	t.Run("generic_alias_forward_reference", func(t *testing.T) {
		t.Parallel()
		f := parse(t, `package p
names : List<utf8> = @slice<utf8>(3)
List<T ~ any> ~ []T
main : proc() = {}`)

		d := stmtAs[*ast.Declaration](t, f, 0)

		if d.Assignment.Identifier.Token.Literal != "names" {
			t.Fatalf("expected name 'names', got %q", d.Assignment.Identifier.Token.Literal)
		}

		vt := d.Assignment.Identifier.ValueType
		alias, ok := vt.(*types.Alias)
		if !ok {
			t.Fatalf("ValueType = %T (%v), want *types.Alias", vt, vt)
		}

		if len(alias.TypeArgs) != 1 {
			t.Fatalf("TypeArgs len = %d, want 1", len(alias.TypeArgs))
		}

		if alias.TypeArgs[0] != types.Basics[types.UTF8] {
			t.Errorf("TypeArgs[0] = %v, want utf8", alias.TypeArgs[0])
		}

		if alias.Kind() != types.SliceKind {
			t.Errorf("Kind() = %v, want SliceKind", alias.Kind())
		}
	})
}
