package builtin

import "github.com/samborkent/cog"

func Map[K comparable, V any, I cog.Index](capacity ...I) map[K]V {
	if len(capacity) == 0 {
		return make(map[K]V)
	}

	if len(capacity) > 1 {
		panic("@map: wrong number of arguments")
	}

	return make(map[K]V, capacity[0])
}

//go:fix inline
func Ptr[T any]() *T {
	return new(T)
}

func Set[K comparable, I cog.Index](capacity ...I) cog.Set[K] {
	if len(capacity) == 0 {
		return make(cog.Set[K])
	}

	if len(capacity) > 1 {
		panic("@set: wrong number of arguments")
	}

	return make(cog.Set[K], capacity[0])
}

func Slice[T any, I cog.Index](length I, capacity ...I) []T {
	if len(capacity) == 0 {
		return make([]T, length)
	}

	if len(capacity) > 1 {
		panic("@slice: wrong number of arguments")
	}

	return make([]T, length, capacity[0])
}
