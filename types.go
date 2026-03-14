package cog

import "hash/maphash"

type (
	ASCII             []byte
	ASCIIHash         uint64
	Set[T comparable] map[T]struct{}
)

var seed = maphash.MakeSeed()

// HashASCII
func HashASCII[Out ~uint64, In ~[]byte](in In) Out {
	return Out(maphash.Bytes(seed, in))
}

type Option[T any] struct {
	Value T
	Set   bool
}

type String interface {
	~[]byte | ~string
}
