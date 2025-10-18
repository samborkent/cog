package ast

import (
	"math/big"

	f16 "github.com/x448/float16"
	u128 "lukechampine.com/uint128"

	"github.com/samborkent/cog/internal/tokens"
	"github.com/samborkent/cog/internal/types"
)

var _ Expression = &Infix{}

type Infix struct {
	expression

	Operator    tokens.Token
	Left, Right Expression
}

func (e *Infix) EqualizeLiteralTypes() {
	if types.Equal(e.Left.Type(), e.Right.Type()) {
		return
	}

	// Handle default inferred literal types.
	switch e.Left.Type().Underlying().Kind() {
	case types.ASCII:
		if right, ok := e.Right.(*UTF8Literal); ok {
			e.Right = &ASCIILiteral{
				Token: right.Token,
				Value: ascii(right.Value),
			}

			return
		}
	case types.Float16:
		if right, ok := e.Right.(*Float64Literal); ok {
			e.Right = &Float16Literal{
				Token: right.Token,
				Value: f16.Fromfloat32(float32(right.Value)),
			}

			return
		}
	case types.Float32:
		if right, ok := e.Right.(*Float64Literal); ok {
			e.Right = &Float32Literal{
				Token: right.Token,
				Value: float32(right.Value),
			}

			return
		}
	case types.Int8:
		if right, ok := e.Right.(*Int64Literal); ok {
			// TODO: handle overflow
			e.Right = &Int8Literal{
				Token: right.Token,
				Value: int8(right.Value),
			}

			return
		}
	case types.Int16:
		if right, ok := e.Right.(*Int64Literal); ok {
			// TODO: handle overflow
			e.Right = &Int16Literal{
				Token: right.Token,
				Value: int16(right.Value),
			}

			return
		}
	case types.Int32:
		if right, ok := e.Right.(*Int64Literal); ok {
			// TODO: handle overflow
			e.Right = &Int32Literal{
				Token: right.Token,
				Value: int32(right.Value),
			}

			return
		}
	case types.Int128:
		if right, ok := e.Right.(*Int64Literal); ok {
			// TODO: handle overflow
			e.Right = &Int128Literal{
				Token: right.Token,
				Value: big.NewInt(right.Value),
			}

			return
		}
	case types.Uint8:
		if right, ok := e.Right.(*Int64Literal); ok {
			// TODO: handle overflow
			e.Right = &Uint8Literal{
				Token: right.Token,
				Value: uint8(right.Value),
			}

			return
		}
	case types.Uint16:
		if right, ok := e.Right.(*Int64Literal); ok {
			// TODO: handle overflow
			e.Right = &Uint16Literal{
				Token: right.Token,
				Value: uint16(right.Value),
			}

			return
		}
	case types.Uint32:
		if right, ok := e.Right.(*Int64Literal); ok {
			// TODO: handle overflow
			e.Right = &Uint32Literal{
				Token: right.Token,
				Value: uint32(right.Value),
			}

			return
		}
	case types.Uint64:
		if right, ok := e.Right.(*Int64Literal); ok {
			// TODO: handle overflow
			e.Right = &Uint64Literal{
				Token: right.Token,
				Value: uint64(right.Value),
			}

			return
		}
	case types.Uint128:
		if right, ok := e.Right.(*Int64Literal); ok {
			// TODO: handle overflow
			e.Right = &Uint128Literal{
				Token: right.Token,
				Value: u128.From64(uint64(right.Value)),
			}

			return
		}
	}

	e.Left = upgradeLiteralType(e.Left, e.Right)
	e.Right = upgradeLiteralType(e.Right, e.Left)
}

func (e *Infix) Pos() (uint32, uint16) {
	return e.Operator.Ln, e.Operator.Col
}

func (e *Infix) Hash() uint64 {
	return hash(e)
}

func (e *Infix) String() string {
	return "(" + e.Left.String() + " " + e.Operator.Type.String() + " " + e.Right.String() + ")"
}

func (e *Infix) Type() types.Type {
	if e.Left.Type() == nil {
		panic("infix with nil type detected")
	}

	// Return bool type for comparison operators
	switch e.Operator.Type {
	case tokens.And, tokens.Or, tokens.Xor,
		tokens.Equal, tokens.NotEqual,
		tokens.GT, tokens.GTEqual, tokens.LT, tokens.LTEqual:
		return types.Basics[types.Bool]
	}

	return e.Left.Type()
}

