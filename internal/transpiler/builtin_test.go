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
	if y? {
		@print(y)
	}
}`)
		mustContain(t, got, "cog.Option[uint32]{Value: uint32(x), Set: true}")
	})

	t.Run("direct int8 to int32", func(t *testing.T) {
		t.Parallel()
		got := transpile(t, `package p
main : proc() = {
	x : int8 = 1
	y := @cast<int32>(x)
	if y? {
		@print(y)
	}
}`)
		mustContain(t, got, "cog.Option[int32]{Value: int32(x), Set: true}")
	})

	t.Run("direct float32 to float64", func(t *testing.T) {
		t.Parallel()
		got := transpile(t, `package p
main : proc() = {
	x : float32 = 1.0
	y := @cast<float64>(x)
	if y? {
		@print(y)
	}
}`)
		mustContain(t, got, "cog.Option[float64]{Value: float64(x), Set: true}")
	})

	t.Run("direct int32 to uint32", func(t *testing.T) {
		t.Parallel()
		got := transpile(t, `package p
main : proc() = {
	x : int32 = 1
	y := @cast<uint32>(x)
	if y? {
		@print(y)
	}
}`)
		mustContain(t, got, "cog.Option[uint32]{Value: uint32(x), Set: true}")
	})

	t.Run("cross-family int32 to float32", func(t *testing.T) {
		t.Parallel()
		got := transpile(t, `package p
main : proc() = {
	x : int32 = 42
	y := @cast<float32>(x)
	if y? {
		@print(y)
	}
}`)
		mustContain(t, got, "Float32frombits")
		mustContain(t, got, "Set: true")
	})

	t.Run("bool to uint8", func(t *testing.T) {
		t.Parallel()
		got := transpile(t, `package p
main : proc() = {
	x := true
	y := @cast<uint8>(x)
	if y? {
		@print(y)
	}
}`)
		mustContain(t, got, "builtin.If[uint8]")
		mustContain(t, got, "cog.Option[uint8]")
	})

	t.Run("float16 to uint32", func(t *testing.T) {
		t.Parallel()
		got := transpile(t, `package p
main : proc() = {
	x : float16 = 1.5
	y := @cast<uint32>(x)
	if y? {
		@print(y)
	}
}`)
		mustContain(t, got, ".Bits()")
		mustContain(t, got, "uint32(")
		mustContain(t, got, "Set: true")
	})

	t.Run("uint64 to int128", func(t *testing.T) {
		t.Parallel()
		got := transpile(t, `package p
main : proc() = {
	x : uint64 = 42
	y := @cast<int128>(x)
	if y? {
		@print(y)
	}
}`)
		mustContain(t, got, "cog.Uint128ToInt128")
		mustContain(t, got, "cog.Uint128From64")
		mustContain(t, got, "Set: true")
	})

	t.Run("direct uint16 to int16", func(t *testing.T) {
		t.Parallel()
		got := transpile(t, `package p
main : proc() = {
	x : uint16 = 1
	y := @cast<int16>(x)
	if y? {
		@print(y)
	}
}`)
		mustContain(t, got, "cog.Option[int16]{Value: int16(x), Set: true}")
	})

	t.Run("uint8 to bool", func(t *testing.T) {
		t.Parallel()
		got := transpile(t, `package p
main : proc() = {
	x : uint8 = 1
	y := @cast<bool>(x)
	if y? {
		@print(y)
	}
}`)
		mustContain(t, got, "!= 0")
		mustContain(t, got, "cog.Option[bool]")
	})

	t.Run("widening uint8 to uint16 returns some", func(t *testing.T) {
		t.Parallel()
		got := transpile(t, `package p
main : proc() = {
	x : uint8 = 1
	y := @cast<uint16>(x)
	if y? {
		@print(y)
	}
}`)
		mustContain(t, got, "cog.Option[uint16]{Value: uint16(x), Set: true}")
	})

	t.Run("widening int8 to int16 returns some", func(t *testing.T) {
		t.Parallel()
		got := transpile(t, `package p
main : proc() = {
	x : int8 = 1
	y := @cast<int16>(x)
	if y? {
		@print(y)
	}
}`)
		mustContain(t, got, "cog.Option[int16]{Value: int16(x), Set: true}")
	})

	t.Run("narrowing int64 to int32 returns none", func(t *testing.T) {
		t.Parallel()
		got := transpile(t, `package p
