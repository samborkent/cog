package cog

type (
	ASCII             []byte
	Set[T comparable] map[T]struct{}
)

type Option[T any] struct {
	Value T
	Set   bool
}

type String interface {
	~[]byte | ~string
}
