package transpiler_test

import "testing"

func TestConvertType(t *testing.T) {
	t.Parallel()

	t.Run("simple_alias", func(t *testing.T) {
		t.Parallel()
		got := transpile(t, `package p
MyInt ~ int32
main : proc() = {}`)
		mustContain(t, got, "type _MyInt int32")
	})

	t.Run("exported_alias", func(t *testing.T) {
		t.Parallel()
		got := transpile(t, `package p
export myType ~ int32
main : proc() = {}`)
		mustContain(t, got, "type MyType int32")
	})

	t.Run("slice_type", func(t *testing.T) {
		t.Parallel()
		got := transpile(t, `package p
Nums ~ []int64
main : proc() = {}`)
		mustContain(t, got, "type _Nums []int64")
	})

	t.Run("array_type", func(t *testing.T) {
		t.Parallel()
		got := transpile(t, `package p
Triple ~ [3]int32
main : proc() = {}`)
		mustContain(t, got, "type _Triple [3]int32")
	})

	t.Run("map_type", func(t *testing.T) {
		t.Parallel()
		got := transpile(t, `package p
Dict ~ map<utf8, int64>
main : proc() = {}`)
		mustContain(t, got, "type _Dict map[string]int64")
	})

	t.Run("struct_type", func(t *testing.T) {
		t.Parallel()
		got := transpile(t, `package p
Point ~ struct {
	x : int32
	y : int32
}
main : proc() = {}`)
		mustContain(t, got, "type _Point struct")
		mustContain(t, got, "x\tint32")
		mustContain(t, got, "y\tint32")
	})

	t.Run("tuple_type", func(t *testing.T) {
		t.Parallel()
		got := transpile(t, `package p
Pair ~ (int32, utf8)
main : proc() = {}`)
		mustContain(t, got, "type _Pair struct")
	})

	t.Run("option_type", func(t *testing.T) {
		t.Parallel()
		got := transpile(t, `package p
MaybeInt ~ int32?
main : proc() = {}`)
		mustContain(t, got, "type _MaybeInt")
	})

	t.Run("either_type", func(t *testing.T) {
		t.Parallel()
		got := transpile(t, `package p
Either ~ int32 ^ utf8
main : proc() = {}`)
		mustContain(t, got, "type _Either")
	})

	t.Run("float16_type", func(t *testing.T) {
		t.Parallel()
		got := transpile(t, `package p
x : float16 = 1.0
main : proc() = {}`)
		mustContain(t, got, "cog.Float16")
	})

	t.Run("complex32_type", func(t *testing.T) {
		t.Parallel()
		got := transpile(t, `package p
c : complex32 = {1.0, 0.0}
main : proc() = {}`)
		mustContain(t, got, "cog.Complex32")
	})

	t.Run("uint128_type", func(t *testing.T) {
		t.Parallel()
		got := transpile(t, `package p
x : uint128 = 1
main : proc() = {}`)
		mustContain(t, got, "cog.Uint128")
	})

	t.Run("int128_type", func(t *testing.T) {
		t.Parallel()
		got := transpile(t, `package p
x : int128 = 1
main : proc() = {}`)
		mustContain(t, got, "cog.Int128")
	})

	t.Run("result_return_type", func(t *testing.T) {
		t.Parallel()
		got := transpile(t, `package p
MyError ~ error<utf8> {
	NotFound := "not found",
}
divide : func(a : int64, b : int64) int64 ! MyError = {
	return a
}
main : proc() = {}`)
		mustContain(t, got, "cog.Result[int64, _MyErrorError]")
	})

	t.Run("result_success_return", func(t *testing.T) {
		t.Parallel()
		got := transpile(t, `package p
MyError ~ error<utf8> {
	NotFound := "not found",
}
divide : func(a : int64, b : int64) int64 ! MyError = {
	return a
}
main : proc() = {}`)
		mustContain(t, got, "Value:")
	})

	t.Run("result_error_return", func(t *testing.T) {
		t.Parallel()
		got := transpile(t, `package p
MyError ~ error<utf8> {
	NotFound := "not found",
}
divide : func(a : int64, b : int64) int64 ! MyError = {
	return MyError.NotFound
}
main : proc() = {}`)
		mustContain(t, got, "Error:")
		mustContain(t, got, "IsError")
	})

	t.Run("result_return_passthrough", func(t *testing.T) {
		t.Parallel()
		got := transpile(t, `package p
MyError ~ error<utf8> {
	NotFound := "not found",
}
inner : func(a : int64) int64 ! MyError = {
	return a
}
outer : func(a : int64) int64 ! MyError = {
	return inner(a)
}
main : proc() = {}`)
		mustContain(t, got, "return inner(a)")
	})
}

