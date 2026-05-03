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
		if f.LenNodes() == 0 {
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
		if f.LenNodes() == 0 {
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

	t.Run("option_direct_check_persists", func(t *testing.T) {
		t.Parallel()
		_ = parse(t, `package p
main : proc() = {
	var x : int64? = 42
	if x? {
		@print(x)
	}
	@print(x)
}`)
	})

	t.Run("option_negated_check_does_not_persist", func(t *testing.T) {
		t.Parallel()
		parseShouldError(t, `package p
main : proc() = {
	var x : int64? = 42
	if !x? {
		@print("not set")
	}
	@print(x)
}`)
	})

	t.Run("result_must_check_bare_access_error", func(t *testing.T) {
		t.Parallel()
		parseShouldError(t, `package p
MyErr ~ error { Fail }
main : proc() = {
	var r : int64 ! MyErr
	@print(r)
}`)
	})

	t.Run("result_must_check_error_access_error", func(t *testing.T) {
		t.Parallel()
		parseShouldError(t, `package p
MyErr ~ error { Fail }
main : proc() = {
	var r : int64 ! MyErr
	@print(r!)
}`)
	})

	t.Run("result_checked_value_ok", func(t *testing.T) {
		t.Parallel()
		_ = parse(t, `package p
MyErr ~ error { Fail }
main : proc() = {
	var r : int64 ! MyErr
	if r? {
		@print(r)
	}
}`)
	})

	t.Run("result_checked_error_in_negated_ok", func(t *testing.T) {
		t.Parallel()
		_ = parse(t, `package p
MyErr ~ error { Fail }
main : proc() = {
	var r : int64 ! MyErr
	if !r? {
		@print(r!)
	}
}`)
	})

	t.Run("result_direct_check_persists", func(t *testing.T) {
		t.Parallel()
		_ = parse(t, `package p
MyErr ~ error { Fail }
main : proc() = {
	var r : int64 ! MyErr
	if r? {
		@print(r)
	}
	@print(r)
}`)
	})

	t.Run("result_negated_check_does_not_persist", func(t *testing.T) {
		t.Parallel()
		parseShouldError(t, `package p
MyErr ~ error { Fail }
main : proc() = {
	var r : int64 ! MyErr
	if !r? {
		@print(r!)
	}
	@print(r)
}`)
	})

	t.Run("result_value_in_error_branch_error", func(t *testing.T) {
		t.Parallel()
		parseShouldError(t, `package p
MyErr ~ error { Fail }
main : proc() = {
	var r : int64 ! MyErr
	if !r? {
		@print(r)
	}
}`)
	})

	t.Run("result_negated_else_value_ok", func(t *testing.T) {
		t.Parallel()
		_ = parse(t, `package p
MyErr ~ error { Fail }
main : proc() = {
	var r : int64 ! MyErr
	if !r? {
		@print(r!)
	} else {
		@print(r)
	}
}`)
	})

	// Early-exit promotion: negated check with return promotes value check.
	t.Run("result_negated_return_promotes_value", func(t *testing.T) {
		t.Parallel()
		_ = parse(t, `package p
MyErr ~ error { Fail }
safeDivide : func(a : int64, b : int64) int64 ! MyErr = {
	var r : int64 ! MyErr
	if !r? {
		return r!
	}
	@print(r)
	return r
}
main : proc() = {}`)
	})

	// Early-exit promotion: negated check with break promotes value check.
	t.Run("result_negated_break_promotes_value", func(t *testing.T) {
		t.Parallel()
		_ = parse(t, `package p
MyErr ~ error { Fail }
main : proc() = {
	var r : int64 ! MyErr
	for {
		if !r? {
			break
		}
		@print(r)
		break
	}
}`)
	})

	// Early-exit promotion: negated check with continue promotes value check.
	t.Run("result_negated_continue_promotes_value", func(t *testing.T) {
		t.Parallel()
		_ = parse(t, `package p
MyErr ~ error { Fail }
main : proc() = {
	var r : int64 ! MyErr
	for {
		if !r? {
			continue
		}
		@print(r)
		break
	}
}`)
	})

	// Without early exit, negated check still does not persist.
	t.Run("result_negated_no_exit_still_fails", func(t *testing.T) {
		t.Parallel()
		parseShouldError(t, `package p
MyErr ~ error { Fail }
main : proc() = {
	var r : int64 ! MyErr
	if !r? {
		@print(r!)
	}
	@print(r)
}`)
	})

	// Option: negated check with return promotes value check.
	t.Run("option_negated_return_promotes_value", func(t *testing.T) {
		t.Parallel()
		_ = parse(t, `package p
getValue : func() utf8 = {
	var opt : utf8?
	if !opt? {
		return "default"
	}
	return opt
}
main : proc() = {}`)
	})

	// Static analysis: value literal assigned → value access safe.
	t.Run("result_value_literal_assigned", func(t *testing.T) {
		t.Parallel()
		_ = parse(t, `package p
MyErr ~ error { Fail }
main : proc() = {
	var r : int64 ! MyErr = 42
	@print(r)
}`)
	})

	// Static analysis: error literal assigned → error access safe.
	t.Run("result_error_literal_assigned", func(t *testing.T) {
		t.Parallel()
		_ = parse(t, `package p
MyErr ~ error<utf8> {
	NotFound := "not found",
}
main : proc() = {
	var r : int64 ! MyErr = MyErr.NotFound
	@print(r!)
}`)
	})

	// Static analysis: function call returning same result → NOT safe.
	t.Run("result_func_call_not_safe", func(t *testing.T) {
		t.Parallel()
		parseShouldError(t, `package p
MyErr ~ error { Fail }
fn : func() int64 ! MyErr = {
	return 0
}
main : proc() = {
	var r : int64 ! MyErr = fn()
	@print(r)
}`)
	})

	// Static analysis: reassignment with value literal → value access safe.
	t.Run("result_reassignment_value_literal", func(t *testing.T) {
		t.Parallel()
		_ = parse(t, `package p
MyErr ~ error { Fail }
main : proc() = {
	var r : int64 ! MyErr
	r = 10 // value literal assigned, marked as checked value
	@print(r)
}`)
	})

	// Regression: a negated check after a direct check must not destroy
	// the persisted direct check state.
	t.Run("result_direct_then_negated_preserves", func(t *testing.T) {
		t.Parallel()
		_ = parse(t, `package p
MyErr ~ error { Fail }
main : proc() = {
	var r : int64 ! MyErr
	if r? {
		@print(r)
	}
	@print(r)
	if !r? {
		@print(r!)
	}
	@print(r)
}`)
	})
}
