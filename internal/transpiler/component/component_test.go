package component_test

import (
	goast "go/ast"
	gotoken "go/token"
	"strings"
	"testing"

	"github.com/samborkent/cog/internal/ast"
	"github.com/samborkent/cog/internal/transpiler/component"
)

func TestIdent(t *testing.T) {
	t.Parallel()

	t.Run("basic", func(t *testing.T) {
		t.Parallel()

		ident := &ast.Identifier{Name: "foo", Exported: false, Global: false}
		got := component.Ident(ident)

		if got == nil {
			t.Fatal("expected non-nil ident")
		}

		if got.Name != "foo" {
			t.Errorf("expected name 'foo', got %q", got.Name)
		}
	})

	t.Run("nil", func(t *testing.T) {
		t.Parallel()

		got := component.Ident(nil)
		if got != nil {
			t.Errorf("expected nil for nil input, got %v", got)
		}
	})

	t.Run("cached", func(t *testing.T) {
		t.Parallel()

		ident := &ast.Identifier{Name: "cached", Exported: false, Global: false}
		a := component.Ident(ident)
		b := component.Ident(ident)

		if a != b {
			t.Error("expected same pointer for cached ident")
		}
	})

	t.Run("exported", func(t *testing.T) {
		t.Parallel()

		ident := &ast.Identifier{Name: "foo", Exported: true, Global: false}
		got := component.Ident(ident)

		if got.Name != "Foo" {
			t.Errorf("expected name 'Foo', got %q", got.Name)
		}
	})

	t.Run("global_unexported", func(t *testing.T) {
		t.Parallel()

		ident := &ast.Identifier{Name: "Foo", Exported: false, Global: true}
		got := component.Ident(ident)

		if got.Name != "_Foo" {
			t.Errorf("expected name '_Foo', got %q", got.Name)
		}
	})
}

func TestIdentName(t *testing.T) {
	t.Parallel()

	got := component.IdentName("any")

	if got == nil {
		t.Fatal("expected non-nil ident")
	}

	if got.Name != "any" {
		t.Errorf("expected name 'any', got %q", got.Name)
	}

	// Should be cached.
	got2 := component.IdentName("any")
	if got != got2 {
		t.Error("expected same pointer for cached ident")
	}
}

func TestSelector(t *testing.T) {
	t.Parallel()

	x := &goast.Ident{Name: "pkg"}
	sel := component.Selector(x, "Method")

	if sel == nil {
		t.Fatal("expected non-nil SelectorExpr")
	}

	if sel.X != x {
		t.Error("X field does not match input")
	}

	if sel.Sel.Name != "Method" {
		t.Errorf("expected sel name 'Method', got %q", sel.Sel.Name)
	}
}

func TestAssignDef(t *testing.T) {
	t.Parallel()

	lhs := &goast.Ident{Name: "x"}
	rhs := &goast.BasicLit{Kind: gotoken.INT, Value: "42"}

	stmt := component.AssignDef(lhs, rhs)

	if stmt == nil {
		t.Fatal("expected non-nil AssignStmt")
	}

	if stmt.Tok != gotoken.DEFINE {
		t.Errorf("expected DEFINE token, got %v", stmt.Tok)
	}

	if len(stmt.Lhs) != 1 || stmt.Lhs[0] != lhs {
		t.Error("lhs does not match input")
	}

	if len(stmt.Rhs) != 1 || stmt.Rhs[0] != rhs {
		t.Error("rhs does not match input")
	}
}

func TestNot(t *testing.T) {
	t.Parallel()

	x := &goast.Ident{Name: "ok"}
	expr := component.Not(x)

	if expr == nil {
		t.Fatal("expected non-nil UnaryExpr")
	}

	if expr.Op != gotoken.NOT {
		t.Errorf("expected NOT op, got %v", expr.Op)
	}

	if expr.X != x {
		t.Error("X field does not match input")
	}
}

