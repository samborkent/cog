package transpiler_test

import "testing"

func TestConvertBuiltin(t *testing.T) {
	t.Parallel()

	t.Run("print", func(t *testing.T) {
		t.Parallel()
		got := transpile(t, `package p
main : proc() = {
	@print("hello")
}`)
		mustContain(t, got, "builtin.Print")
	})

	t.Run("if", func(t *testing.T) {
		t.Parallel()
		got := transpile(t, `package p
main : proc() = {
	x := @if(true, 1, 2)
	@print(x)
}`)
		mustContain(t, got, "builtin.If")
	})

	t.Run("slice", func(t *testing.T) {
		t.Parallel()
		got := transpile(t, `package p
main : proc() = {
	xs := @slice<int64>(3)
	@print(xs)
}`)
		mustContain(t, got, "make([]int64,")
	})

	t.Run("slice with capacity", func(t *testing.T) {
		t.Parallel()
		got := transpile(t, `package p
main : proc() = {
	xs := @slice<int64>(3, 10)
	@print(xs)
}`)
		mustContain(t, got, "make([]int64,")
	})

	t.Run("map", func(t *testing.T) {
		t.Parallel()
		got := transpile(t, `package p
main : proc() = {
	m := @map<utf8, int64>()
	@print(m)
}`)
		mustContain(t, got, "make(map[string]int64)")
	})

	t.Run("set", func(t *testing.T) {
		t.Parallel()
		got := transpile(t, `package p
main : proc() = {
	s := @set<int64>()
	@print(s)
}`)
		mustContain(t, got, "make(cog.Set[int64])")
	})

	t.Run("ref", func(t *testing.T) {
		t.Parallel()
		got := transpile(t, `package p
main : proc() = {
	ref := @ref<utf8>()
	_ = ref
}`)
		mustContain(t, got, "new(string)")
	})
}

func TestConvertBuiltinCast(t *testing.T) {
	t.Parallel()

	t.Run("direct uint8 to uint32", func(t *testing.T) {
		t.Parallel()
		got := transpile(t, `package p
main : proc() = {
	x : uint8 = 1
	y := @cast<uint32>(x)
	@print(y)
}`)
		mustContain(t, got, "uint32(x)")
	})

	t.Run("direct int8 to int32", func(t *testing.T) {
		t.Parallel()
		got := transpile(t, `package p
main : proc() = {
	x : int8 = 1
	y := @cast<int32>(x)
	@print(y)
}`)
		mustContain(t, got, "int32(x)")
	})

	t.Run("direct float32 to float64", func(t *testing.T) {
		t.Parallel()
		got := transpile(t, `package p
main : proc() = {
	x : float32 = 1.0
	y := @cast<float64>(x)
	@print(y)
}`)
		mustContain(t, got, "float64(x)")
	})

	t.Run("direct int32 to uint32", func(t *testing.T) {
		t.Parallel()
		got := transpile(t, `package p
main : proc() = {
	x : int32 = 1
	y := @cast<uint32>(x)
	@print(y)
}`)
		mustContain(t, got, "uint32(x)")
	})

	t.Run("cross-family int32 to float32", func(t *testing.T) {
		t.Parallel()
		got := transpile(t, `package p
main : proc() = {
	x : int32 = 42
	y := @cast<float32>(x)
	@print(y)
}`)
		mustContain(t, got, "Float32frombits")
	})

	t.Run("bool to uint8", func(t *testing.T) {
		t.Parallel()
		got := transpile(t, `package p
main : proc() = {
	x := true
	y := @cast<uint8>(x)
	@print(y)
}`)
		mustContain(t, got, "builtin.If[uint8]")
	})

	t.Run("float16 to uint32", func(t *testing.T) {
		t.Parallel()
		got := transpile(t, `package p
main : proc() = {
	x : float16 = 1.5
	y := @cast<uint32>(x)
	@print(y)
}`)
		mustContain(t, got, ".Bits()")
		mustContain(t, got, "uint32(")
	})

	t.Run("uint64 to int128", func(t *testing.T) {
		t.Parallel()
		got := transpile(t, `package p
main : proc() = {
	x : uint64 = 42
	y := @cast<int128>(x)
	@print(y)
}`)
		mustContain(t, got, "cog.Uint128ToInt128")
		mustContain(t, got, "cog.Uint128From64")
	})

	t.Run("direct uint16 to int16", func(t *testing.T) {
		t.Parallel()
		got := transpile(t, `package p
main : proc() = {
	x : uint16 = 1
	y := @cast<int16>(x)
	@print(y)
}`)
		mustContain(t, got, "int16(x)")
	})

	t.Run("uint8 to bool", func(t *testing.T) {
		t.Parallel()
		got := transpile(t, `package p
main : proc() = {
	x : uint8 = 1
	y := @cast<bool>(x)
	@print(y)
}`)
		mustContain(t, got, "!= 0")
	})
}
