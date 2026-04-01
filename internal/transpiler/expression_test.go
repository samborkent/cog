package transpiler_test

import "testing"

func TestConvertExpr(t *testing.T) {
	t.Parallel()

	t.Run("int_literal", func(t *testing.T) {
		t.Parallel()
		got := transpile(t, `package p
x := 42
main : proc() = {}`)
		mustContain(t, got, "42")
	})

	t.Run("string_literal", func(t *testing.T) {
		t.Parallel()
		got := transpile(t, `package p
s := "hello"
main : proc() = {}`)
		mustContain(t, got, `"hello"`)
	})

	t.Run("bool_literal", func(t *testing.T) {
		t.Parallel()
		got := transpile(t, `package p
b := true
main : proc() = {}`)
		mustContain(t, got, "true")
	})

	t.Run("negative_literal", func(t *testing.T) {
		t.Parallel()
		got := transpile(t, `package p
x := -1
main : proc() = {}`)
		mustContain(t, got, "-1")
	})

	t.Run("infix_add", func(t *testing.T) {
		t.Parallel()
		got := transpile(t, `package p
x := 1 + 2
main : proc() = {}`)
		mustContain(t, got, "1 + 2")
	})

	t.Run("infix_multiply", func(t *testing.T) {
		t.Parallel()
		got := transpile(t, `package p
x := 3 * 4
main : proc() = {}`)
		mustContain(t, got, "3 * 4")
	})

	t.Run("go_call", func(t *testing.T) {
		t.Parallel()
		got := transpile(t, `package p
goimport (
	"strings"
)
main : proc() = {
	x := @go.strings.ToUpper("hello")
	@print(x)
}`)
		mustContain(t, got, "strings.ToUpper")
	})

	t.Run("selector", func(t *testing.T) {
		t.Parallel()
		got := transpile(t, `package p
Point ~ struct {
	x : int32
	y : int32
}
main : proc() = {
	p : Point = {
		x = 1,
		y = 2,
	}
	@print(p.x)
}`)
		mustContain(t, got, "p.x")
	})

	t.Run("result_question_as_expression", func(t *testing.T) {
		t.Parallel()
		got := transpile(t, `package p
MyErr ~ error { Fail }
main : proc() = {
	var r : int64 ! MyErr = 1
	ok := r?
	@print(ok)
}`)
		mustContain(t, got, "!r.IsError")
	})

	t.Run("dyn_read", func(t *testing.T) {
		t.Parallel()
		got := transpile(t, `package p
dyn val : utf8 = "default"
main : proc() = {
	@print(val)
}`)
		mustContain(t, got, "dyn.val")
	})

	t.Run("index_expression", func(t *testing.T) {
		t.Parallel()
		got := transpile(t, `package p
xs : []int64 = {1, 2, 3}
main : proc() = {
	x := xs[0]
	@print(x)
}`)
		mustContain(t, got, "xs[0]")
	})

	t.Run("float16_literal", func(t *testing.T) {
		t.Parallel()
		got := transpile(t, `package p
x : float16 = 1.5
main : proc() = {}`)
		mustContain(t, got, "cog.Float16Fromfloat32")
		mustContain(t, got, "1.5")
	})

	t.Run("float16_add", func(t *testing.T) {
		t.Parallel()
		got := transpile(t, `package p
main : proc() = {
	a : float16 = 1.0
	b : float16 = 2.0
	c := a + b
	@print(c)
}`)
		mustContain(t, got, ".Float32()")
		mustContain(t, got, "cog.Float16Fromfloat32")
	})

	t.Run("float16_comparison", func(t *testing.T) {
		t.Parallel()
		got := transpile(t, `package p
main : proc() = {
	a : float16 = 1.0
	b : float16 = 2.0
	c := a < b
	@print(c)
}`)
		mustContain(t, got, ".Float32()")
		mustContain(t, got, "<")
	})

	t.Run("float16_negate", func(t *testing.T) {
		t.Parallel()
		got := transpile(t, `package p
main : proc() = {
	a : float16 = 1.0
	b := -a
	@print(b)
}`)
		mustContain(t, got, "cog.Float16Fromfloat32(-")
		mustContain(t, got, ".Float32()")
	})

	t.Run("complex32_literal", func(t *testing.T) {
		t.Parallel()
		got := transpile(t, `package p
c : complex32 = {1.0, 2.0}
main : proc() = {}`)
		mustContain(t, got, "cog.Complex32{")
		mustContain(t, got, "Real:")
		mustContain(t, got, "Imag:")
		mustContain(t, got, "cog.Float16Fromfloat32")
	})

	t.Run("complex32_add", func(t *testing.T) {
		t.Parallel()
		got := transpile(t, `package p
main : proc() = {
	a : complex32 = {1.0, 2.0}
	b : complex32 = {3.0, 4.0}
	c := a + b
	@print(c)
}`)
		mustContain(t, got, ".Complex64()")
		mustContain(t, got, "cog.Complex32FromComplex64")
	})

	t.Run("complex32_equality", func(t *testing.T) {
		t.Parallel()
		got := transpile(t, `package p
main : proc() = {
	a : complex32 = {1.0, 2.0}
	b : complex32 = {3.0, 4.0}
	c := a == b
	@print(c)
}`)
		mustContain(t, got, ".Complex64()")
		mustContain(t, got, "==")
	})

	t.Run("complex32_negate", func(t *testing.T) {
		t.Parallel()
		got := transpile(t, `package p
main : proc() = {
	a : complex32 = {1.0, 2.0}
	b := -a
	@print(b)
}`)
		mustContain(t, got, "cog.Complex32FromComplex64(-")
		mustContain(t, got, ".Complex64()")
	})

	t.Run("uint128_literal", func(t *testing.T) {
		t.Parallel()
		got := transpile(t, `package p
x : uint128 = 42
main : proc() = {}`)
		mustContain(t, got, "cog.Uint128FromString")
		mustContain(t, got, `"42"`)
	})

	t.Run("uint128_add", func(t *testing.T) {
		t.Parallel()
		got := transpile(t, `package p
main : proc() = {
	a : uint128 = 1
	b : uint128 = 2
	c := a + b
	@print(c)
}`)
		mustContain(t, got, ".Add(")
	})

	t.Run("uint128_equality", func(t *testing.T) {
		t.Parallel()
		got := transpile(t, `package p
main : proc() = {
	a : uint128 = 1
	b : uint128 = 2
	c := a == b
	@print(c)
}`)
		mustContain(t, got, ".Equals(")
	})

	t.Run("uint128_comparison", func(t *testing.T) {
		t.Parallel()
		got := transpile(t, `package p
main : proc() = {
	a : uint128 = 1
	b : uint128 = 2
	c := a < b
	@print(c)
}`)
		mustContain(t, got, ".Cmp(")
		mustContain(t, got, "< 0")
	})

	t.Run("int128_literal", func(t *testing.T) {
		t.Parallel()
		got := transpile(t, `package p
x : int128 = 42
main : proc() = {}`)
		mustContain(t, got, "cog.Int128FromString")
		mustContain(t, got, `"42"`)
	})

	t.Run("int128_add", func(t *testing.T) {
		t.Parallel()
		got := transpile(t, `package p
main : proc() = {
	a : int128 = 1
	b : int128 = 2
	c := a + b
	@print(c)
}`)
		mustContain(t, got, ".Add(")
	})

	t.Run("int128_equality", func(t *testing.T) {
		t.Parallel()
		got := transpile(t, `package p
main : proc() = {
	a : int128 = 1
	b : int128 = 2
	c := a == b
	@print(c)
}`)
		mustContain(t, got, ".Eq(")
	})

	t.Run("int128_comparison", func(t *testing.T) {
		t.Parallel()
		got := transpile(t, `package p
main : proc() = {
	a : int128 = 1
	b : int128 = 2
	c := a < b
	@print(c)
}`)
		mustContain(t, got, ".Cmp(")
		mustContain(t, got, "< 0")
	})

	t.Run("int128_negation", func(t *testing.T) {
		t.Parallel()
		got := transpile(t, `package p
main : proc() = {
	a : int128 = 1
	b := -a
	@print(b)
}`)
		mustContain(t, got, ".Neg()")
	})
}
