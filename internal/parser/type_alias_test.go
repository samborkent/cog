package parser_test

import (
	"testing"

	"github.com/samborkent/cog/internal/ast"
	"github.com/samborkent/cog/internal/types"
)

func TestParseTypeAlias(t *testing.T) {
	t.Parallel()

	t.Run("simple", func(t *testing.T) {
		t.Parallel()
		f := parse(t, `package p
MyInt ~ int32
main : proc() = {}`)

		ta := stmtAs[*ast.Type](t, f, 0)
		if ta.Identifier.Name != "MyInt" {
			t.Errorf("expected name 'MyInt', got %q", ta.Identifier.Name)
		}

		if ta.Alias.Kind() != types.Int32 {
			t.Errorf("expected Int32, got %s", ta.Alias.Kind())
		}
	})

	t.Run("tuple", func(t *testing.T) {
		t.Parallel()
		f := parse(t, `package p
Pair ~ (int32, utf8)
main : proc() = {}`)

		ta := stmtAs[*ast.Type](t, f, 0)
		if ta.Alias.Kind() != types.TupleKind {
			t.Errorf("expected TupleKind, got %s", ta.Alias.Kind())
		}
	})

	t.Run("either", func(t *testing.T) {
		t.Parallel()
		f := parse(t, `package p
Either ~ int32 ^ utf8
main : proc() = {}`)

		ta := stmtAs[*ast.Type](t, f, 0)
		if ta.Alias.Kind() != types.EitherKind {
			t.Errorf("expected EitherKind, got %s", ta.Alias.Kind())
		}
	})

	t.Run("option", func(t *testing.T) {
		t.Parallel()
		f := parse(t, `package p
MaybeInt ~ int32?
main : proc() = {}`)

		ta := stmtAs[*ast.Type](t, f, 0)
		if ta.Alias.Kind() != types.OptionKind {
			t.Errorf("expected OptionKind, got %s", ta.Alias.Kind())
		}
	})

	t.Run("slice", func(t *testing.T) {
		t.Parallel()
		f := parse(t, `package p
Ints ~ []int32
main : proc() = {}`)

		ta := stmtAs[*ast.Type](t, f, 0)
		if ta.Alias.Kind() != types.SliceKind {
			t.Errorf("expected SliceKind, got %s", ta.Alias.Kind())
		}
	})

	t.Run("array", func(t *testing.T) {
		t.Parallel()
		f := parse(t, `package p
Triple ~ [3]int32
main : proc() = {}`)

		ta := stmtAs[*ast.Type](t, f, 0)
		if ta.Alias.Kind() != types.ArrayKind {
			t.Errorf("expected ArrayKind, got %s", ta.Alias.Kind())
		}
	})

	t.Run("map", func(t *testing.T) {
		t.Parallel()
		f := parse(t, `package p
Dict ~ map<utf8, int32>
main : proc() = {}`)

		ta := stmtAs[*ast.Type](t, f, 0)
		if ta.Alias.Kind() != types.MapKind {
			t.Errorf("expected MapKind, got %s", ta.Alias.Kind())
		}
	})

	t.Run("set", func(t *testing.T) {
		t.Parallel()
		f := parse(t, `package p
UniqueInts ~ set<int32>
main : proc() = {}`)

		ta := stmtAs[*ast.Type](t, f, 0)
		if ta.Alias.Kind() != types.SetKind {
			t.Errorf("expected SetKind, got %s", ta.Alias.Kind())
		}
	})

	t.Run("struct", func(t *testing.T) {
		t.Parallel()
		f := parse(t, `package p
Point ~ struct {
	x : int32
	y : int32
}
main : proc() = {}`)

		ta := stmtAs[*ast.Type](t, f, 0)
		if ta.Alias.Kind() != types.StructKind {
			t.Errorf("expected StructKind, got %s", ta.Alias.Kind())
		}
	})

	t.Run("struct_with_proc_field_no_return", func(t *testing.T) {
		t.Parallel()

		// proc() with no return type and no '=' must not confuse the parser.
		f := parse(t, `package p
Handler ~ struct {
	callback : proc()
}
main : proc() = {}`)

		ta := stmtAs[*ast.Type](t, f, 0)

		s, ok := ta.Alias.Underlying().(*types.Struct)
		if !ok {
			t.Fatal("expected *types.Struct")
		}

		if len(s.Fields) != 1 {
			t.Fatalf("expected 1 field, got %d", len(s.Fields))
		}

		if s.Fields[0].Type.Kind() != types.ProcedureKind {
			t.Errorf("expected ProcedureKind field, got %s", s.Fields[0].Type.Kind())
		}
	})

	t.Run("enum", func(t *testing.T) {
		t.Parallel()
		f := parse(t, `package p
Color ~ enum<utf8> {
	Red := "red",
	Green := "green",
	Blue := "blue",
}
main : proc() = {}`)

		ta := stmtAs[*ast.Type](t, f, 0)
		if ta.Alias.Kind() != types.EnumKind {
			t.Errorf("expected EnumKind, got %s", ta.Alias.Kind())
		}
	})

	t.Run("typed_error", func(t *testing.T) {
		t.Parallel()
		f := parse(t, `package p
MyError ~ error<utf8> {
	NotFound := "not found",
	Timeout := "timeout",
}
main : proc() = {}`)

		ta := stmtAs[*ast.Type](t, f, 0)
		if ta.Alias.Kind() != types.ErrorKind {
			t.Errorf("expected ErrorKind, got %s", ta.Alias.Kind())
		}
	})

	t.Run("typeless_error", func(t *testing.T) {
		t.Parallel()
		f := parse(t, `package p
MyError ~ error {
	NotFound,
	Timeout,
}
main : proc() = {}`)

		ta := stmtAs[*ast.Type](t, f, 0)
		if ta.Alias.Kind() != types.ErrorKind {
			t.Errorf("expected ErrorKind, got %s", ta.Alias.Kind())
		}
	})

	t.Run("ascii_error", func(t *testing.T) {
		t.Parallel()
		f := parse(t, `package p
MyError ~ error<ascii> {
	NotFound := "not found",
}
main : proc() = {}`)

		ta := stmtAs[*ast.Type](t, f, 0)
		if ta.Alias.Kind() != types.ErrorKind {
			t.Errorf("expected ErrorKind, got %s", ta.Alias.Kind())
		}
	})

	t.Run("error_invalid_type_param", func(t *testing.T) {
		t.Parallel()
		parseShouldError(t, `package p
MyError ~ error<int32> {
	Fail := 1,
}
main : proc() = {}`)
	})

	t.Run("result_requires_error_type", func(t *testing.T) {
		t.Parallel()
		parseShouldError(t, `package p
NotAnError ~ int32
main : proc() = {
	var r : int64 ! NotAnError
}`)
	})

	t.Run("result_value_cannot_be_error", func(t *testing.T) {
		t.Parallel()
		parseShouldError(t, `package p
MyErr ~ error { Fail }
OtherErr ~ error { Bad }
main : proc() = {
	var r : MyErr ! OtherErr
}`)
	})
}

