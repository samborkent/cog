package parser_test

import "testing"

func TestParseAssignment(t *testing.T) {
	t.Parallel()

	t.Run("valid", func(t *testing.T) {
		t.Parallel()

		f := parse(t, `package p
main : proc() = {
	var x := 1
	x = 2
}`)
		if len(f.Statements) == 0 {
			t.Fatal("expected statements")
		}
	})

	t.Run("immutable_error", func(t *testing.T) {
		t.Parallel()
		parseShouldError(t, `package p
main : proc() = {
	x := 1
	x = 2
}`)
	})

	t.Run("undefined_error", func(t *testing.T) {
		t.Parallel()
		parseShouldError(t, `package p
main : proc() = {
	x = 1
}`)
	})

	t.Run("result_reassignment_clears_checked_state", func(t *testing.T) {
		t.Parallel()
		parseShouldError(t, `package p
MyErr ~ error { Fail }
get : func(x : int64) int64 ! MyErr = {
	return 1
}
main : proc() = {
	var r : int64 ! MyErr = 1
	if r? {
		@print(r)
	}
	r = get(1)
	@print(r)
}`)
	})

	t.Run("result_reassignment_requires_new_check", func(t *testing.T) {
		t.Parallel()
		_ = parse(t, `package p
MyErr ~ error { Fail }
get : func(x : int64) int64 ! MyErr = {
	return 1
}
main : proc() = {
	var r : int64 ! MyErr = 1
	r = get(1)
	if r? {
		@print(r)
	}
}`)
	})
}
