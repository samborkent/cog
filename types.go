package cog

import (
	"bytes"
	"hash/maphash"
)

type (
	ASCII             []byte
	ASCIIHash         uint64
	Set[T comparable] map[T]struct{}
)

func (a ASCII) Equal(b ASCII) bool {
	return bytes.Equal(a, b)
}

var seed = maphash.MakeSeed()

// HashASCII hashes an ASCII value to a uint64 to use for map and set keys.
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

type Index interface {
	~uint8 | ~uint16 | ~uint32 | ~uint64
}
