package transpiler_test

import (
	"testing"
)

func TestOwnershipTranspile(t *testing.T) {
	t.Run("var_from_immutable_pointer_like_copies", func(t *testing.T) {
		got := transpile(t, `package p
a : []int64 = []int64{1, 2, 3}
main : proc() = {
	b : var []int64 = a
	@print(b[0])
}`)
		mustContain(t, got, "cog.Copy(a)")
	})

	t.Run("var_from_immutable_primitive_no_copy", func(t *testing.T) {
		got := transpile(t, `package p
main : proc() = {
	a : int64 = 42
	b : var int64 = a
	@print(b)
}`)
		mustNotContain(t, got, "cog.Copy")
	})

	t.Run("var_to_var_pointer_like_no_copy", func(t *testing.T) {
		got := transpile(t, `package p
main : proc() = {
	a : var []int64 = []int64{1, 2, 3}
	b : var []int64 = a
	@print(b[0])
}`)
		mustNotContain(t, got, "cog.Copy")
	})

	t.Run("dyn_read_pointer_like_copies", func(t *testing.T) {
		got := transpile(t, `package p
dyn a : []int64 = []int64{1, 2, 3}
main : proc() = {
	b := a
	@print(b[0])
}`)
		mustContain(t, got, "cog.Copy(dyn.a)")
	})

	t.Run("dyn_read_primitive_no_copy", func(t *testing.T) {
		got := transpile(t, `package p
dyn a : int64 = 42
main : proc() = {
	b := a
	@print(b)
}`)
		mustNotContain(t, got, "cog.Copy")
	})

	t.Run("dyn_write_pointer_like_copies", func(t *testing.T) {
		got := transpile(t, `package p
dyn a : []int64 = []int64{1, 2, 3}
main : proc() = {
	a = []int64{4, 5, 6}
	b := a
	@print(b[0])
}`)
		mustContain(t, got, "cog.Copy")
	})
}