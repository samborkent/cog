package parser_test

import (
	"testing"

	"github.com/samborkent/cog/internal/ast"
	"github.com/samborkent/cog/internal/types"
)

func TestParseMatch(t *testing.T) {
	t.Parallel()

	t.Run("generic_basic", func(t *testing.T) {
		t.Parallel()

		f := parse(t, `package p
show : func<T ~ int32 | utf8>(x : T) = {
	match x {
	case int32:
		@print(x)
	case utf8:
		@print(x)
	}
}
main : proc() = {}`)

		decl := stmtAs[*ast.Declaration](t, f, 0)
		procLit := decl.Assignment.Expression.(*ast.ProcedureLiteral)
		block := procLit.Body

		if len(block.Statements) == 0 {
			t.Fatal("expected statements in function body")
		}

		matchStmt, ok := block.Statements[0].(*ast.Match)
		if !ok {
			t.Fatalf("expected *ast.Match, got %T", block.Statements[0])
		}

		if matchStmt.Binding != nil {
			t.Error("expected no binding")
		}

		if len(matchStmt.Cases) != 2 {
			t.Fatalf("expected 2 cases, got %d", len(matchStmt.Cases))
		}

		if matchStmt.Cases[0].MatchType.Kind() != types.Int32 {
			t.Errorf("expected first case int32, got %s", matchStmt.Cases[0].MatchType)
		}

		if matchStmt.Cases[1].MatchType.Kind() != types.UTF8 {
			t.Errorf("expected second case utf8, got %s", matchStmt.Cases[1].MatchType)
		}
	})

	t.Run("generic_with_binding", func(t *testing.T) {
		t.Parallel()

		f := parse(t, `package p
show : func<T ~ int32 | utf8>(x : T) = {
	match val := x {
	case int32:
		@print(val)
	case utf8:
		@print(val)
	}
}
main : proc() = {}`)

		decl := stmtAs[*ast.Declaration](t, f, 0)
		procLit := decl.Assignment.Expression.(*ast.ProcedureLiteral)
		matchStmt := procLit.Body.Statements[0].(*ast.Match)

		if matchStmt.Binding == nil {
			t.Fatal("expected binding")
		}

		if matchStmt.Binding.Name != "val" {
			t.Errorf("expected binding name 'val', got %q", matchStmt.Binding.Name)
		}

		if len(matchStmt.Cases) != 2 {
			t.Fatalf("expected 2 cases, got %d", len(matchStmt.Cases))
		}
	})

	t.Run("generic_with_default", func(t *testing.T) {
		t.Parallel()

		f := parse(t, `package p
show : func<T ~ any>(x : T) = {
	match x {
	case int32:
		@print(x)
	default:
		@print(x)
	}
}
main : proc() = {}`)

		decl := stmtAs[*ast.Declaration](t, f, 0)
		procLit := decl.Assignment.Expression.(*ast.ProcedureLiteral)
		matchStmt := procLit.Body.Statements[0].(*ast.Match)

		if len(matchStmt.Cases) != 1 {
			t.Fatalf("expected 1 case, got %d", len(matchStmt.Cases))
		}

		if matchStmt.Default == nil {
			t.Fatal("expected default branch")
		}
	})

	t.Run("generic_tilde_case", func(t *testing.T) {
		t.Parallel()

		f := parse(t, `package p
show : func<T ~ any>(x : T) = {
	match x {
	case ~int32:
		@print(x)
	default:
		@print(x)
	}
}
main : proc() = {}`)

		decl := stmtAs[*ast.Declaration](t, f, 0)
		procLit := decl.Assignment.Expression.(*ast.ProcedureLiteral)
		matchStmt := procLit.Body.Statements[0].(*ast.Match)

		if !matchStmt.Cases[0].Tilde {
			t.Error("expected tilde flag to be true")
		}
	})

	// NOTE: Either union match tests are omitted from parser tests because
	// the expression parser currently greedily consumes '{' after a union-typed
	// identifier as a union literal initialization.
	// The Either paths are covered in transpiler tests where the union value
	// is passed as a function parameter.

	t.Run("error_non_union_subject", func(t *testing.T) {
		t.Parallel()

		parseShouldError(t, `package p
main : proc() = {
	x := 42
	match x {
	case int32:
		@print(x)
	}
}`)
	})

	t.Run("case_body_statements", func(t *testing.T) {
		t.Parallel()

		f := parse(t, `package p
show : func<T ~ int32 | utf8>(x : T) = {
	match x {
	case int32:
		@print(x)
		@print("int32")
	case utf8:
		@print(x)
	}
}
main : proc() = {}`)

		decl := stmtAs[*ast.Declaration](t, f, 0)
		procLit := decl.Assignment.Expression.(*ast.ProcedureLiteral)
		matchStmt := procLit.Body.Statements[0].(*ast.Match)

		if len(matchStmt.Cases[0].Body) != 2 {
			t.Errorf("expected 2 statements in first case, got %d", len(matchStmt.Cases[0].Body))
		}

		if len(matchStmt.Cases[1].Body) != 1 {
			t.Errorf("expected 1 statement in second case, got %d", len(matchStmt.Cases[1].Body))
		}
	})

	t.Run("any_constraint", func(t *testing.T) {
		t.Parallel()

		f := parse(t, `package p
show : func<T ~ any>(x : T) = {
	match x {
	case int64:
		@print(x)
	case utf8:
		@print(x)
	}
}
main : proc() = {}`)

		decl := stmtAs[*ast.Declaration](t, f, 0)
		procLit := decl.Assignment.Expression.(*ast.ProcedureLiteral)
		matchStmt := procLit.Body.Statements[0].(*ast.Match)

		if len(matchStmt.Cases) != 2 {
			t.Fatalf("expected 2 cases, got %d", len(matchStmt.Cases))
		}
	})

	t.Run("string_representation", func(t *testing.T) {
		t.Parallel()

		f := parse(t, `package p
show : func<T ~ int32 | utf8>(x : T) = {
	match val := x {
	case int32:
		@print(val)
	case utf8:
		@print(val)
	}
}
main : proc() = {}`)

		decl := stmtAs[*ast.Declaration](t, f, 0)
		procLit := decl.Assignment.Expression.(*ast.ProcedureLiteral)
		matchStmt := procLit.Body.Statements[0].(*ast.Match)

		s := matchStmt.String()
		if s == "" {
			t.Error("expected non-empty string representation")
		}

		mustContainStr(t, s, "match")
		mustContainStr(t, s, "case")
	})

	t.Run("match_case_string", func(t *testing.T) {
		t.Parallel()

		f := parse(t, `package p
show : func<T ~ any>(x : T) = {
	match x {
	case ~int32:
		@print(x)
	default:
		@print(x)
	}
}
main : proc() = {}`)

		decl := stmtAs[*ast.Declaration](t, f, 0)
		procLit := decl.Assignment.Expression.(*ast.ProcedureLiteral)
		matchStmt := procLit.Body.Statements[0].(*ast.Match)

		caseStr := matchStmt.Cases[0].String()

		mustContainStr(t, caseStr, "case")
		mustContainStr(t, caseStr, "~")
	})
}

func mustContainStr(t *testing.T, got, want string) {
	t.Helper()

	if len(got) == 0 {
		t.Errorf("output is empty, expected it to contain %q", want)

		return
	}

	for i := 0; i <= len(got)-len(want); i++ {
		if got[i:i+len(want)] == want {
			return
		}
	}

	t.Errorf("output missing %q\ngot:\n%s", want, got)
}