func TestConvertEnumDecl(t *testing.T) {
	t.Parallel()

	t.Run("string_enum", func(t *testing.T) {
		t.Parallel()
		got := transpile(t, `package p
Status ~ enum<utf8> {
	Open := "open",
	Closed := "closed",
}
main : proc() = {}`)
		mustContain(t, got, "type _StatusEnum uint8")
		mustContain(t, got, "_StatusOpen")
		mustContain(t, got, "_StatusClosed")
		mustContain(t, got, "type _StatusType string")
	})
}

func TestGenericConstraintTranspilation(t *testing.T) {
	t.Parallel()

	// Critical #3: constraints must emit proper Go type sets, not 'any'.
	t.Run("number_constraint", func(t *testing.T) {
		t.Parallel()
		got := transpile(t, `package p
NumSlice<T ~ number> ~ []T
main : proc() = {}`)
		mustContain(t, got, "~int8")
		mustContain(t, got, "~uint64")
		mustContain(t, got, "~float64")
		mustNotContain(t, got, "[T any]")
	})

	t.Run("int_constraint", func(t *testing.T) {
		t.Parallel()
		got := transpile(t, `package p
IntSlice<T ~ int> ~ []T
main : proc() = {}`)
		mustContain(t, got, "~int8")
		mustContain(t, got, "~int64")
		mustNotContain(t, got, "~uint")
	})

	t.Run("uint_constraint", func(t *testing.T) {
		t.Parallel()
		got := transpile(t, `package p
UintSlice<T ~ uint> ~ []T
main : proc() = {}`)
		mustContain(t, got, "~uint8")
		mustContain(t, got, "~uint64")
		mustNotContain(t, got, "~int8")
	})

	t.Run("float_constraint", func(t *testing.T) {
		t.Parallel()
		got := transpile(t, `package p
FloatSlice<T ~ float> ~ []T
main : proc() = {}`)
		mustContain(t, got, "~float32")
		mustContain(t, got, "~float64")
		mustNotContain(t, got, "~int")
	})

	t.Run("complex_constraint", func(t *testing.T) {
		t.Parallel()
		got := transpile(t, `package p
ComplexSlice<T ~ complex> ~ []T
main : proc() = {}`)
		mustContain(t, got, "~complex64")
		mustContain(t, got, "~complex128")
	})

	t.Run("string_constraint", func(t *testing.T) {
		t.Parallel()
		got := transpile(t, `package p
StrSlice<T ~ string> ~ []T
main : proc() = {}`)
		mustContain(t, got, "~string")
	})

	t.Run("ordered_uses_cmp", func(t *testing.T) {
		t.Parallel()
		got := transpile(t, `package p
Sortable<T ~ ordered> ~ []T
main : proc() = {}`)
		mustContain(t, got, "go_cmp.Ordered")
	})

	t.Run("comparable_builtin", func(t *testing.T) {
		t.Parallel()
		got := transpile(t, `package p
Dict<K ~ comparable, V ~ any> ~ map<K, V>
main : proc() = {}`)
		mustContain(t, got, "K comparable")
		mustContain(t, got, "V any")
	})

	// Critical #4: multi-constraint union must not produce any|any.
	t.Run("multi_constraint_no_any_dup", func(t *testing.T) {
		t.Parallel()
		got := transpile(t, `package p
TagSlice<T ~ string | int> ~ []T
main : proc() = {}`)
		mustContain(t, got, "~string")
		mustContain(t, got, "~int8")
		mustNotContain(t, got, "any | any")
	})

	t.Run("multi_constraint_flat_no_parens", func(t *testing.T) {
		t.Parallel()
		got := transpile(t, `package p
TagSlice<T ~ string | int> ~ []T
main : proc() = {}`)
		// Should be flat: ~string | ~int8 | ~int16 | ...
		// Must NOT contain parenthesized sub-expressions.
		mustNotContain(t, got, "(~")
	})

	t.Run("any_constraint_absorbs_union", func(t *testing.T) {
		t.Parallel()
		got := transpile(t, `package p
List<T ~ any> ~ []T
main : proc() = {}`)
		mustContain(t, got, "T any")
	})

	t.Run("generic_func_constraint", func(t *testing.T) {
		t.Parallel()
		got := transpile(t, `package p
showNum : func<T ~ number>(x : T) = {
	@print(x)
}
main : proc() = {
	showNum(42)
}`)
		mustContain(t, got, "~int8")
		mustContain(t, got, "~float64")
		mustNotContain(t, got, "[T any]")
	})

	// Generic declaration: func with type params emits Go [T constraint] syntax.
	t.Run("generic_func_decl_syntax", func(t *testing.T) {
		t.Parallel()
		got := transpile(t, `package p
identity : func<T ~ any>(x : T) T = {
	return x
}
main : proc() = {
	y := identity(42)
	@print(y)
}`)
		mustContain(t, got, "[T any]")
		mustContain(t, got, "func identity")
	})

	// Generic call: inferred type arg emits Go [TypeArg] at call site.
	t.Run("generic_call_inferred_type_arg", func(t *testing.T) {
		t.Parallel()
		got := transpile(t, `package p
echo : func<T ~ any>(x : T) T = {
	return x
}
main : proc() = {
	result := echo("hello")
	@print(result)
}`)
		// Call site should have [string] type arg (utf8 -> Go string).
		mustContain(t, got, "echo[string]")
	})

	// Generic call: explicit type arg emits Go [TypeArg] at call site.
	t.Run("generic_call_explicit_type_arg", func(t *testing.T) {
		t.Parallel()
		got := transpile(t, `package p
echo : func<T ~ any>(x : T) T = {
	return x
}
main : proc() = {
	result := echo<int64>(42)
	@print(result)
}`)
		mustContain(t, got, "echo[int64]")
	})

	// Multi-param generic declaration.
	t.Run("generic_multi_param_decl", func(t *testing.T) {
		t.Parallel()
		got := transpile(t, `package p
Pair<K ~ comparable, V ~ any> ~ struct {
	key : K
	value : V
}
main : proc() = {}`)
		mustContain(t, got, "K comparable")
		mustContain(t, got, "V any")
	})

	// Generic func with constrained type param: call preserves constraint.
	t.Run("generic_func_constrained_call", func(t *testing.T) {
		t.Parallel()
		got := transpile(t, `package p
show : func<T ~ int>(x : T) = {
	@print(x)
}
main : proc() = {
	show(21)
	@print("done")
}`)
		// Declaration should have the int constraint union.
		mustContain(t, got, "~int8")
		mustContain(t, got, "~int64")
		// Call should have [int64] type arg.
		mustContain(t, got, "show[int64]")
	})

	// Signed constraint emits int + float + complex union.
	t.Run("signed_constraint", func(t *testing.T) {
		t.Parallel()
		got := transpile(t, `package p
SignedSlice<T ~ signed> ~ []T
main : proc() = {}`)
		mustContain(t, got, "~int8")
		mustContain(t, got, "~float32")
		mustContain(t, got, "~complex64")
		mustNotContain(t, got, "~uint")
	})

	// Summable constraint emits number + string union.
	t.Run("summable_constraint", func(t *testing.T) {
		t.Parallel()
		got := transpile(t, `package p
SumSlice<T ~ summable> ~ []T
main : proc() = {}`)
		mustContain(t, got, "~int8")
		mustContain(t, got, "~uint8")
		mustContain(t, got, "~float32")
		mustContain(t, got, "~complex64")
		mustContain(t, got, "~string")
	})
}