main : proc() = {
	x : int64 = 1
	y := @cast<int32>(x)
	if y? {
		@print(y)
	}
}`)
		mustContain(t, got, "cog.Option[int32]{Set: false}")
	})
}

func TestConvertBuiltinAs(t *testing.T) {
	t.Parallel()

	t.Run("int64 to int8 widening", func(t *testing.T) {
		t.Parallel()
		got := transpile(t, `package p
main : proc() = {
	x : int8 = 1
	y := @as<int64>(x)
	_ = y
}`)
		mustContain(t, got, "int64(x)")
	})

	t.Run("int8 to int64 narrowing with overflow check", func(t *testing.T) {
		t.Parallel()
		got := transpile(t, `package p
main : proc() = {
	x : int64 = 1
	y := @as<int8>(x)
	_ = y
}`)
		mustContain(t, got, "int8(x)")
		mustContain(t, got, "!=")
	})

	t.Run("uint64 to uint8 narrowing", func(t *testing.T) {
		t.Parallel()
		got := transpile(t, `package p
main : proc() = {
	x : uint64 = 1
	y := @as<uint8>(x)
	_ = y
}`)
		mustContain(t, got, "uint8(x)")
		mustContain(t, got, "!=")
	})

	t.Run("int64 to uint8 cross-family narrowing", func(t *testing.T) {
		t.Parallel()
		got := transpile(t, `package p
main : proc() = {
	x : int64 = 1
	y := @as<uint8>(x)
	_ = y
}`)
		mustContain(t, got, "uint8(x)")
		mustContain(t, got, "!=")
	})

	t.Run("int32 to uint8 cross-family narrowing", func(t *testing.T) {
		t.Parallel()
		got := transpile(t, `package p
main : proc() = {
	x : int32 = 1
	y := @as<uint8>(x)
	_ = y
}`)
		mustContain(t, got, "uint8(x)")
		mustContain(t, got, "!=")
	})

	t.Run("bool to utf8", func(t *testing.T) {
		t.Parallel()
		got := transpile(t, `package p
main : proc() = {
	x := true
	y := @as<utf8>(x)
	_ = y
}`)
		mustContain(t, got, `"true"`)
	})

	t.Run("int32 to utf8", func(t *testing.T) {
		t.Parallel()
		got := transpile(t, `package p
main : proc() = {
	x : int32 = 42
	y := @as<utf8>(x)
	_ = y
}`)
		mustContain(t, got, "strconv.FormatInt")
	})

	t.Run("utf8 to ascii", func(t *testing.T) {
		t.Parallel()
		got := transpile(t, `package p
main : proc() = {
	x : utf8 = "hello"
	y := @as<ascii>(x)
	_ = y
}`)
		mustContain(t, got, "cog.ASCII")
	})

	t.Run("ascii to utf8", func(t *testing.T) {
		t.Parallel()
		got := transpile(t, `package p
main : proc() = {
	x : utf8 = "hello"
	y := @as<utf8>(x)
	_ = y
}`)
		mustNotContain(t, got, "strconv")
	})

	t.Run("float64 to int32", func(t *testing.T) {
		t.Parallel()
		got := transpile(t, `package p
main : proc() = {
	x : float64 = 1.5
	y := @as<int32>(x)
	_ = y
}`)
		mustContain(t, got, "math.IsNaN")
	})

	t.Run("float32 to int32", func(t *testing.T) {
		t.Parallel()
		got := transpile(t, `package p
main : proc() = {
	x : float32 = 1.5
	y := @as<int32>(x)
	_ = y
}`)
		mustContain(t, got, "float64(x)")
		mustContain(t, got, "math.IsNaN")
		mustContain(t, got, "math.Trunc")
	})

	t.Run("bool to int8", func(t *testing.T) {
		t.Parallel()
		got := transpile(t, `package p
main : proc() = {
	x := true
	y := @as<int8>(x)
	_ = y
}`)
		mustContain(t, got, "builtin.If[int8]")
	})

	t.Run("int32 to bool", func(t *testing.T) {
		t.Parallel()
		got := transpile(t, `package p
