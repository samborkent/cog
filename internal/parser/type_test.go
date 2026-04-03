package parser_test

import (
	"testing"

	"github.com/samborkent/cog/internal/ast"
	"github.com/samborkent/cog/internal/types"
)

func TestParseProcedureType(t *testing.T) {
	t.Parallel()

	t.Run("basic_proc", func(t *testing.T) {
		t.Parallel()
		f := parse(t, `package p
main : proc() = {}`)
		d := stmtAs[*ast.Declaration](t, f, 0)
		if d.Assignment.Identifier.Name != "main" {
			t.Errorf("expected name 'main', got %q", d.Assignment.Identifier.Name)
		}
		if d.Assignment.Expression.Type().Kind() != types.ProcedureKind {
			t.Errorf("expected ProcedureKind, got %s", d.Assignment.Expression.Type().Kind())
		}
	})

	t.Run("with_params", func(t *testing.T) {
		t.Parallel()
		f := parse(t, `package p
greet : proc(name : utf8) = {
	@print(name)
}
main : proc() = {}`)
		d := stmtAs[*ast.Declaration](t, f, 0)
		procType, ok := d.Assignment.Expression.Type().(*types.Procedure)
		if !ok {
			t.Fatal("expected procedure type")
		}
		if len(procType.Parameters) != 1 {
			t.Fatalf("expected 1 parameter, got %d", len(procType.Parameters))
		}
		if procType.Parameters[0].Name != "name" {
			t.Errorf("expected param name 'name', got %q", procType.Parameters[0].Name)
		}
	})

	t.Run("func_with_return", func(t *testing.T) {
		t.Parallel()
		f := parse(t, `package p
add : func(a : int64, b : int64) int64 = {
	return a + b
}
main : proc() = {}`)
		d := stmtAs[*ast.Declaration](t, f, 0)
		procType, ok := d.Assignment.Expression.Type().(*types.Procedure)
		if !ok {
			t.Fatal("expected procedure type")
		}
		if !procType.Function {
			t.Error("expected Function flag to be true")
		}
		if procType.ReturnType == nil {
			t.Fatal("expected return type")
		}
		if procType.ReturnType.Kind() != types.Int64 {
			t.Errorf("expected Int64 return, got %s", procType.ReturnType.Kind())
		}
		if len(procType.Parameters) != 2 {
			t.Errorf("expected 2 parameters, got %d", len(procType.Parameters))
		}
	})

	t.Run("optional_param", func(t *testing.T) {
		t.Parallel()
		f := parse(t, `package p
greet : func(name : utf8, greeting? : utf8 = "hello") utf8 = {
	return greeting + " " + name
}
main : proc() = {}`)
		d := stmtAs[*ast.Declaration](t, f, 0)
		procType, ok := d.Assignment.Expression.Type().(*types.Procedure)
		if !ok {
			t.Fatal("expected procedure type")
		}
		if len(procType.Parameters) != 2 {
			t.Fatalf("expected 2 parameters, got %d", len(procType.Parameters))
		}
		if procType.Parameters[1].Default == nil {
			t.Error("expected second parameter to have a default value")
		}
	})

	t.Run("main_as_func_error", func(t *testing.T) {
		t.Parallel()
		parseShouldError(t, `package p
main : func() = {}`)
	})

	t.Run("main_with_params_error", func(t *testing.T) {
		t.Parallel()
		parseShouldError(t, `package p
main : proc(x : int32) = {}`)
	})

	t.Run("result_return_type", func(t *testing.T) {
		t.Parallel()
		f := parse(t, `package p
MyError ~ error { Fail }
divide : func(a : int64, b : int64) int64 ! MyError = {
	return a
}
main : proc() = {}`)
		d := stmtAs[*ast.Declaration](t, f, 1)
		procType, ok := d.Assignment.Expression.Type().(*types.Procedure)
		if !ok {
			t.Fatal("expected procedure type")
		}
		if procType.ReturnType == nil {
			t.Fatal("expected return type")
		}
		if procType.ReturnType.Kind() != types.ResultKind {
			t.Errorf("expected ResultKind, got %s", procType.ReturnType.Kind())
		}
		resultType, ok := procType.ReturnType.(*types.Result)
		if !ok {
			t.Fatal("expected *types.Result")
		}
		if resultType.Value.Kind() != types.Int64 {
			t.Errorf("expected value type Int64, got %s", resultType.Value.Kind())
		}
	})

	t.Run("result_return_wraps_value", func(t *testing.T) {
		t.Parallel()
		f := parse(t, `package p
MyError ~ error { Fail }
divide : func(a : int64, b : int64) int64 ! MyError = {
	return a
}
main : proc() = {}`)
		d := stmtAs[*ast.Declaration](t, f, 1)
		procLit, ok := d.Assignment.Expression.(*ast.ProcedureLiteral)
		if !ok {
			t.Fatal("expected ProcedureLiteral")
		}
		ret, ok := procLit.Body.Statements[0].(*ast.Return)
		if !ok {
			t.Fatal("expected Return statement")
		}
		if len(ret.Values) != 1 {
			t.Fatalf("expected 1 return value, got %d", len(ret.Values))
		}
		rl, ok := ret.Values[0].(*ast.ResultLiteral)
		if !ok {
			t.Fatalf("expected ResultLiteral, got %T", ret.Values[0])
		}
		if rl.IsError {
			t.Error("expected success return (IsError=false)")
		}
	})

	t.Run("result_return_wraps_error", func(t *testing.T) {
		t.Parallel()
		f := parse(t, `package p
MyError ~ error { Fail }
divide : func(a : int64, b : int64) int64 ! MyError = {
	return MyError.Fail
}
main : proc() = {}`)
		d := stmtAs[*ast.Declaration](t, f, 1)
		procLit, ok := d.Assignment.Expression.(*ast.ProcedureLiteral)
		if !ok {
			t.Fatal("expected ProcedureLiteral")
		}
		ret, ok := procLit.Body.Statements[0].(*ast.Return)
		if !ok {
			t.Fatal("expected Return statement")
		}
		if len(ret.Values) != 1 {
			t.Fatalf("expected 1 return value, got %d", len(ret.Values))
		}
		rl, ok := ret.Values[0].(*ast.ResultLiteral)
		if !ok {
			t.Fatalf("expected ResultLiteral, got %T", ret.Values[0])
		}
		if !rl.IsError {
			t.Error("expected error return (IsError=true)")
		}
	})

	t.Run("result_return_passthrough_not_wrapped", func(t *testing.T) {
		t.Parallel()
		f := parse(t, `package p
MyError ~ error { Fail }
inner : func(a : int64) int64 ! MyError = {
	return a
}
outer : func(a : int64) int64 ! MyError = {
	return inner(a)
}
main : proc() = {}`)
		d := stmtAs[*ast.Declaration](t, f, 2)
		procLit, ok := d.Assignment.Expression.(*ast.ProcedureLiteral)
		if !ok {
			t.Fatal("expected ProcedureLiteral")
		}
		ret, ok := procLit.Body.Statements[0].(*ast.Return)
		if !ok {
			t.Fatal("expected Return statement")
		}
		if len(ret.Values) != 1 {
			t.Fatalf("expected 1 return value, got %d", len(ret.Values))
		}
		if _, wrapped := ret.Values[0].(*ast.ResultLiteral); wrapped {
			t.Fatalf("expected pass-through result return, got %T", ret.Values[0])
		}
	})
}