func TestParseGenericTypeAlias(t *testing.T) {
	t.Parallel()

	t.Run("slice_of_T", func(t *testing.T) {
		t.Parallel()
		f := parse(t, `package p
List<T ~ any> ~ []T
main : proc() = {}`)

		ta := stmtAs[*ast.Type](t, f, 0)
		if ta.Identifier.Name != "List" {
			t.Errorf("expected name 'List', got %q", ta.Identifier.Name)
		}

		if len(ta.TypeParameters) != 1 {
			t.Fatalf("expected 1 type param, got %d", len(ta.TypeParameters))
		}

		if ta.TypeParameters[0].Name != "T" {
			t.Errorf("expected type param name 'T', got %q", ta.TypeParameters[0].Name)
		}

		if ta.TypeParameters[0].ConstraintString() != "any" {
			t.Errorf("expected constraint 'any', got %q", ta.TypeParameters[0].ConstraintString())
		}
	})

	t.Run("two_params", func(t *testing.T) {
		t.Parallel()
		f := parse(t, `package p
Pair<A ~ any, B ~ any> ~ (A, B)
main : proc() = {}`)
		ta := stmtAs[*ast.Type](t, f, 0)

		if len(ta.TypeParameters) != 2 {
			t.Fatalf("expected 2 type params, got %d", len(ta.TypeParameters))
		}

		if ta.TypeParameters[0].Name != "A" || ta.TypeParameters[1].Name != "B" {
			t.Errorf("expected params [A, B], got [%s, %s]", ta.TypeParameters[0].Name, ta.TypeParameters[1].Name)
		}
	})

	t.Run("constrained_param", func(t *testing.T) {
		t.Parallel()
		f := parse(t, `package p
NumList<T ~ number> ~ []T
main : proc() = {}`)
		ta := stmtAs[*ast.Type](t, f, 0)

		if ta.TypeParameters[0].ConstraintString() != "number" {
			t.Errorf("expected constraint 'number', got %q", ta.TypeParameters[0].ConstraintString())
		}
	})

	t.Run("multi_constraint", func(t *testing.T) {
		t.Parallel()
		f := parse(t, `package p
SList<T ~ string | int> ~ []T
main : proc() = {}`)
		ta := stmtAs[*ast.Type](t, f, 0)

		cs := ta.TypeParameters[0].ConstraintString()
		if cs != "string | int" {
			t.Errorf("expected constraint 'string | int', got %q", cs)
		}
	})

	t.Run("map_generic", func(t *testing.T) {
		t.Parallel()
		f := parse(t, `package p
Dict<K ~ comparable, V ~ any> ~ map<K, V>
main : proc() = {}`)
		ta := stmtAs[*ast.Type](t, f, 0)

		if len(ta.TypeParameters) != 2 {
			t.Fatalf("expected 2 type params, got %d", len(ta.TypeParameters))
		}
	})

	t.Run("instantiate_slice", func(t *testing.T) {
		t.Parallel()

		f := parse(t, `package p
List<T ~ any> ~ []T
main : proc() = {
	xs : List<int32> = @slice<int32>(3)
	@print(xs)
}`)
		if f.LenNodes() < 2 {
			t.Fatal("expected at least 2 statements")
		}
	})

	t.Run("instantiate_wrong_arity", func(t *testing.T) {
		t.Parallel()
		parseShouldError(t, `package p
List<T ~ any> ~ []T
main : proc() = {
	xs : List<int32, utf8> = @slice<int32>(3)
}`)
	})

	t.Run("instantiate_constraint_violation", func(t *testing.T) {
		t.Parallel()
		parseShouldError(t, `package p
NumList<T ~ number> ~ []T
main : proc() = {
	xs : NumList<utf8> = @slice<utf8>(3)
}`)
	})

	t.Run("forward_reference", func(t *testing.T) {
		t.Parallel()

		f := parse(t, `package p
names : List<utf8> = @slice<utf8>(3)
List<T ~ any> ~ []T
main : proc() = {
	@print(names)
}`)
		if f.LenNodes() < 3 {
			t.Fatalf("expected at least 3 statements, got %d", f.LenNodes())
		}
	})

	t.Run("comparison_not_confused_with_type_params", func(t *testing.T) {
		t.Parallel()

		f := parse(t, `package p
main : proc() = {
	index := 10
	if index < 5 {
		@print(index)
	}
}`)
		if f.LenNodes() < 1 {
			t.Fatal("expected at least 1 statement")
		}
	})

	t.Run("comparison_with_generic_alias_in_same_file", func(t *testing.T) {
		t.Parallel()

		f := parse(t, `package p
List<T ~ any> ~ []T
main : proc() = {
	index := 10
	if index < 5 {
		@print(index)
	}
	xs : List<int32> = @slice<int32>(3)
	@print(xs)
}`)
		if f.LenNodes() < 2 {
			t.Fatal("expected at least 2 statements")
		}
	})

	t.Run("instantiate_map_two_params", func(t *testing.T) {
		t.Parallel()

		f := parse(t, `package p
Dict<K ~ comparable, V ~ any> ~ map<K, V>
main : proc() = {
	m : Dict<utf8, int64> = @map<utf8, int64>()
	@print(m)
}`)
		if f.LenNodes() < 2 {
			t.Fatal("expected at least 2 statements")
		}
	})

	t.Run("ordered_constraint", func(t *testing.T) {
		t.Parallel()

		f := parse(t, `package p
SortableSlice<T ~ ordered> ~ []T
main : proc() = {
	words : SortableSlice<utf8> = @slice<utf8>(10)
	@print(words)
}`)
		if f.LenNodes() < 2 {
			t.Fatal("expected at least 2 statements")
		}
	})

	t.Run("comparable_constraint", func(t *testing.T) {
		t.Parallel()

		f := parse(t, `package p
Dict<K ~ comparable, V ~ any> ~ map<K, V>
main : proc() = {
	lookup : Dict<utf8, int64> = @map<utf8, int64>()
	@print(lookup)
}`)
		if f.LenNodes() < 2 {
			t.Fatal("expected at least 2 statements")
		}
	})

	t.Run("union_constraint_instantiate", func(t *testing.T) {
		t.Parallel()

		f := parse(t, `package p
TagSlice<T ~ string | int> ~ []T
main : proc() = {
	labels : TagSlice<utf8> = @slice<utf8>(3)
	@print(labels)
}`)
		if f.LenNodes() < 2 {
			t.Fatal("expected at least 2 statements")
		}
	})
}

