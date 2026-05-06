package parser_test

import (
	"testing"

	"github.com/samborkent/cog/internal/ast"
	"github.com/samborkent/cog/internal/types"
)

func TestGenericFunctionCall(t *testing.T) {
	t.Parallel()

	t.Run("infer_utf8", func(t *testing.T) {
		t.Parallel()
		f := parse(t, `package p
genFunc : func<T ~ any>(x : T) = {
	@print(x)
}
main : proc() = {
	genFunc("hello")
}`)
		// main proc body should contain the call.
		if f.LenNodes() < 2 {
			t.Fatalf("expected at least 2 statements, got %d", f.LenNodes())
		}
	})

	t.Run("infer_int64", func(t *testing.T) {
		t.Parallel()

		f := parse(t, `package p
genFunc : func<T ~ any>(x : T) = {
	@print(x)
}
main : proc() = {
	genFunc(42)
}`)
		if f.LenNodes() < 2 {
			t.Fatalf("expected at least 2 statements, got %d", f.LenNodes())
		}
	})

	t.Run("explicit_type_arg", func(t *testing.T) {
		t.Parallel()

		f := parse(t, `package p
genFunc : func<T ~ any>(x : T) = {
	@print(x)
}
main : proc() = {
	genFunc<utf8>("hello")
}`)
		if f.LenNodes() < 2 {
			t.Fatalf("expected at least 2 statements, got %d", f.LenNodes())
		}
	})

	t.Run("infer_with_return_type", func(t *testing.T) {
		t.Parallel()

		f := parse(t, `package p
identity : func<T ~ any>(x : T) T = {
	return x
}
main : proc() = {
	result := identity("hello")
	@print(result)
}`)
		if f.LenNodes() < 2 {
			t.Fatalf("expected at least 2 statements, got %d", f.LenNodes())
		}
	})

	t.Run("explicit_with_return_type", func(t *testing.T) {
		t.Parallel()

		f := parse(t, `package p
identity : func<T ~ any>(x : T) T = {
	return x
}
main : proc() = {
	result := identity<utf8>("hello")
	@print(result)
}`)
		if f.LenNodes() < 2 {
			t.Fatalf("expected at least 2 statements, got %d", f.LenNodes())
		}
	})

	t.Run("constrained_number", func(t *testing.T) {
		t.Parallel()

		f := parse(t, `package p
showNum : func<T ~ number>(x : T) = {
	@print(x)
}
main : proc() = {
	showNum(42)
}`)
		if f.LenNodes() < 2 {
			t.Fatalf("expected at least 2 statements, got %d", f.LenNodes())
		}
	})

	t.Run("forward_reference", func(t *testing.T) {
		t.Parallel()

		f := parse(t, `package p
main : proc() = {
	genFunc("hello")
}
genFunc : func<T ~ any>(x : T) = {
	@print(x)
}`)
		if f.LenNodes() < 2 {
			t.Fatalf("expected at least 2 statements, got %d", f.LenNodes())
		}
	})

	t.Run("constraint_violation_error", func(t *testing.T) {
		t.Parallel()
		parseShouldError(t, `package p
numOnly : func<T ~ number>(x : T) = {
	@print(x)
}
main : proc() = {
	numOnly("hello")
}`)
	})

	t.Run("explicit_type_arg_mismatch_error", func(t *testing.T) {
		t.Parallel()
		parseShouldError(t, `package p
genFunc : func<T ~ any>(x : T) = {
	@print(x)
}
main : proc() = {
	genFunc<utf8>(42)
}`)
	})

	t.Run("explicit_constraint_violation_error", func(t *testing.T) {
		t.Parallel()
		parseShouldError(t, `package p
numOnly : func<T ~ number>(x : T) = {
	@print(x)
}
main : proc() = {
	numOnly<utf8>("hello")
}`)
	})

	t.Run("wrong_type_arg_count_error", func(t *testing.T) {
		t.Parallel()
		parseShouldError(t, `package p
genFunc : func<T ~ any>(x : T) = {
	@print(x)
}
main : proc() = {
	genFunc<utf8, int64>("hello")
}`)
	})

	t.Run("infer_call_with_comparison_in_same_scope", func(t *testing.T) {
		t.Parallel()

		f := parse(t, `package p
genFunc : func<T ~ any>(x : T) = {
	@print(x)
}
main : proc() = {
	index := 10
	if index < 5 {
		@print(index)
	}
	genFunc("hello")
}`)
		if f.LenNodes() < 2 {
			t.Fatalf("expected at least 2 statements, got %d", f.LenNodes())
		}
	})

	t.Run("type_args_in_ast_call", func(t *testing.T) {
		t.Parallel()
		f := parse(t, `package p
genFunc : func<T ~ any>(x : T) = {
	@print(x)
}
main : proc() = {
	genFunc("hello")
}`)
		// Get the main proc body and find the call.
		mainDecl := stmtAs[*ast.Declaration](t, f, 1)
		procLit := f.Expr(mainDecl.Assignment.Expr).(*ast.ProcedureLiteral)

		block := procLit.Body
		if len(block.Statements) == 0 {
			t.Fatal("expected statements in main block")
		}

		exprStmt, ok := f.Node(block.Statements[0]).(*ast.ExpressionStatement)
		if !ok {
			t.Fatalf("expected ExpressionStatement, got %T", f.Node(block.Statements[0]))
		}

		call, ok := f.Expr(exprStmt.Expr).(*ast.Call)
		if !ok {
			t.Fatalf("expected Call, got %T", f.Expr(exprStmt.Expr))
		}

		if len(call.TypeArgs) != 1 {
			t.Fatalf("expected 1 type arg, got %d", len(call.TypeArgs))
		}

		if call.TypeArgs[0].Kind() != types.UTF8 {
			t.Errorf("expected inferred type arg utf8, got %s", call.TypeArgs[0])
		}
	})

	// Critical #1: zero-arg calls must not corrupt parser state.
	t.Run("zero_arg_proc_call", func(t *testing.T) {
		t.Parallel()

		f := parse(t, `package p
noArgs : proc() = {
	@print("hello")
}
main : proc() = {
	noArgs()
}`)
		if f.LenNodes() < 2 {
			t.Fatalf("expected at least 2 statements, got %d", f.LenNodes())
		}
	})

	t.Run("zero_arg_func_call", func(t *testing.T) {
		t.Parallel()

		f := parse(t, `package p
getVal : func() int64 = {
	return 42
}
main : proc() = {
	x := getVal()
	@print(x)
}`)
		if f.LenNodes() < 2 {
			t.Fatalf("expected at least 2 statements, got %d", f.LenNodes())
		}
	})

	// Critical #2: explicit type arg validation failure on func with return type
	// must not produce a Call with nil ReturnType (which would panic).
	t.Run("explicit_validation_failure_with_return_type", func(t *testing.T) {
		t.Parallel()
		parseShouldError(t, `package p
identity : func<T ~ any>(x : T) T = {
	return x
}
main : proc() = {
	result := identity<utf8>(42)
	@print(result)
}`)
	})

	t.Run("explicit_constraint_failure_with_return_type", func(t *testing.T) {
		t.Parallel()
		parseShouldError(t, `package p
numOnly : func<T ~ number>(x : T) T = {
	return x
}
main : proc() = {
	result := numOnly<utf8>("hello")
	@print(result)
}`)
	})

	// Major #9: concrete type constraints (not just keywords).
	t.Run("concrete_type_constraint", func(t *testing.T) {
		t.Parallel()

		f := parse(t, `package p
intOnly : func<T ~ int32 | int64>(x : T) = {
	@print(x)
}
main : proc() = {
	intOnly(42)
}`)
		if f.LenNodes() < 2 {
			t.Fatalf("expected at least 2 statements, got %d", f.LenNodes())
		}

		// Verify the first func's type parameter has concrete type constraints.
		decl := stmtAs[*ast.Declaration](t, f, 0)
		procLit := f.Expr(decl.Assignment.Expr).(*ast.ProcedureLiteral)

		procType := procLit.ProcedureType.(*types.Procedure)
		if len(procType.TypeParams) != 1 {
			t.Fatalf("expected 1 type param, got %d", len(procType.TypeParams))
		}

		tp := procType.TypeParams[0]
		if tp.Name != "T" {
			t.Errorf("expected type param name T, got %s", tp.Name)
		}

		union, ok := tp.Constraint.(*types.Union)
		if !ok || len(union.Variants) != 2 {
			t.Fatalf("expected 2 constraints, got %v", tp.Constraint)
		}

		if union.Variants[0].Kind() != types.Int32 {
			t.Errorf("expected first constraint int32, got %s", union.Variants[0])
		}

		if union.Variants[1].Kind() != types.Int64 {
			t.Errorf("expected second constraint int64, got %s", union.Variants[1])
		}
	})

	t.Run("single_concrete_type_constraint", func(t *testing.T) {
		t.Parallel()

		f := parse(t, `package p
utf8Only : func<T ~ utf8>(x : T) = {
	@print(x)
}
main : proc() = {
	utf8Only("hello")
}`)
		if f.LenNodes() < 2 {
			t.Fatalf("expected at least 2 statements, got %d", f.LenNodes())
		}
	})
}
