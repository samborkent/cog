package parser_test

import "testing"

func TestParseIfStatement(t *testing.T) {
	t.Parallel()

	t.Run("simple", func(t *testing.T) {
		t.Parallel()
		f := parse(t, `package p
main : proc() = {
	if true {
		@print("yes")
	}
}`)
		if len(f.Statements) == 0 {
			t.Fatal("expected statements")
		}
	})

	t.Run("if_else", func(t *testing.T) {
		t.Parallel()
		f := parse(t, `package p
main : proc() = {
	if true {
		@print("yes")
	} else {
		@print("no")
	}
}`)
		if len(f.Statements) == 0 {
			t.Fatal("expected statements")
		}
	})

	t.Run("option_must_check_bare_access_error", func(t *testing.T) {
		t.Parallel()
		parseShouldError(t, `package p
main : proc() = {
	var x : int64?
	@print(x)
}`)
	})

	t.Run("option_checked_access_ok", func(t *testing.T) {
		t.Parallel()
		_ = parse(t, `package p
main : proc() = {
	var x : int64? = 42
	if x? {
		@print(x)
	}
}`)
	})

	t.Run("option_negated_check_else_ok", func(t *testing.T) {
		t.Parallel()
		_ = parse(t, `package p
main : proc() = {
	var x : int64? = 42
	if !x? {
		@print("not set")
	} else {
		@print(x)
	}
}`)
	})

	t.Run("option_negated_check_consequence_error", func(t *testing.T) {
		t.Parallel()
		parseShouldError(t, `package p
main : proc() = {
	var x : int64? = 42
	if !x? {
		@print(x)
	}
}`)
	})

	t.Run("result_must_check_bare_access_error", func(t *testing.T) {
		t.Parallel()
		parseShouldError(t, `package p
MyErr ~ int32
foo : func(a : int64, e : MyErr) int64 ! MyErr = {
	return a
}
main : proc() = {
	var r : int64 ! MyErr
	@print(r)
}`)
	})

	t.Run("result_suffix_in_error_block_ok", func(t *testing.T) {
		t.Parallel()
		_ = parse(t, `package p
MyErr ~ int32
main : proc() = {
	var r : int64 ! MyErr
	if r! {
		@print(r!)
	}
}`)
	})

	t.Run("result_negated_check_value_ok", func(t *testing.T) {
		t.Parallel()
		_ = parse(t, `package p
MyErr ~ int32
main : proc() = {
	var r : int64 ! MyErr
	if !r! {
		@print(r)
	}
}`)
	})

	t.Run("result_direct_check_persists", func(t *testing.T) {
		t.Parallel()
		_ = parse(t, `package p
MyErr ~ int32
main : proc() = {
	var r : int64 ! MyErr
	if r! {
		@print(r!)
	}
	@print(r)
}`)
	})

	t.Run("result_negated_undoes_persistent_check", func(t *testing.T) {
		t.Parallel()
		parseShouldError(t, `package p
MyErr ~ int32
main : proc() = {
	var r : int64 ! MyErr
	if r! {
		@print(r!)
	}
	if !r! {
		@print(r)
	}
	@print(r)
}`)
	})

	t.Run("result_error_block_bare_access_error", func(t *testing.T) {
		t.Parallel()
		parseShouldError(t, `package p
MyErr ~ int32
main : proc() = {
	var r : int64 ! MyErr
	if r! {
		@print(r)
	}
}`)
	})

	t.Run("result_negated_else_bare_access_error", func(t *testing.T) {
		t.Parallel()
		parseShouldError(t, `package p
MyErr ~ int32
main : proc() = {
	var r : int64 ! MyErr
	if !r! {
		@print(r)
	} else {
		@print(r)
	}
}`)
	})
}
