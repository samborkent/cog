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
Pair ~ int32 & utf8
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

	t.Run("union_type", func(t *testing.T) {
		t.Parallel()
		got := transpile(t, `package p
Either ~ int32 | utf8
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
