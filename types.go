package cog

import (
	"bytes"
	"hash/maphash"
	"math"
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

// Cast helpers: bitwise type reinterpretation.

// Complex32Bits returns the big-endian bit representation of a Complex32 as uint32.
// Real occupies the high 16 bits, Imag the low 16 bits.
func Complex32Bits(c Complex32) uint32 {
	return uint32(c.Real.Bits())<<16 | uint32(c.Imag.Bits())
}

// Complex32FromBits constructs a Complex32 from a uint32 bit pattern (big-endian).
func Complex32FromBits(bits uint32) Complex32 {
	return Complex32{
		Real: f16.Frombits(uint16(bits >> 16)),
		Imag: f16.Frombits(uint16(bits)),
	}
}

// Complex64Bits returns the big-endian bit representation of a complex64 as uint64.
// Real occupies the high 32 bits, Imag the low 32 bits.
func Complex64Bits(c complex64) uint64 {
	return uint64(math.Float32bits(real(c)))<<32 | uint64(math.Float32bits(imag(c)))
}

// Complex64FromBits constructs a complex64 from a uint64 bit pattern (big-endian).
func Complex64FromBits(bits uint64) complex64 {
	return complex(math.Float32frombits(uint32(bits>>32)), math.Float32frombits(uint32(bits)))
}

// Complex128Bits returns the big-endian bit representation of a complex128 as Uint128.
// Real occupies the high 64 bits, Imag the low 64 bits.
func Complex128Bits(c complex128) Uint128 {
	return u128.New(math.Float64bits(imag(c)), math.Float64bits(real(c)))
}

// Complex128FromBits constructs a complex128 from a Uint128 bit pattern (big-endian).
func Complex128FromBits(bits Uint128) complex128 {
	return complex(math.Float64frombits(bits.Hi), math.Float64frombits(bits.Lo))
}

// Uint128ToInt128 reinterprets a Uint128 as an Int128 by preserving bits.
func Uint128ToInt128(v Uint128) Int128 {
	return wide.NewInt128(int64(v.Hi), v.Lo)
}

// Int128ToUint128 reinterprets an Int128 as a Uint128 by preserving bits.
func Int128ToUint128(v Int128) Uint128 {
	lo := v.Uint64()
	hi := v.RShiftN(64).Uint64()
	return u128.New(lo, hi)
}

// Float16Frombits converts a uint16 to a Float16.
func Float16Frombits(bits uint16) Float16 {
	return f16.Frombits(bits)
}
