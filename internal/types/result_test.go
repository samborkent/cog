package types

import "testing"

func TestResultType(t *testing.T) {
	t.Parallel()

	r := &Result{Value: Basics[Int64], Error: &Error{}}

	t.Run("kind", func(t *testing.T) {
		t.Parallel()

		if r.Kind() != ResultKind {
			t.Errorf("Result.Kind() = %v, want ResultKind", r.Kind())
		}
	})

	t.Run("string", func(t *testing.T) {
		t.Parallel()

		got := r.String()
		if got != "int64 ! error {}" {
			t.Errorf("Result.String() = %q", got)
		}
	})

	t.Run("underlying", func(t *testing.T) {
		t.Parallel()

		if r.Underlying() != r {
			t.Error("Result.Underlying() should return itself")
		}
	})
}
