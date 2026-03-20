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
