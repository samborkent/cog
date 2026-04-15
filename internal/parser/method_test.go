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
Foo.GetValue : func() utf8 = {
	return this.value
}
main : proc() = {}`)

		// Statement 0: type alias, statement 1: method, statement 2: main
		m := stmtAs[*ast.Method](t, f, 1)

		if m.Receiver == nil {
			t.Fatal("method receiver is nil")
		}

		if m.Receiver.Name != "Foo" {
			t.Errorf("expected receiver name Foo, got %q", m.Receiver.Name)
		}

		if m.Declaration == nil {
			t.Fatal("method declaration is nil")
		}

		if m.Declaration.Assignment.Identifier.Name != "GetValue" {
			t.Errorf("expected method name GetValue, got %q", m.Declaration.Assignment.Identifier.Name)
		}
	})

	t.Run("this_return", func(t *testing.T) {
		t.Parallel()

		f := parse(t, `package p
Foo ~ struct {
	x : int64
}
Foo.Self : func() int64 = {
	return this.x
}
main : proc() = {}`)

		m := stmtAs[*ast.Method](t, f, 1)

		if m.Declaration == nil {
			t.Fatal("method declaration is nil")
		}

		procType, ok := m.Declaration.Assignment.Identifier.ValueType.(*types.Procedure)
		if !ok {
			t.Fatal("method type is not a procedure")
		}

		if procType.ReturnType == nil || procType.ReturnType.Kind() != types.Int64 {
			t.Errorf("expected return type int64, got %v", procType.ReturnType)
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
Foo.String : func() utf8 = {
	return this.value
}
main : proc() = {}`)

		m := stmtAs[*ast.Method](t, f, 2)

		if m.Receiver.Name != "Foo" {
			t.Errorf("expected receiver Foo, got %q", m.Receiver.Name)
		}

		if m.Declaration.Assignment.Identifier.Name != "String" {
			t.Errorf("expected method name String, got %q", m.Declaration.Assignment.Identifier.Name)
		}
	})

	t.Run("exported_method", func(t *testing.T) {
		t.Parallel()

		f := parse(t, `package p
export Foo ~ struct {
	value : utf8
}
export Foo.GetValue : func() utf8 = {
	return this.value
}
main : proc() = {}`)

		m := stmtAs[*ast.Method](t, f, 1)

		if !m.Declaration.Assignment.Identifier.Exported {
			t.Error("expected exported method")
		}
	})

	t.Run("reference_receiver", func(t *testing.T) {
		t.Parallel()

		f := parse(t, `package p
Foo ~ struct {
	value : utf8
}
&Foo.GetRef : func() utf8 = {
	return this.value
}
main : proc() = {}`)

		m := stmtAs[*ast.Method](t, f, 1)

		if !m.Reference {
			t.Error("expected reference receiver")
		}

		if m.Receiver.Name != "Foo" {
			t.Errorf("expected receiver Foo, got %q", m.Receiver.Name)
		}
	})

	t.Run("exported_reference_method", func(t *testing.T) {
		t.Parallel()

		f := parse(t, `package p
export Foo ~ struct {}
export &Foo.Mutate : proc() = {}
main : proc() = {}`)

		m := stmtAs[*ast.Method](t, f, 1)

		if !m.Reference {
			t.Error("expected reference receiver")
		}

		if !m.Declaration.Assignment.Identifier.Exported {
			t.Error("expected exported method")
		}
	})

	t.Run("method_proc_no_return", func(t *testing.T) {
		t.Parallel()

		f := parse(t, `package p
Foo ~ struct {
	name : utf8
}
Foo.Greet : proc() = {
	@print(this.name)
}
main : proc() = {}`)

		m := stmtAs[*ast.Method](t, f, 1)

		if m.Declaration.Assignment.Identifier.Name != "Greet" {
			t.Errorf("expected method name 'Greet', got %q", m.Declaration.Assignment.Identifier.Name)
		}
	})

	t.Run("multiple_methods_on_struct", func(t *testing.T) {
		t.Parallel()

		f := parse(t, `package p
Point ~ struct {
	x : int64
	y : int64
}
Point.GetX : func() int64 = {
	return this.x
}
Point.GetY : func() int64 = {
	return this.y
}
main : proc() = {}`)

		m1 := stmtAs[*ast.Method](t, f, 1)
		m2 := stmtAs[*ast.Method](t, f, 2)

		if m1.Declaration.Assignment.Identifier.Name != "GetX" {
			t.Errorf("expected first method 'GetX', got %q", m1.Declaration.Assignment.Identifier.Name)
		}

		if m2.Declaration.Assignment.Identifier.Name != "GetY" {
			t.Errorf("expected second method 'GetY', got %q", m2.Declaration.Assignment.Identifier.Name)
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
Adder.Add : func(n : int64) int64 = {
	return this.base + n
}
main : proc() = {}`)

		m := stmtAs[*ast.Method](t, f, 1)

		procType, ok := m.Declaration.Assignment.Identifier.ValueType.(*types.Procedure)
		if !ok {
			t.Fatal("expected procedure type")
		}

		if len(procType.Parameters) != 1 {
			t.Fatalf("expected 1 param, got %d", len(procType.Parameters))
		}

		if procType.Parameters[0].Name != "n" {
			t.Errorf("expected param 'n', got %q", procType.Parameters[0].Name)
		}
	})

	t.Run("method_declaration_order", func(t *testing.T) {
		t.Parallel()

		// Methods can be declared before or after the struct.
		f := parse(t, `package p
Foo.Method : proc() = {}
Foo ~ struct {}
main : proc() = {}`)

		if len(f.Statements) < 3 {
			t.Fatalf("expected at least 3 statements, got %d", len(f.Statements))
		}
	})
}
