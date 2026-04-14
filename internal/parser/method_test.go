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
}