func TestCall(t *testing.T) {
	t.Parallel()

	fun := &goast.Ident{Name: "fn"}
	arg1 := &goast.Ident{Name: "a"}
	arg2 := &goast.Ident{Name: "b"}

	expr := component.Call(fun, arg1, arg2)

	if expr == nil {
		t.Fatal("expected non-nil CallExpr")
	}

	if expr.Fun != fun {
		t.Error("Fun field does not match input")
	}

	if len(expr.Args) != 2 {
		t.Fatalf("expected 2 args, got %d", len(expr.Args))
	}
}

func TestTypeAssert(t *testing.T) {
	t.Parallel()

	t.Run("with_type", func(t *testing.T) {
		t.Parallel()

		x := &goast.Ident{Name: "v"}
		typ := &goast.Ident{Name: "int64"}

		expr := component.TypeAssert(x, typ)

		if expr == nil {
			t.Fatal("expected non-nil TypeAssertExpr")
		}

		if expr.X != x {
			t.Error("X field does not match input")
		}

		if expr.Type != typ {
			t.Error("Type field does not match input")
		}
	})

	t.Run("nil_type_for_switch", func(t *testing.T) {
		t.Parallel()

		x := &goast.Ident{Name: "v"}
		expr := component.TypeAssert(x, nil)

		if expr.Type != nil {
			t.Error("expected nil Type for type switch assertion")
		}
	})
}

func TestIfStmt(t *testing.T) {
	t.Parallel()

	t.Run("without_else", func(t *testing.T) {
		t.Parallel()

		cond := &goast.Ident{Name: "ok"}
		body := []goast.Stmt{&goast.EmptyStmt{}}

		stmt := component.IfStmt(cond, body, nil)

		if stmt == nil {
			t.Fatal("expected non-nil IfStmt")
		}

		if stmt.Cond != cond {
			t.Error("Cond field does not match input")
		}

		if len(stmt.Body.List) != 1 {
			t.Errorf("expected 1 body stmt, got %d", len(stmt.Body.List))
		}

		if stmt.Else != nil {
			t.Error("expected nil Else")
		}
	})

	t.Run("with_else", func(t *testing.T) {
		t.Parallel()

		cond := &goast.Ident{Name: "ok"}
		body := []goast.Stmt{&goast.EmptyStmt{}}
		elseBlock := component.BlockStmt(&goast.EmptyStmt{})

		stmt := component.IfStmt(cond, body, elseBlock)

		if stmt.Else == nil {
			t.Fatal("expected non-nil Else")
		}
	})
}

func TestBlockStmt(t *testing.T) {
	t.Parallel()

	s1 := &goast.EmptyStmt{}
	s2 := &goast.EmptyStmt{}

	block := component.BlockStmt(s1, s2)

	if block == nil {
		t.Fatal("expected non-nil BlockStmt")
	}

	if len(block.List) != 2 {
		t.Fatalf("expected 2 stmts, got %d", len(block.List))
	}
}

func TestKeyValue(t *testing.T) {
	t.Parallel()

	key := &goast.Ident{Name: "Left"}
	val := &goast.BasicLit{Kind: gotoken.INT, Value: "42"}

	kv := component.KeyValue(key, val)

	if kv == nil {
		t.Fatal("expected non-nil KeyValueExpr")
	}

	if kv.Key != key {
		t.Error("Key field does not match input")
	}

	if kv.Value != val {
		t.Error("Value field does not match input")
	}
}

func TestBoolLit(t *testing.T) {
	t.Parallel()

	t.Run("true", func(t *testing.T) {
		t.Parallel()

		got := component.BoolLit(true)
		if got.Name != "true" {
			t.Errorf("expected 'true', got %q", got.Name)
		}
	})

	t.Run("false", func(t *testing.T) {
		t.Parallel()

		got := component.BoolLit(false)
		if got.Name != "false" {
			t.Errorf("expected 'false', got %q", got.Name)
		}
	})
}

