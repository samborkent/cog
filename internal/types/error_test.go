package types

import "testing"

func TestErrorType(t *testing.T) {
	t.Parallel()

	t.Run("kind", func(t *testing.T) {
		t.Parallel()

		e := &Error{}
		if e.Kind() != ErrorKind {
			t.Errorf("Error.Kind() = %v, want ErrorKind", e.Kind())
		}
	})

	t.Run("string_typed", func(t *testing.T) {
		t.Parallel()

		e := &Error{ValueType: Basics[UTF8]}

		got := e.String()
		if got != "error<utf8> {}" {
			t.Errorf("Error.String() = %q", got)
		}
	})

	t.Run("string_untyped", func(t *testing.T) {
		t.Parallel()

		e := &Error{}

		got := e.String()
		if got != "error {}" {
			t.Errorf("Error.String() = %q", got)
		}
	})

	t.Run("underlying", func(t *testing.T) {
		t.Parallel()

		e := &Error{}
		if e.Underlying() != e {
			t.Error("Error.Underlying() should return itself")
		}
	})
}