func TestParseAnyType(t *testing.T) {
	t.Parallel()

	t.Run("type_alias_error", func(t *testing.T) {
		t.Parallel()
		parseShouldError(t, `package p
A ~ any
main : proc() = {}`)
	})

	t.Run("func_param_error", func(t *testing.T) {
		t.Parallel()
		parseShouldError(t, `package p
doSomething : proc(x : any) = {
	@print(x)
}
main : proc() = {}`)
	})

	t.Run("func_return_error", func(t *testing.T) {
		t.Parallel()
		parseShouldError(t, `package p
identity : func(x : utf8) any = {
	return x
}
main : proc() = {}`)
	})

	t.Run("var_decl_error", func(t *testing.T) {
		t.Parallel()
		parseShouldError(t, `package p
main : proc() = {
	var x : any
}`)
	})
}

func TestConstraintNamesRejectedInTypePosition(t *testing.T) {
	t.Parallel()

	constraints := []string{"int", "uint", "float", "complex", "string", "signed", "number", "ordered", "summable", "comparable"}

	for _, name := range constraints {
		t.Run(name+"_var_decl", func(t *testing.T) {
			t.Parallel()
			parseShouldError(t, `package p
main : proc() = {
	var x : `+name+`
}`)
		})

		t.Run(name+"_type_alias", func(t *testing.T) {
			t.Parallel()
			parseShouldError(t, `package p
A ~ `+name+`
main : proc() = {}`)
		})
	}
}