func upgradeLiteralType(expr Expression, ref Expression) Expression {
	refType := ref.Type().Underlying().Kind()

	if expr.Type().Underlying().Kind() == refType {
		return expr
	}

	switch e := expr.(type) {
	case *ASCIILiteral:
		if refType == types.UTF8 {
			// Upgrade ascii to utf8.
			return &UTF8Literal{
				Token: e.Token,
				Value: utf8(e.Value),
			}
		}
	case *Complex32Literal:
		switch refType {
		case types.Complex64:
			// Upgrade complex32 to complex64.
			return &Complex64Literal{
				Token: e.Token,
				Value: complex32to64(e.Value),
			}
		case types.Complex128:
			// Upgrade complex32 to complex128.
			return &Complex128Literal{
				Token: e.Token,
				Value: complex128(complex32to64(e.Value)),
			}
		}
	case *Complex64Literal:
		if refType == types.Complex128 {
			// Upgrade complex32 to complex128.
			return &Complex128Literal{
				Token: e.Token,
				Value: complex128(e.Value),
			}
		}
	case *Float16Literal:
		switch refType {
		case types.Float32:
			// Upgrade float16 to float32.
			return &Float32Literal{
				Token: e.Token,
				Value: e.Value.Float32(),
			}
		case types.Float64:
			// Upgrade float16 to float64.
			return &Float64Literal{
				Token: e.Token,
				Value: float64(e.Value.Float32()),
			}
		}
	case *Float32Literal:
		if refType == types.Float64 {
			// Upgrade float32 to float64.
			return &Float64Literal{
				Token: e.Token,
				Value: float64(e.Value),
			}
		}
	case *Int8Literal:
		switch refType {
		case types.Int16:
			// Upgrade int8 to int16.
			return &Int16Literal{
				Token: e.Token,
				Value: int16(e.Value),
			}
		case types.Int32:
			// Upgrade int8 to int32.
			return &Int32Literal{
				Token: e.Token,
				Value: int32(e.Value),
			}
		case types.Int64:
			// Upgrade int8 to int64.
			return &Int64Literal{
				Token: e.Token,
				Value: int64(e.Value),
			}
		case types.Int128:
			// Upgrade int8 to int128.
			return &Int128Literal{
				Token: e.Token,
				Value: big.NewInt(int64(e.Value)),
			}
		case types.Float16:
			// Upgrade int8 to float16.
			return &Float16Literal{
				Token: e.Token,
				Value: f16.Fromfloat32(float32(e.Value)),
			}
		case types.Float32:
			// Upgrade int8 to float32.
			return &Float32Literal{
				Token: e.Token,
				Value: float32(e.Value),
			}
		case types.Float64:
			// Upgrade int8 to float64.
			return &Float64Literal{
				Token: e.Token,
				Value: float64(e.Value),
			}
		}
	case *Int16Literal:
		switch refType {
		case types.Int32:
			// Upgrade int16 to int32.
			return &Int32Literal{
				Token: e.Token,
				Value: int32(e.Value),
			}
		case types.Int64:
			// Upgrade int16 to int64.
			return &Int64Literal{
				Token: e.Token,
				Value: int64(e.Value),
			}
		case types.Int128:
			// Upgrade int16 to int128.
			return &Int128Literal{
				Token: e.Token,
				Value: big.NewInt(int64(e.Value)),
			}
		case types.Float16:
			// Upgrade int16 to float16.
			return &Float16Literal{
				Token: e.Token,
				Value: f16.Fromfloat32(float32(e.Value)),
			}
		case types.Float32:
			// Upgrade int16 to float32.
			return &Float32Literal{
				Token: e.Token,
				Value: float32(e.Value),
			}
		case types.Float64:
			// Upgrade int16 to float64.
			return &Float64Literal{
				Token: e.Token,
				Value: float64(e.Value),
			}
		}
	case *Int32Literal:
		switch refType {
		case types.Int64:
			// Upgrade int32 to int64.
			return &Int64Literal{
				Token: e.Token,
				Value: int64(e.Value),
			}
		case types.Int128:
			// Upgrade int32 to int128.
			return &Int128Literal{
				Token: e.Token,
				Value: big.NewInt(int64(e.Value)),
			}
		case types.Float32:
			// Upgrade int32 to float32.
			return &Float32Literal{
				Token: e.Token,
				Value: float32(e.Value),
			}
		case types.Float64:
			// Upgrade int32 to float64.
			return &Float64Literal{
				Token: e.Token,
				Value: float64(e.Value),
			}
		}
	case *Int64Literal:
		switch refType {
		case types.Int128:
			// Upgrade int64 to int128.
			return &Int128Literal{
				Token: e.Token,
				Value: big.NewInt(int64(e.Value)),
			}
		case types.Float64:
			// Upgrade int64 to float64.
			return &Float64Literal{
				Token: e.Token,
				Value: float64(e.Value),
			}
		}
	case *Uint8Literal:
		switch refType {
		case types.Uint16:
			// Upgrade uint8 to uint16.
			return &Uint16Literal{
				Token: e.Token,
				Value: uint16(e.Value),
			}
		case types.Uint32:
			// Upgrade uint8 to uint32.
			return &Uint32Literal{
				Token: e.Token,
				Value: uint32(e.Value),
			}
		case types.Uint64:
			// Upgrade uint8 to uint64.
			return &Uint64Literal{
				Token: e.Token,
				Value: uint64(e.Value),
			}
		case types.Uint128:
			// Upgrade uint8 to uint128.
			return &Uint128Literal{
				Token: e.Token,
				Value: u128.From64(uint64(e.Value)),
			}
		case types.Int16:
			// Upgrade uint8 to int16.
			return &Int16Literal{
				Token: e.Token,
				Value: int16(e.Value),
			}
		case types.Int32:
			// Upgrade uint8 to int32.
			return &Int32Literal{
				Token: e.Token,
				Value: int32(e.Value),
			}
		case types.Int64:
			// Upgrade uint8 to int64.
			return &Int64Literal{
				Token: e.Token,
				Value: int64(e.Value),
			}
		case types.Int128:
			// Upgrade uint8 to int128.
			return &Int128Literal{
				Token: e.Token,
				Value: big.NewInt(int64(e.Value)),
			}
		case types.Float16:
			// Upgrade uint8 to float16.
			return &Float16Literal{
				Token: e.Token,
				Value: f16.Fromfloat32(float32(e.Value)),
			}
		case types.Float32:
			// Upgrade uint8 to float32.
			return &Float32Literal{
				Token: e.Token,
				Value: float32(e.Value),
			}
		case types.Float64:
			// Upgrade uint8 to float64.
			return &Float64Literal{
				Token: e.Token,
				Value: float64(e.Value),
			}
		}
	case *Uint16Literal:
		switch refType {
		case types.Uint32:
			// Upgrade uint16 to uint32.
			return &Uint32Literal{
				Token: e.Token,
				Value: uint32(e.Value),
			}
		case types.Uint64:
			// Upgrade uint16 to uint64.
			return &Uint64Literal{
				Token: e.Token,
				Value: uint64(e.Value),
			}
		case types.Uint128:
			// Upgrade uint16 to uint128.
			return &Uint128Literal{
				Token: e.Token,
				Value: u128.From64(uint64(e.Value)),
			}
		case types.Int32:
			// Upgrade uint16 to int32.
			return &Int32Literal{
				Token: e.Token,
				Value: int32(e.Value),
			}
		case types.Int64:
			// Upgrade uint16 to int64.
			return &Int64Literal{
				Token: e.Token,
				Value: int64(e.Value),
			}
		case types.Int128:
			// Upgrade uint16 to int128.
			return &Int128Literal{
				Token: e.Token,
				Value: big.NewInt(int64(e.Value)),
			}
		case types.Float16:
			// Upgrade uint16 to float16.
			return &Float16Literal{
				Token: e.Token,
				Value: f16.Fromfloat32(float32(e.Value)),
			}
		case types.Float32:
			// Upgrade uint16 to float32.
			return &Float32Literal{
				Token: e.Token,
				Value: float32(e.Value),
			}
		case types.Float64:
			// Upgrade uint16 to float64.
			return &Float64Literal{
				Token: e.Token,
				Value: float64(e.Value),
			}
		}
	case *Uint32Literal:
		switch refType {
		case types.Uint64:
			// Upgrade uint32 to uint64.
			return &Uint64Literal{
				Token: e.Token,
				Value: uint64(e.Value),
			}
		case types.Uint128:
			// Upgrade uint32 to uint128.
			return &Uint128Literal{
				Token: e.Token,
				Value: u128.From64(uint64(e.Value)),
			}
		case types.Int64:
			// Upgrade uint32 to int64.
			return &Int64Literal{
				Token: e.Token,
				Value: int64(e.Value),
			}
		case types.Int128:
			// Upgrade uint32 to int128.
			return &Int128Literal{
				Token: e.Token,
				Value: big.NewInt(int64(e.Value)),
			}
		case types.Float32:
			// Upgrade uint32 to float32.
			return &Float32Literal{
				Token: e.Token,
				Value: float32(e.Value),
			}
		case types.Float64:
			// Upgrade uint32 to float64.
			return &Float64Literal{
				Token: e.Token,
				Value: float64(e.Value),
			}
		}
	case *Uint64Literal:
		switch refType {
		case types.Uint128:
			// Upgrade uint32 to uint128.
			return &Uint128Literal{
				Token: e.Token,
				Value: u128.From64(uint64(e.Value)),
			}
		case types.Int128:
			// Upgrade uint32 to int128.
			return &Int128Literal{
				Token: e.Token,
				Value: big.NewInt(int64(e.Value)),
			}
		case types.Float64:
			// Upgrade uint32 to float64.
			return &Float64Literal{
				Token: e.Token,
				Value: float64(e.Value),
			}
		}
	}

	return expr
}