func TestConvertExport(t *testing.T) {
	t.Parallel()

	t.Run("export_uppercase", func(t *testing.T) {
		t.Parallel()

		got := component.ConvertExport("foo", true, false)
		if got != "Foo" {
			t.Errorf("expected 'Foo', got %q", got)
		}
	})

	t.Run("already_uppercase", func(t *testing.T) {
		t.Parallel()

		got := component.ConvertExport("Foo", true, false)
		if got != "Foo" {
			t.Errorf("expected 'Foo', got %q", got)
		}
	})

	t.Run("global_unexported", func(t *testing.T) {
		t.Parallel()

		got := component.ConvertExport("Foo", false, true)
		if got != "_Foo" {
			t.Errorf("expected '_Foo', got %q", got)
		}
	})

	t.Run("local_lowercase", func(t *testing.T) {
		t.Parallel()

		got := component.ConvertExport("foo", false, false)
		if got != "foo" {
			t.Errorf("expected 'foo', got %q", got)
		}
	})
}

func TestNumericLiterals(t *testing.T) {
	t.Parallel()

	t.Run("int8", func(t *testing.T) {
		got := component.Int8Lit(8)
		if got.Value != "8" {
			t.Errorf("expected '8', got %q", got.Value)
		}
	})

	t.Run("int16", func(t *testing.T) {
		got := component.Int16Lit(16)
		if got.Value != "16" {
			t.Errorf("expected '16', got %q", got.Value)
		}
	})

	t.Run("int32", func(t *testing.T) {
		got := component.Int32Lit(32)
		if got.Value != "32" {
			t.Errorf("expected '32', got %q", got.Value)
		}
	})

	t.Run("int64", func(t *testing.T) {
		got := component.Int64Lit(64)
		if got.Value != "64" {
			t.Errorf("expected '64', got %q", got.Value)
		}
	})

	t.Run("int128", func(t *testing.T) {
		got := component.Int128Lit("128")
		if got.Value != "128" {
			t.Errorf("expected '128', got %q", got.Value)
		}
	})

	t.Run("uint8", func(t *testing.T) {
		got := component.Uint8Lit(8)
		if got.Value != "8" {
			t.Errorf("expected '8', got %q", got.Value)
		}
	})

	t.Run("uint16", func(t *testing.T) {
		got := component.Uint16Lit(16)
		if got.Value != "16" {
			t.Errorf("expected '16', got %q", got.Value)
		}
	})

	t.Run("uint32", func(t *testing.T) {
		got := component.Uint32Lit(32)
		if got.Value != "32" {
			t.Errorf("expected '32', got %q", got.Value)
		}
	})

	t.Run("uint64", func(t *testing.T) {
		got := component.Uint64Lit(64)
		if got.Value != "64" {
			t.Errorf("expected '64', got %q", got.Value)
		}
	})

	t.Run("uint128", func(t *testing.T) {
		got := component.Uint128Lit("128")
		if got.Value != "128" {
			t.Errorf("expected '128', got %q", got.Value)
		}
	})

	t.Run("float32", func(t *testing.T) {
		got := component.Float32Lit(3.14)
		if !strings.Contains(got.Value, "3.14") {
			t.Errorf("expected '3.14', got %q", got.Value)
		}
	})

	t.Run("float64", func(t *testing.T) {
		got := component.Float64Lit(3.14159)
		if !strings.Contains(got.Value, "3.14159") {
			t.Errorf("expected '3.14159', got %q", got.Value)
		}
	})
}

func TestStringLiterals(t *testing.T) {
	t.Parallel()

	t.Run("utf8_simple", func(t *testing.T) {
		got := component.UTF8Lit("hello")
		if got.Value != `"hello"` {
			t.Errorf("expected '\"hello\"', got %q", got.Value)
		}
	})

	t.Run("utf8_raw", func(t *testing.T) {
		got := component.UTF8Lit("hello\nworld")
		if got.Value != "`hello\nworld`" {
			t.Errorf("expected '`hello\\nworld`', got %q", got.Value)
		}
	})

	t.Run("ascii", func(t *testing.T) {
		got := component.ASCIILit([]byte("abc"))
		if len(got.Elts) != 3 {
			t.Fatalf("expected 3 elements, got %d", len(got.Elts))
		}
	})
}

func TestResetCache(t *testing.T) {
	// Verify ResetCache doesn't panic.
	component.ResetCache()
}
