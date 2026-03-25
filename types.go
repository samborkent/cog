package cog

import (
	"bytes"
	"hash/maphash"

	f16 "github.com/x448/float16"
)

type (
	ASCII             []byte
	ASCIIHash         uint64
	Float16           = f16.Float16
	Set[T comparable] map[T]struct{}
)

// Float16Fromfloat32 converts a float32 to a Float16.
func Float16Fromfloat32(f float32) Float16 {
	return f16.Fromfloat32(f)
}

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

// Complex32 represents a complex number with float16 real and imaginary parts.
type Complex32 struct {
	Real Float16
	Imag Float16
}

// Complex64 promotes the Complex32 to a native complex64.
func (c Complex32) Complex64() complex64 {
	return complex(c.Real.Float32(), c.Imag.Float32())
}

// Complex32FromComplex64 converts a complex64 to a Complex32.
func Complex32FromComplex64(c complex64) Complex32 {
	return Complex32{
		Real: Float16Fromfloat32(real(c)),
		Imag: Float16Fromfloat32(imag(c)),
	}
}
