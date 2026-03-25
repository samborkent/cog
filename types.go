package cog

import (
	"bytes"
	"hash/maphash"
	"math/big"

	"github.com/ryanavella/wide"
	f16 "github.com/x448/float16"
	u128 "lukechampine.com/uint128"
)

type (
	ASCII             []byte
	ASCIIHash         uint64
	Float16           = f16.Float16
	Int128            = wide.Int128
	Set[T comparable] map[T]struct{}
	Uint128           = u128.Uint128
)

// Float16Fromfloat32 converts a float32 to a Float16.
func Float16Fromfloat32(f float32) Float16 {
	return f16.Fromfloat32(f)
}

// Uint128From64 converts a uint64 to a Uint128.
func Uint128From64(v uint64) Uint128 {
	return u128.From64(v)
}

// Uint128FromString parses a decimal string into a Uint128.
func Uint128FromString(s string) Uint128 {
	v, _ := u128.FromString(s)
	return v
}

// Int128FromString parses a decimal string into an Int128.
func Int128FromString(s string) Int128 {
	v := new(big.Int)
	v.SetString(s, 10)
	return wide.Int128FromBigInt(v)
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
