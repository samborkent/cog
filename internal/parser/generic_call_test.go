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
		if len(f.Statements) < 2 {
			t.Fatalf("expected at least 2 statements, got %d", len(f.Statements))
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
		if len(f.Statements) < 2 {
			t.Fatalf("expected at least 2 statements, got %d", len(f.Statements))
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
		if len(f.Statements) < 2 {
			t.Fatalf("expected at least 2 statements, got %d", len(f.Statements))
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
		if len(f.Statements) < 2 {
			t.Fatalf("expected at least 2 statements, got %d", len(f.Statements))
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
		if len(f.Statements) < 2 {
			t.Fatalf("expected at least 2 statements, got %d", len(f.Statements))
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
		if len(f.Statements) < 2 {
			t.Fatalf("expected at least 2 statements, got %d", len(f.Statements))
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
		if len(f.Statements) < 2 {
			t.Fatalf("expected at least 2 statements, got %d", len(f.Statements))
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
		if len(f.Statements) < 2 {
			t.Fatalf("expected at least 2 statements, got %d", len(f.Statements))
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
		procLit := mainDecl.Assignment.Expression.(*ast.ProcedureLiteral)
		block := procLit.Body
		if len(block.Statements) == 0 {
			t.Fatal("expected statements in main block")
		}
		exprStmt, ok := block.Statements[0].(*ast.ExpressionStatement)
		if !ok {
			t.Fatalf("expected ExpressionStatement, got %T", block.Statements[0])
		}
		call, ok := exprStmt.Expression.(*ast.Call)
		if !ok {
			t.Fatalf("expected Call, got %T", exprStmt.Expression)
		}
		if len(call.TypeArgs) != 1 {
			t.Fatalf("expected 1 type arg, got %d", len(call.TypeArgs))
		}
		if call.TypeArgs[0].Kind() != types.UTF8 {
			t.Errorf("expected inferred type arg utf8, got %s", call.TypeArgs[0])
		}
	})
}
