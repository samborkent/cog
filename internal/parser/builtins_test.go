package parser_test

import "testing"

func TestParseBuiltinPrint(t *testing.T) {
	t.Parallel()

	t.Run("valid", func(t *testing.T) {
		t.Parallel()
		f := parse(t, `package p
main : proc() = {
	@print("hello")
}`)
		if len(f.Statements) == 0 {
			t.Fatal("expected statements")
		}
	})
}

func TestParseBuiltinIf(t *testing.T) {
	t.Parallel()

	t.Run("valid", func(t *testing.T) {
		t.Parallel()
		f := parse(t, `package p
main : proc() = {
	x := @if(true, 1, 2)
	@print(x)
}`)
		if len(f.Statements) == 0 {
			t.Fatal("expected statements")
		}
	})
}

func TestParseBuiltinSlice(t *testing.T) {
	t.Parallel()

	t.Run("valid", func(t *testing.T) {
		t.Parallel()
		f := parse(t, `package p
main : proc() = {
	xs := @slice<int64>(3)
	@print(xs)
}`)
		if len(f.Statements) == 0 {
			t.Fatal("expected statements")
		}
	})

	t.Run("valid with capacity", func(t *testing.T) {
		t.Parallel()
		f := parse(t, `package p
main : proc() = {
	xs := @slice<int64>(3, 10)
	@print(xs)
}`)
		if len(f.Statements) == 0 {
			t.Fatal("expected statements")
		}
	})

	t.Run("valid with typed capacity", func(t *testing.T) {
		t.Parallel()
		f := parse(t, `package p
main : proc() = {
	xs := @slice<int64, uint8>(3, 10)
	@print(xs)
}`)
		if len(f.Statements) == 0 {
			t.Fatal("expected statements")
		}
	})
}

func TestParseBuiltinMap(t *testing.T) {
	t.Parallel()

	t.Run("valid", func(t *testing.T) {
		t.Parallel()
		f := parse(t, `package p
main : proc() = {
	m := @map<utf8, int64>()
	@print(m)
}`)
		if len(f.Statements) == 0 {
			t.Fatal("expected statements")
		}
	})
}

func TestParseBuiltinSet(t *testing.T) {
	t.Parallel()

	t.Run("valid", func(t *testing.T) {
		t.Parallel()
		f := parse(t, `package p
main : proc() = {
	s := @set<int64>()
	@print(s)
}`)
		if len(f.Statements) == 0 {
			t.Fatal("expected statements")
		}
	})
}

func TestParseBuiltinCast(t *testing.T) {
	t.Parallel()

	t.Run("int8 to int16", func(t *testing.T) {
		t.Parallel()
		f := parse(t, `package p
main : proc() = {
	x : int8 = 1
	y := @cast<int16>(x)
	@print(y)
}`)
		if len(f.Statements) == 0 {
			t.Fatal("expected statements")
		}
	})

	t.Run("int32 to float32", func(t *testing.T) {
		t.Parallel()
		f := parse(t, `package p
main : proc() = {
	x : int32 = 42
	y := @cast<float32>(x)
	@print(y)
}`)
		if len(f.Statements) == 0 {
			t.Fatal("expected statements")
		}
	})

	t.Run("bool to uint8", func(t *testing.T) {
		t.Parallel()
		f := parse(t, `package p
main : proc() = {
	x := true
	y := @cast<uint8>(x)
	@print(y)
}`)
		if len(f.Statements) == 0 {
			t.Fatal("expected statements")
		}
	})

	t.Run("float16 to uint32", func(t *testing.T) {
		t.Parallel()
		f := parse(t, `package p
main : proc() = {
	x : float16 = 1.5
	y := @cast<uint32>(x)
	@print(y)
}`)
		if len(f.Statements) == 0 {
			t.Fatal("expected statements")
		}
	})

	t.Run("uint64 to int128", func(t *testing.T) {
		t.Parallel()
		f := parse(t, `package p
main : proc() = {
	x : uint64 = 42
	y := @cast<int128>(x)
	@print(y)
}`)
		if len(f.Statements) == 0 {
			t.Fatal("expected statements")
		}
	})

	t.Run("with explicit source type", func(t *testing.T) {
		t.Parallel()
		f := parse(t, `package p
main : proc() = {
	x : int8 = 1
	y := @cast<int16, int8>(x)
	@print(y)
}`)
		if len(f.Statements) == 0 {
			t.Fatal("expected statements")
		}
	})

	t.Run("literal inferred from source type arg", func(t *testing.T) {
		t.Parallel()
		f := parse(t, `package p
main : proc() = {
	y := @cast<int16, int8>(1)
	@print(y)
}`)
		if len(f.Statements) == 0 {
			t.Fatal("expected statements")
		}
	})

	t.Run("same type rejected", func(t *testing.T) {
		t.Parallel()
		parseShouldError(t, `package p
main : proc() = {
	x : int32 = 1
	y := @cast<int32>(x)
	@print(y)
}`)
	})

	t.Run("narrowing rejected", func(t *testing.T) {
		t.Parallel()
		parseShouldError(t, `package p
main : proc() = {
	x : int64 = 1
	y := @cast<int32>(x)
	@print(y)
}`)
	})

	t.Run("wrong source type arg rejected", func(t *testing.T) {
		t.Parallel()
		parseShouldError(t, `package p
main : proc() = {
	x : int8 = 1
	y := @cast<int16, uint8>(x)
	@print(y)
}`)
	})

	t.Run("ascii to utf8", func(t *testing.T) {
		t.Parallel()
		f := parse(t, `package p
main : proc() = {
	x : ascii = "hello"
	y := @cast<utf8>(x)
	@print(y)
}`)
		if len(f.Statements) == 0 {
			t.Fatal("expected statements")
		}
	})

	t.Run("utf8 to int32 rejected", func(t *testing.T) {
		t.Parallel()
		parseShouldError(t, `package p
main : proc() = {
	x : utf8 = "hello"
	y := @cast<int32>(x)
	@print(y)
}`)
	})

	t.Run("int32 to ascii rejected", func(t *testing.T) {
		t.Parallel()
		parseShouldError(t, `package p
main : proc() = {
	x : int32 = 42
	y := @cast<ascii>(x)
	@print(y)
}`)
	})

	t.Run("utf8 to ascii rejected", func(t *testing.T) {
		t.Parallel()
		parseShouldError(t, `package p
main : proc() = {
	x : utf8 = "hello"
	y := @cast<ascii>(x)
	@print(y)
}`)
	})
}
