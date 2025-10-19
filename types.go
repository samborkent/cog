package cog

type (
	ASCII             []byte
	UTF8              string
	Set[T comparable] map[T]struct{}
)

type Option[T any] struct {
	Value T
	Set   bool
}

type String interface {
	~[]byte | ~string
}
