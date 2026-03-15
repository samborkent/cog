package cog

import "arena"

//go:fix inline
func NewArena() *arena.Arena {
	return arena.NewArena()
}

//go:fix inline
func New[T any](a *arena.Arena) *T {
	return arena.New[T](a)
}

//go:fix inline
func MakeSlice[T any](a *arena.Arena, len, cap int) []T {
	return arena.MakeSlice[T](a, len, cap)
}

//go:fix inline
func Clone[T any](s T) T {
	return arena.Clone(s)
}