func TestParseInterface(t *testing.T) {
	t.Parallel()

	t.Run("single_method", func(t *testing.T) {
		t.Parallel()

		f := parse(t, `package p
Stringer ~ interface {
	String : func() utf8
}
main : proc() = {}`)

		ta := stmtAs[*ast.Type](t, f, 0)
		if ta.Identifier.Name != "Stringer" {
			t.Errorf("expected name 'Stringer', got %q", ta.Identifier.Name)
		}

		if ta.Alias.Kind() != types.InterfaceKind {
			t.Errorf("expected InterfaceKind, got %s", ta.Alias.Kind())
		}

		iface, ok := ta.Alias.(*types.Interface)
		if !ok {
			t.Fatal("expected *types.Interface")
		}

		if len(iface.Methods) != 1 {
			t.Fatalf("expected 1 method, got %d", len(iface.Methods))
		}

		if iface.Methods[0].Name != "String" {
			t.Errorf("expected method name 'String', got %q", iface.Methods[0].Name)
		}
	})

	t.Run("multi_method", func(t *testing.T) {
		t.Parallel()

		f := parse(t, `package p
ReadWriter ~ interface {
	Read : func() int64
	Write : proc(data : utf8)
}
main : proc() = {}`)

		ta := stmtAs[*ast.Type](t, f, 0)

		iface, ok := ta.Alias.(*types.Interface)
		if !ok {
			t.Fatal("expected *types.Interface")
		}

		if len(iface.Methods) != 2 {
			t.Fatalf("expected 2 methods, got %d", len(iface.Methods))
		}

		if iface.Methods[0].Name != "Read" {
			t.Errorf("expected first method 'Read', got %q", iface.Methods[0].Name)
		}

		if iface.Methods[1].Name != "Write" {
			t.Errorf("expected second method 'Write', got %q", iface.Methods[1].Name)
		}
	})

	t.Run("empty_interface", func(t *testing.T) {
		t.Parallel()

		f := parse(t, `package p
Empty ~ interface {}
main : proc() = {}`)

		ta := stmtAs[*ast.Type](t, f, 0)

		iface, ok := ta.Alias.(*types.Interface)
		if !ok {
			t.Fatal("expected *types.Interface")
		}

		if len(iface.Methods) != 0 {
			t.Errorf("expected 0 methods, got %d", len(iface.Methods))
		}
	})

	t.Run("interface_as_constraint", func(t *testing.T) {
		t.Parallel()

		f := parse(t, `package p
Stringer ~ interface {
	String : func() utf8
}
export Foo ~ struct {
	value : utf8
}
export (f : Foo).String : func() utf8 = {
	return f.value
}
Print : func<T ~ Stringer>(x : T) = {
	@print(x.String())
}
main : proc() = {
	Print(Foo{
		value = "test",
	})
}`)

		if f.LenNodes() < 5 {
			t.Fatalf("expected at least 5 statements, got %d", f.LenNodes())
		}
	})

	t.Run("missing_brace_errors", func(t *testing.T) {
		t.Parallel()

		parseShouldError(t, `package p
Bad ~ interface
main : proc() = {}`)
	})

	t.Run("non_method_in_body_errors", func(t *testing.T) {
		t.Parallel()

		parseShouldError(t, `package p
Bad ~ interface {
	123
}
main : proc() = {}`)
	})
}
