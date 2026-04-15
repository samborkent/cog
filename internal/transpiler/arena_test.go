package transpiler_test

import "testing"

func TestArenaInjection(t *testing.T) {
	t.Parallel()

	t.Run("literal-length slice no arena", func(t *testing.T) {
		t.Parallel()
		got := transpile(t, `package p
main : proc() = {
	xs := @slice<int64>(3)
	@print(xs)
}`)
		mustNotContain(t, got, "cog.NewArena()")
		mustContain(t, got, "make([]int64,")
	})

	t.Run("ref only no arena", func(t *testing.T) {
		t.Parallel()
		got := transpile(t, `package p
main : proc() = {
	ref := @ref<int64>()
	@print(ref)
}`)
		mustNotContain(t, got, "cog.NewArena()")
		mustContain(t, got, "new(int64)")
	})

	t.Run("single var-len slice no arena", func(t *testing.T) {
		t.Parallel()
		got := transpile(t, `package p
main : proc() = {
	n := 10
	xs := @slice<int64>(n)
	@print(xs)
}`)
		mustNotContain(t, got, "cog.NewArena()")
		mustContain(t, got, "make([]int64,")
	})

	t.Run("two var-len slices gets arena", func(t *testing.T) {
		t.Parallel()
		got := transpile(t, `package p
main : proc() = {
	n := 10
	xs := @slice<int64>(n)
	ys := @slice<int64>(n)
	@print(xs)
	@print(ys)
}`)
		mustContain(t, got, "cog.NewArena()")
		mustContain(t, got, "cog.MakeSlice[int64]")
		mustContain(t, got, "_arena.Free()")
	})

	t.Run("var-len slice plus ref no arena for ref", func(t *testing.T) {
		t.Parallel()
		got := transpile(t, `package p
main : proc() = {
	n := 10
	xs := @slice<int64>(n)
	ys := @slice<int64>(n)
	ref := @ref<int64>()
	@print(xs)
	@print(ys)
	@print(ref)
}`)
		mustContain(t, got, "cog.NewArena()")
		mustContain(t, got, "cog.MakeSlice[int64]")
		mustContain(t, got, "new(int64)")
		mustNotContain(t, got, "cog.New[int64]")
	})

	t.Run("var-len plus literal slice no arena for literal", func(t *testing.T) {
		t.Parallel()
		got := transpile(t, `package p
main : proc() = {
	n := 10
	xs := @slice<int64>(n)
	ys := @slice<int64>(n)
	zs := @slice<int64>(5)
	@print(xs)
	@print(ys)
	@print(zs)
}`)
		mustContain(t, got, "cog.NewArena()")
		mustContain(t, got, "cog.MakeSlice[int64]")
		mustContain(t, got, "make([]int64,")
	})

	t.Run("returned var-len slice no arena", func(t *testing.T) {
		t.Parallel()
		got := transpile(t, `package p
main : proc() = {}
getSlice : proc(n : int64) []int64 = {
	xs := @slice<int64>(n)
	return xs
}`)
		mustNotContain(t, got, "cog.MakeSlice")
		mustContain(t, got, "make([]int64,")
	})

	t.Run("no allocations no arena", func(t *testing.T) {
		t.Parallel()
		got := transpile(t, `package p
main : proc() = {
	x := 5
	@print(x)
}`)
		mustNotContain(t, got, "cog.NewArena()")
	})

	t.Run("map allocation no arena", func(t *testing.T) {
		t.Parallel()
		got := transpile(t, `package p
main : proc() = {
	m := @map<utf8, int64>()
	@print(m)
}`)
		mustNotContain(t, got, "cog.NewArena()")
		mustContain(t, got, "make(map[string]int64)")
	})

	t.Run("set allocation no arena", func(t *testing.T) {
		t.Parallel()
		got := transpile(t, `package p
main : proc() = {
	s := @set<int64>()
	@print(s)
}`)
		mustNotContain(t, got, "cog.NewArena()")
		mustContain(t, got, "make(cog.Set[int64])")
	})

	t.Run("one returned one non-returned var-len no arena", func(t *testing.T) {
		t.Parallel()
		got := transpile(t, `package p
main : proc() = {}
mix : proc(n : int64) []int64 = {
	xs := @slice<int64>(n)
	ys := @slice<int64>(n)
	@print(xs)
	return ys
}`)
		mustNotContain(t, got, "cog.NewArena()")
		mustContain(t, got, "make([]int64,")
	})

	t.Run("two non-returned plus one returned var-len gets arena", func(t *testing.T) {
		t.Parallel()
		got := transpile(t, `package p
main : proc() = {}
mix : proc(n : int64) []int64 = {
	xs := @slice<int64>(n)
	ys := @slice<int64>(n)
	zs := @slice<int64>(n)
	@print(xs)
	@print(ys)
	return zs
}`)
		mustContain(t, got, "cog.NewArena()")
		mustContain(t, got, "cog.MakeSlice[int64]")
		mustContain(t, got, "make([]int64,")
	})

	t.Run("func gets arena too", func(t *testing.T) {
		t.Parallel()
		got := transpile(t, `package p
main : proc() = {}
compute : func(n : int64) int64 = {
	xs := @slice<int64>(n)
	ys := @slice<int64>(n)
	@print(xs)
	@print(ys)
	return n
}`)
		mustContain(t, got, "cog.NewArena()")
		mustContain(t, got, "cog.MakeSlice[int64]")
	})
}
