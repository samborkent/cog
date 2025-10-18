package cog

type (
	ASCII             []byte
	UTF8              string
	Set[T comparable] map[T]struct{}
)

type String interface {
	~[]byte | ~string
}