main : proc() = {
	x : int32 = 42
	y := @as<bool>(x)
	_ = y
}`)
		mustContain(t, got, "!= 0")
	})

	t.Run("utf8 to int32", func(t *testing.T) {
		t.Parallel()
		got := transpile(t, `package p
main : proc() = {
	x : utf8 = "42"
	y := @as<int32>(x)
	_ = y
}`)
		mustContain(t, got, "strconv.ParseInt")
	})

	t.Run("utf8 to bool", func(t *testing.T) {
		t.Parallel()
		got := transpile(t, `package p
main : proc() = {
	x : utf8 = "true"
	y := @as<bool>(x)
	_ = y
}`)
		mustContain(t, got, "strconv.ParseBool")
	})

	t.Run("utf8 to float32", func(t *testing.T) {
		t.Parallel()
		got := transpile(t, `package p
main : proc() = {
	x : utf8 = "3.14"
	y := @as<float32>(x)
	_ = y
}`)
		mustContain(t, got, "strconv.ParseFloat")
	})

	t.Run("float32 to utf8", func(t *testing.T) {
		t.Parallel()
		got := transpile(t, `package p
main : proc() = {
	x : float32 = 3.14
	y := @as<utf8>(x)
	_ = y
}`)
		mustContain(t, got, "strconv.FormatFloat")
	})

	t.Run("same type passthrough", func(t *testing.T) {
		t.Parallel()
		got := transpile(t, `package p
main : proc() = {
	x : int32 = 42
	y := @as<int32>(x)
	_ = y
}`)
		mustNotContain(t, got, "int64")
		mustNotContain(t, got, "strconv")
	})

	t.Run("float16 to utf8", func(t *testing.T) {
		t.Parallel()
		got := transpile(t, `package p
main : proc() = {
	x : float16 = 1.5
	y := @as<utf8>(x)
	_ = y
}`)
		mustContain(t, got, ".String()")
	})

	t.Run("utf8 to float16", func(t *testing.T) {
		t.Parallel()
		got := transpile(t, `package p
main : proc() = {
	x : utf8 = "1.5"
	y := @as<float16>(x)
	_ = y
}`)
		mustContain(t, got, "strconv.ParseFloat")
		mustContain(t, got, "cog.Float16Fromfloat32")
	})

	t.Run("int128 to utf8", func(t *testing.T) {
		t.Parallel()
		got := transpile(t, `package p
main : proc() = {
	x : int128 = 42
	y := @as<utf8>(x)
	_ = y
}`)
		mustContain(t, got, ".String()")
	})

	t.Run("uint128 to utf8", func(t *testing.T) {
		t.Parallel()
		got := transpile(t, `package p
main : proc() = {
	x : uint128 = 42
	y := @as<utf8>(x)
	_ = y
}`)
		mustContain(t, got, ".String()")
	})

	t.Run("utf8 to int128", func(t *testing.T) {
		t.Parallel()
		got := transpile(t, `package p
main : proc() = {
	x : utf8 = "42"
	y := @as<int128>(x)
	_ = y
}`)
		mustContain(t, got, "cog.Int128FromString")
	})

	t.Run("utf8 to uint128", func(t *testing.T) {
		t.Parallel()
		got := transpile(t, `package p
main : proc() = {
	x : utf8 = "42"
	y := @as<uint128>(x)
	_ = y
}`)
		mustContain(t, got, "cog.Uint128FromString")
	})

	t.Run("complex32 to complex64", func(t *testing.T) {
		t.Parallel()
		got := transpile(t, `package p
main : proc() = {
	x : complex32 = {1.0, 2.0}
	y := @as<complex64>(x)
	_ = y
}`)
		mustContain(t, got, ".Complex64()")
	})

	t.Run("complex32 to float64", func(t *testing.T) {
		t.Parallel()
		got := transpile(t, `package p
main : proc() = {
	x : complex32 = {1.0, 2.0}
	y := @as<float64>(x)
	_ = y
}`)
		mustContain(t, got, ".Complex64()")
		mustContain(t, got, "real(")
	})

	t.Run("complex32 to utf8", func(t *testing.T) {
		t.Parallel()
		got := transpile(t, `package p
main : proc() = {
	x : complex32 = {1.0, 2.0}
	y := @as<utf8>(x)
	_ = y
}`)
		mustContain(t, got, "Sprintf")
		mustContain(t, got, ".Complex64()")
	})
}
