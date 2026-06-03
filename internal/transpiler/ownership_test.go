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
a : dyn []int64 = []int64{1, 2, 3}
main : proc() = {
	b := a
	@print(b[0])
}`)
		mustContain(t, got, "cog.Copy(dyn.a)")
	})

	t.Run("dyn_read_primitive_no_copy", func(t *testing.T) {
		got := transpile(t, `package p
a : dyn int64 = 42
main : proc() = {
	b := a
	@print(b)
}`)
		mustNotContain(t, got, "cog.Copy")
	})

	t.Run("dyn_write_pointer_like_copies", func(t *testing.T) {
		got := transpile(t, `package p
a : dyn []int64 = []int64{1, 2, 3}
main : proc() = {
	a = []int64{4, 5, 6}
	b := a
	@print(b[0])
}`)
		mustContain(t, got, "cog.Copy")
	})

	t.Run("struct_field_assign", func(t *testing.T) {
		got := transpile(t, `package p
Buf ~ struct { data : []int64; label : utf8 }
main : proc() = {
	c : var Buf = Buf{data = @slice<int64>(3), label = "hello"}
	c.data = @slice<int64>(5)
	worker(c)
}
worker : proc(data : Buf) = {}`)
		mustContain(t, got, "c.data = make")
	})

	t.Run("struct_field_assign_exported", func(t *testing.T) {
		got := transpile(t, `package p
Buf ~ struct { export Data : []int64; label : utf8 }
main : proc() = {
	c : var Buf = Buf{Data = @slice<int64>(3), label = "hello"}
	c.Data = @slice<int64>(5)
	worker(c)
}
worker : proc(data : Buf) = {}`)
		mustContain(t, got, "c.Data = make")
	})

	t.Run("struct_field_read_after_assign", func(t *testing.T) {
		got := transpile(t, `package p
Buf ~ struct { data : []int64; label : utf8 }
main : proc() = {
	c : var Buf = Buf{data = @slice<int64>(3), label = "hello"}
	c.data = @slice<int64>(5)
	@print(c.data[0])
}`)
		mustContain(t, got, "c.data = make")
		mustContain(t, got, "c.data[0]")
	})
}