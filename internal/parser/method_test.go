package parser_test

import (
	"testing"

	"github.com/samborkent/cog/internal/ast"
	"github.com/samborkent/cog/internal/types"
)

func TestParseMethod(t *testing.T) {
	t.Parallel()

	t.Run("this_field_access", func(t *testing.T) {
		t.Parallel()

		f := parse(t, `package p
Foo ~ struct {
	value : utf8
}
(f : Foo).GetValue : func() utf8 = {
	return f.value
}
main : proc() = {}`)

		// Statement 0: type alias, statement 1: method, statement 2: main
		m := stmtAs[*ast.Method](t, f, 1)

		if m.ReceiverIdent == nil {
			t.Fatal("method receiver identifier is nil")
		}

		if m.ReceiverIdent.Token.Literal != "f" {
			t.Errorf("expected receiver name f, got %q", m.ReceiverIdent.Token.Literal)
		}

		if m.ReceiverType == nil {
			t.Fatal("method receiver type is nil")
		}

		if m.ReceiverType.Kind() != types.StructKind {
			t.Errorf("expected struct kind receiver type, got %q", m.ReceiverType.Kind())
		}

		if m.Type == nil {
			t.Fatal("method type is nil")
		}

		if m.Body == ast.ZeroExprIndex {
			t.Fatal("method body is nil")
		}

		if m.Token.Literal != "GetValue" {
			t.Errorf("expected method name GetValue, got %q", m.Token.Literal)
		}
	})
	t.Run("this_return", func(t *testing.T) {
		t.Parallel()

		f := parse(t, `package p
Foo ~ struct {
	x : int64
}
(f : Foo).Self : func() int64 = {
	return f.x
}
main : proc() = {}`)

		m := stmtAs[*ast.Method](t, f, 1)

		if m.Type == nil {
			t.Fatal("method type is nil")
		}

		if m.Body == ast.ZeroExprIndex {
			t.Fatal("method body is nil")
		}

		if m.Type.ReturnType == nil || m.Type.ReturnType.Kind() != types.Int64 {
			t.Errorf("expected return type int64, got %v", m.Type.ReturnType)
		}
	})
	t.Run("this_outside_method_errors", func(t *testing.T) {
		t.Parallel()

		parseShouldError(t, `package p
main : proc() = {
	x := this
}`)
	})
	t.Run("this_in_function_errors", func(t *testing.T) {
		t.Parallel()

		parseShouldError(t, `package p
f : func(a : int64) int64 = {
	return this
}
main : proc() = {}`)
	})
	t.Run("method_name_shadows_global", func(t *testing.T) {
		t.Parallel()

		// A method name that matches a global symbol should not
		// trigger a redeclaration error.
		f := parse(t, `package p
String ~ utf8
Foo ~ struct {
	value : utf8
}
(f : Foo).String : func() utf8 = {
	return f.value
}
main : proc() = {}`)

		m := stmtAs[*ast.Method](t, f, 2)

		if m.ReceiverIdent.Token.Literal != "f" {
			t.Errorf("expected receiver f, got %q", m.ReceiverIdent.Token.Literal)
		}

		if m.Token.Literal != "String" {
			t.Errorf("expected method name String, got %q", m.Token.Literal)
		}
	})
	t.Run("exported_method", func(t *testing.T) {
		t.Parallel()

		f := parse(t, `package p
export Foo ~ struct {
	value : utf8
}
export (f : Foo).GetValue : func() utf8 = {
	return f.value
}
main : proc() = {}`)

		m := stmtAs[*ast.Method](t, f, 1)

		if !m.Export {
			t.Error("expected exported method")
		}
	})

	t.Run("reference_receiver", func(t *testing.T) {
		t.Parallel()

		f := parse(t, `package p
Foo ~ struct {
	value : utf8
}
(f : &Foo).GetRef : func() utf8 = {
	return f.value
}
main : proc() = {}`)

		m := stmtAs[*ast.Method](t, f, 1)

		if m.ReceiverType.Kind() != types.ReferenceKind {
			t.Error("expected reference receiver")
		}

		refType, ok := m.ReceiverType.(*types.Reference)
		if !ok {
			t.Fatalf("unable to cast reference receiver type")
		}

		if refType.Value.String() != "Foo" {
			t.Errorf("expected receiver Foo, got %q", refType.Value.String())
		}
	})
	t.Run("exported_reference_method", func(t *testing.T) {
		t.Parallel()

		f := parse(t, `package p
export Foo ~ struct {}
export &Foo.Mutate : proc() = {}
main : proc() = {}`)

		m := stmtAs[*ast.Method](t, f, 1)

		if m.Type.Kind() != types.ReferenceKind {
			t.Error("expected reference receiver")
		}

		if !m.Export {
			t.Error("expected exported method")
		}
	})
	t.Run("method_proc_no_return", func(t *testing.T) {
		t.Parallel()

		f := parse(t, `package p
Foo ~ struct {
	name : utf8
}
(f : Foo).Greet : proc() = {
	@print(f.name)
}
main : proc() = {}`)

		m := stmtAs[*ast.Method](t, f, 1)

		if m.Token.Literal != "Greet" {
			t.Errorf("expected method name 'Greet', got %q", m.Token.Literal)
		}
	})
	t.Run("multiple_methods_on_struct", func(t *testing.T) {
		t.Parallel()

		f := parse(t, `package p
Point ~ struct {
	x : int64
	y : int64
}
(p : Point).GetX : func() int64 = {
	return p.x
}
(p : Point).GetY : func() int64 = {
	return p.y
}
main : proc() = {}`)

		m1 := stmtAs[*ast.Method](t, f, 1)
		m2 := stmtAs[*ast.Method](t, f, 2)

		if m1.Token.Literal != "GetX" {
			t.Errorf("expected first method 'GetX', got %q", m1.Token.Literal)
		}

		if m2.Token.Literal != "GetY" {
			t.Errorf("expected second method 'GetY', got %q", m2.Token.Literal)
		}
	})
	t.Run("exported_method_on_unexported_type_errors", func(t *testing.T) {
		t.Parallel()

		parseShouldError(t, `package p
Foo ~ struct {
	value : utf8
}
export Foo.Bad : proc() = {}
main : proc() = {}`)
	})

	t.Run("method_on_undefined_receiver_errors", func(t *testing.T) {
		t.Parallel()

		parseShouldError(t, `package p
NotDefined.Method : proc() = {}
main : proc() = {}`)
	})

	t.Run("method_with_params", func(t *testing.T) {
		t.Parallel()

		f := parse(t, `package p
Adder ~ struct {
	base : int64
}
(a : Adder).Add : func(n : int64) int64 = {
	return a.base + n
}
main : proc() = {}`)

		m := stmtAs[*ast.Method](t, f, 1)

		if len(m.Type.Parameters) != 1 {
			t.Fatalf("expected 1 param, got %d", len(m.Type.Parameters))
		}

		if m.Type.Parameters[0].Name != "n" {
			t.Errorf("expected param 'n', got %q", m.Type.Parameters[0].Name)
		}
	})

	t.Run("method_declaration_order", func(t *testing.T) {
		t.Parallel()

		// Methods can be declared before or after the struct.
		f := parse(t, `package p
Foo.Method : proc() = {}
Foo ~ struct {}
main : proc() = {}`)

		if f.LenNodes() < 3 {
			t.Fatalf("expected at least 3 statements, got %d", f.LenNodes())
		}
	})

	t.Run("immutable_receiver_field_assign_errors", func(t *testing.T) {
		t.Parallel()

		parseShouldError(t, `package p
Foo ~ struct { value : utf8 }
(f : &Foo).Mutate : proc() utf8 = {
	f.value = "changed"
	return f.value
}
main : proc() = {}`)
	})

	t.Run("var_receiver_on_func_errors", func(t *testing.T) {
		t.Parallel()

		parseShouldError(t, `package p
Foo ~ struct { value : utf8 }
(f : var &Foo).Get : func() utf8 = {
	return f.value
}
main : proc() = {}`)
	})

	t.Run("duplicate_method_name_errors", func(t *testing.T) {
		t.Parallel()

		parseShouldError(t, `package p
Foo ~ struct { value : utf8 }
Foo.String : func() utf8 = {
	return "foo"
}
(f : var &Foo).String : proc() utf8 = {
	return f.value
}
main : proc() = {}`)
	})
}
