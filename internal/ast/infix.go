package ast

import (
	"math"
	"strings"

	"github.com/ryanavella/wide"
	f16 "github.com/x448/float16"
	u128 "lukechampine.com/uint128"

	"github.com/samborkent/cog/internal/tokens"
	"github.com/samborkent/cog/internal/types"
)

var _ Expr = &Infix{}

type Infix struct {
	Operator    tokens.Token
	Left, Right ExprValue
}

func (e *Infix) EqualizeLiteralTypes() {
	if e.Left == e.Right || types.Equal(e.Left.expr.Type(), e.Right.expr.Type()) {
		return
	}

	// Handle default inferred literal types.
	switch e.Left.TypeKind {
	case types.ASCII:
		if e.Right.NodeKind == KindUTF8Literal {
			right := e.Right.expr.(*UTF8Literal)
			e.Right.expr = &ASCIILiteral{
				Token: right.Token,
				Value: ascii(right.Value),
			}

			return
		}
	case types.Float16:
		if e.Right.NodeKind == KindFloat64Literal {
			right := e.Right.expr.(*Float64Literal)
			e.Right.expr = &Float16Literal{
				Token: right.Token,
				Value: f16.Fromfloat32(float32(right.Value)),
			}

			return
		}
	case types.Float32:
		if e.Right.NodeKind == KindFloat64Literal {
			right := e.Right.expr.(*Float64Literal)
			e.Right.expr = &Float32Literal{
				Token: right.Token,
				Value: float32(right.Value),
			}

			return
		}
	case types.Int8:
		if e.Right.NodeKind == KindInt64Literal {
			right := e.Right.expr.(*Int64Literal)
			if right.Value >= math.MinInt8 && right.Value <= math.MaxInt8 {
				e.Right.expr = &Int8Literal{
					Token: right.Token,
					Value: int8(right.Value),
				}

				return
			}
		}
	case types.Int16:
		if e.Right.NodeKind == KindInt64Literal {
			right := e.Right.expr.(*Int64Literal)
			if right.Value >= math.MinInt16 && right.Value <= math.MaxInt16 {
				e.Right.expr = &Int16Literal{
					Token: right.Token,
					Value: int16(right.Value),
				}

				return
			}
		}
	case types.Int32:
		if e.Right.NodeKind == KindInt64Literal {
			right := e.Right.expr.(*Int64Literal)
			if right.Value >= math.MinInt32 && right.Value <= math.MaxInt32 {
				e.Right.expr = &Int32Literal{
					Token: right.Token,
					Value: int32(right.Value),
				}

				return
			}
		}
	case types.Int128:
		if e.Right.NodeKind == KindInt64Literal {
			right := e.Right.expr.(*Int64Literal)
			e.Right.expr = &Int128Literal{
				Token: right.Token,
				Value: wide.Int128FromInt64(right.Value),
			}

			return
		}
	case types.Uint8:
		if e.Right.NodeKind == KindInt64Literal {
			right := e.Right.expr.(*Int64Literal)
			if right.Value >= 0 && right.Value <= math.MaxUint8 {
				e.Right.expr = &Uint8Literal{
					Token: right.Token,
					Value: uint8(right.Value),
				}

				return
			}
		}
	case types.Uint16:
		if e.Right.NodeKind == KindInt64Literal {
			right := e.Right.expr.(*Int64Literal)
			if right.Value >= 0 && right.Value <= math.MaxUint16 {
				e.Right.expr = &Uint16Literal{
					Token: right.Token,
					Value: uint16(right.Value),
				}

				return
			}
		}
	case types.Uint32:
		if e.Right.NodeKind == KindInt64Literal {
			right := e.Right.expr.(*Int64Literal)
			if right.Value >= 0 && right.Value <= math.MaxUint32 {
				e.Right.expr = &Uint32Literal{
					Token: right.Token,
					Value: uint32(right.Value),
				}

				return
			}
		}
	case types.Uint64:
		if e.Right.NodeKind == KindInt64Literal {
			right := e.Right.expr.(*Int64Literal)
			if right.Value >= 0 {
				e.Right.expr = &Uint64Literal{
					Token: right.Token,
					Value: uint64(right.Value),
				}

				return
			}
		}
	case types.Uint128:
		if e.Right.NodeKind == KindInt64Literal {
			right := e.Right.expr.(*Int64Literal)
			if right.Value >= 0 {
				e.Right.expr = &Uint128Literal{
					Token: right.Token,
					Value: u128.From64(uint64(right.Value)),
				}

				return
			}
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

func (e *Infix) stringTo(out *strings.Builder) {
	_ = out.WriteByte('(')
	e.Left.expr.stringTo(out)
	_ = out.WriteByte(' ')
	_, _ = out.WriteString(e.Operator.Type.String())
	_ = out.WriteByte(' ')
	e.Right.expr.stringTo(out)
	_ = out.WriteByte(')')
}

func (e *Infix) String() string {
	var out strings.Builder
	e.stringTo(&out)

	return out.String()
}

func (e *Infix) Type() types.Type {
	// Return bool type for comparison operators
	switch e.Operator.Type {
	case tokens.And, tokens.Or,
		tokens.Equal, tokens.NotEqual,
		tokens.GT, tokens.GTEqual, tokens.LT, tokens.LTEqual:
		return types.Basics[types.Bool]
	}

	return e.Left.expr.Type()
}

func upgradeLiteralType(expr, ref ExprValue) ExprValue {
	refType := ref.TypeKind

	if expr.TypeKind == refType {
		return expr
	}

	switch expr.NodeKind {
	case KindASCIILiteral:
		if refType == types.UTF8 {
			e := expr.expr.(*ASCIILiteral)

			// Upgrade ascii to utf8.
			return ExprValue{
				NodeKind: KindUTF8Literal,
				TypeKind: types.UTF8,
				expr: &UTF8Literal{
					Token: e.Token,
					Value: utf8(e.Value),
				},
			}
		}
	case KindComplex32Literal:
		switch refType {
		case types.Complex64:
			// Upgrade complex32 to complex64.
			e := expr.expr.(*Complex32Literal)
			return ExprValue{
				NodeKind: KindComplex64Literal,
				TypeKind: types.Complex64,
				expr: &Complex64Literal{
					Token: e.Token,
					Value: Complex32To64(e.Value),
				},
			}
		case types.Complex128:
			// Upgrade complex32 to complex128.
			e := expr.expr.(*Complex32Literal)
			return ExprValue{
				NodeKind: KindComplex128Literal,
				TypeKind: types.Complex128,
				expr: &Complex128Literal{
					Token: e.Token,
					Value: complex128(Complex32To64(e.Value)),
				},
			}
		}
	case KindComplex64Literal:
		if refType == types.Complex128 {
			// Upgrade complex64 to complex128.
			e := expr.expr.(*Complex64Literal)
			return ExprValue{
				NodeKind: KindComplex128Literal,
				TypeKind: types.Complex128,
				expr: &Complex128Literal{
					Token: e.Token,
					Value: complex128(e.Value),
				},
			}
		}
	case KindFloat16Literal:
		switch refType {
		case types.Float32:
			// Upgrade float16 to float32.
			e := expr.expr.(*Float16Literal)
			return ExprValue{
				NodeKind: KindFloat32Literal,
				TypeKind: types.Float32,
				expr: &Float32Literal{
					Token: e.Token,
					Value: e.Value.Float32(),
				},
			}
		case types.Float64:
			// Upgrade float16 to float64.
			e := expr.expr.(*Float16Literal)
			return ExprValue{
				NodeKind: KindFloat64Literal,
				TypeKind: types.Float64,
				expr: &Float64Literal{
					Token: e.Token,
					Value: float64(e.Value.Float32()),
				},
			}
		}
	case KindFloat32Literal:
		if refType == types.Float64 {
			// Upgrade float32 to float64.
			e := expr.expr.(*Float32Literal)
			return ExprValue{
				NodeKind: KindFloat64Literal,
				TypeKind: types.Float64,
				expr: &Float64Literal{
					Token: e.Token,
					Value: float64(e.Value),
				},
			}
		}
	case KindInt8Literal:
		switch refType {
		case types.Int16:
			// Upgrade int8 to int16.
			e := expr.expr.(*Int8Literal)
			return ExprValue{
				NodeKind: KindInt16Literal,
				TypeKind: types.Int16,
				expr: &Int16Literal{
					Token: e.Token,
					Value: int16(e.Value),
				},
			}
		case types.Int32:
			// Upgrade int8 to int32.
			e := expr.expr.(*Int8Literal)
			return ExprValue{
				NodeKind: KindInt32Literal,
				TypeKind: types.Int32,
				expr: &Int32Literal{
					Token: e.Token,
					Value: int32(e.Value),
				},
			}
		case types.Int64:
			// Upgrade int8 to int64.
			e := expr.expr.(*Int8Literal)
			return ExprValue{
				NodeKind: KindInt64Literal,
				TypeKind: types.Int64,
				expr: &Int64Literal{
					Token: e.Token,
					Value: int64(e.Value),
				},
			}
		case types.Int128:
			// Upgrade int8 to int128.
			e := expr.expr.(*Int8Literal)
			return ExprValue{
				NodeKind: KindInt128Literal,
				TypeKind: types.Int128,
				expr: &Int128Literal{
					Token: e.Token,
					Value: wide.Int128FromInt64(int64(e.Value)),
				},
			}
		case types.Float16:
			// Upgrade int8 to float16.
			e := expr.expr.(*Int8Literal)
			return ExprValue{
				NodeKind: KindFloat16Literal,
				TypeKind: types.Float16,
				expr: &Float16Literal{
					Token: e.Token,
					Value: f16.Fromfloat32(float32(e.Value)),
				},
			}
		case types.Float32:
			// Upgrade int8 to float32.
			e := expr.expr.(*Int8Literal)
			return ExprValue{
				NodeKind: KindFloat32Literal,
				TypeKind: types.Float32,
				expr: &Float32Literal{
					Token: e.Token,
					Value: float32(e.Value),
				},
			}
		case types.Float64:
			// Upgrade int8 to float64.
			e := expr.expr.(*Int8Literal)
			return ExprValue{
				NodeKind: KindFloat64Literal,
				TypeKind: types.Float64,
				expr: &Float64Literal{
					Token: e.Token,
					Value: float64(e.Value),
				},
			}
		}
	case KindInt16Literal:
		switch refType {
		case types.Int32:
			// Upgrade int16 to int32.
			e := expr.expr.(*Int16Literal)
			return ExprValue{
				NodeKind: KindInt32Literal,
				TypeKind: types.Int32,
				expr: &Int32Literal{
					Token: e.Token,
					Value: int32(e.Value),
				},
			}
		case types.Int64:
			// Upgrade int16 to int64.
			e := expr.expr.(*Int16Literal)
			return ExprValue{
				NodeKind: KindInt64Literal,
				TypeKind: types.Int64,
				expr: &Int64Literal{
					Token: e.Token,
					Value: int64(e.Value),
				},
			}
		case types.Int128:
			// Upgrade int16 to int128.
			e := expr.expr.(*Int16Literal)
			return ExprValue{
				NodeKind: KindInt128Literal,
				TypeKind: types.Int128,
				expr: &Int128Literal{
					Token: e.Token,
					Value: wide.Int128FromInt64(int64(e.Value)),
				},
			}
		case types.Float16:
			// Upgrade int16 to float16.
			e := expr.expr.(*Int16Literal)
			return ExprValue{
				NodeKind: KindFloat16Literal,
				TypeKind: types.Float16,
				expr: &Float16Literal{
					Token: e.Token,
					Value: f16.Fromfloat32(float32(e.Value)),
				},
			}
		case types.Float32:
			// Upgrade int16 to float32.
			e := expr.expr.(*Int16Literal)
			return ExprValue{
				NodeKind: KindFloat32Literal,
				TypeKind: types.Float32,
				expr: &Float32Literal{
					Token: e.Token,
					Value: float32(e.Value),
				},
			}
		case types.Float64:
			// Upgrade int16 to float64.
			e := expr.expr.(*Int16Literal)
			return ExprValue{
				NodeKind: KindFloat64Literal,
				TypeKind: types.Float64,
				expr: &Float64Literal{
					Token: e.Token,
					Value: float64(e.Value),
				},
			}
		}
	case KindInt32Literal:
		switch refType {
		case types.Int64:
			// Upgrade int32 to int64.
			e := expr.expr.(*Int32Literal)
			return ExprValue{
				NodeKind: KindInt64Literal,
				TypeKind: types.Int64,
				expr: &Int64Literal{
					Token: e.Token,
					Value: int64(e.Value),
				},
			}
		case types.Int128:
			// Upgrade int32 to int128.
			e := expr.expr.(*Int32Literal)
			return ExprValue{
				NodeKind: KindInt128Literal,
				TypeKind: types.Int128,
				expr: &Int128Literal{
					Token: e.Token,
					Value: wide.Int128FromInt64(int64(e.Value)),
				},
			}
		case types.Float32:
			// Upgrade int32 to float32.
			e := expr.expr.(*Int32Literal)
			return ExprValue{
				NodeKind: KindFloat32Literal,
				TypeKind: types.Float32,
				expr: &Float32Literal{
					Token: e.Token,
					Value: float32(e.Value),
				},
			}
		case types.Float64:
			// Upgrade int32 to float64.
			e := expr.expr.(*Int32Literal)
			return ExprValue{
				NodeKind: KindFloat64Literal,
				TypeKind: types.Float64,
				expr: &Float64Literal{
					Token: e.Token,
					Value: float64(e.Value),
				},
			}
		}
	case KindInt64Literal:
		switch refType {
		case types.Int128:
			// Upgrade int64 to int128.
			e := expr.expr.(*Int64Literal)
			return ExprValue{
				NodeKind: KindInt128Literal,
				TypeKind: types.Int128,
				expr: &Int128Literal{
					Token: e.Token,
					Value: wide.Int128FromInt64(e.Value),
				},
			}
		case types.Float64:
			// Upgrade int64 to float64.
			e := expr.expr.(*Int64Literal)
			return ExprValue{
				NodeKind: KindFloat64Literal,
				TypeKind: types.Float64,
				expr: &Float64Literal{
					Token: e.Token,
					Value: float64(e.Value),
				},
			}
		}
	case KindUint8Literal:
		switch refType {
		case types.Uint16:
			// Upgrade uint8 to uint16.
			e := expr.expr.(*Uint8Literal)
			return ExprValue{
				NodeKind: KindUint16Literal,
				TypeKind: types.Uint16,
				expr: &Uint16Literal{
					Token: e.Token,
					Value: uint16(e.Value),
				},
			}
		case types.Uint32:
			// Upgrade uint8 to uint32.
			e := expr.expr.(*Uint8Literal)
			return ExprValue{
				NodeKind: KindUint32Literal,
				TypeKind: types.Uint32,
				expr: &Uint32Literal{
					Token: e.Token,
					Value: uint32(e.Value),
				},
			}
		case types.Uint64:
			// Upgrade uint8 to uint64.
			e := expr.expr.(*Uint8Literal)
			return ExprValue{
				NodeKind: KindUint64Literal,
				TypeKind: types.Uint64,
				expr: &Uint64Literal{
					Token: e.Token,
					Value: uint64(e.Value),
				},
			}
		case types.Uint128:
			// Upgrade uint8 to uint128.
			e := expr.expr.(*Uint8Literal)
			return ExprValue{
				NodeKind: KindUint128Literal,
				TypeKind: types.Uint128,
				expr: &Uint128Literal{
					Token: e.Token,
					Value: u128.From64(uint64(e.Value)),
				},
			}
		case types.Int16:
			// Upgrade uint8 to int16.
			e := expr.expr.(*Uint8Literal)
			return ExprValue{
				NodeKind: KindInt16Literal,
				TypeKind: types.Int16,
				expr: &Int16Literal{
					Token: e.Token,
					Value: int16(e.Value),
				},
			}
		case types.Int32:
			// Upgrade uint8 to int32.
			e := expr.expr.(*Uint8Literal)
			return ExprValue{
				NodeKind: KindInt32Literal,
				TypeKind: types.Int32,
				expr: &Int32Literal{
					Token: e.Token,
					Value: int32(e.Value),
				},
			}
		case types.Int64:
			// Upgrade uint8 to int64.
			e := expr.expr.(*Uint8Literal)
			return ExprValue{
				NodeKind: KindInt64Literal,
				TypeKind: types.Int64,
				expr: &Int64Literal{
					Token: e.Token,
					Value: int64(e.Value),
				},
			}
		case types.Int128:
			// Upgrade uint8 to int128.
			e := expr.expr.(*Uint8Literal)
			return ExprValue{
				NodeKind: KindInt128Literal,
				TypeKind: types.Int128,
				expr: &Int128Literal{
					Token: e.Token,
					Value: wide.Int128FromInt64(int64(e.Value)),
				},
			}
		case types.Float16:
			// Upgrade uint8 to float16.
			e := expr.expr.(*Uint8Literal)
			return ExprValue{
				NodeKind: KindFloat16Literal,
				TypeKind: types.Float16,
				expr: &Float16Literal{
					Token: e.Token,
					Value: f16.Fromfloat32(float32(e.Value)),
				},
			}
		case types.Float32:
			// Upgrade uint8 to float32.
			e := expr.expr.(*Uint8Literal)
			return ExprValue{
				NodeKind: KindFloat32Literal,
				TypeKind: types.Float32,
				expr: &Float32Literal{
					Token: e.Token,
					Value: float32(e.Value),
				},
			}
		case types.Float64:
			// Upgrade uint8 to float64.
			e := expr.expr.(*Uint8Literal)
			return ExprValue{
				NodeKind: KindFloat64Literal,
				TypeKind: types.Float64,
				expr: &Float64Literal{
					Token: e.Token,
					Value: float64(e.Value),
				},
			}
		}
	case KindUint16Literal:
		switch refType {
		case types.Uint32:
			// Upgrade uint16 to uint32.
			e := expr.expr.(*Uint16Literal)
			return ExprValue{
				NodeKind: KindUint32Literal,
				TypeKind: types.Uint32,
				expr: &Uint32Literal{
					Token: e.Token,
					Value: uint32(e.Value),
				},
			}
		case types.Uint64:
			// Upgrade uint16 to uint64.
			e := expr.expr.(*Uint16Literal)
			return ExprValue{
				NodeKind: KindUint64Literal,
				TypeKind: types.Uint64,
				expr: &Uint64Literal{
					Token: e.Token,
					Value: uint64(e.Value),
				},
			}
		case types.Uint128:
			// Upgrade uint16 to uint128.
			e := expr.expr.(*Uint16Literal)
			return ExprValue{
				NodeKind: KindUint128Literal,
				TypeKind: types.Uint128,
				expr: &Uint128Literal{
					Token: e.Token,
					Value: u128.From64(uint64(e.Value)),
				},
			}
		case types.Int32:
			// Upgrade uint16 to int32.
			e := expr.expr.(*Uint16Literal)
			return ExprValue{
				NodeKind: KindInt32Literal,
				TypeKind: types.Int32,
				expr: &Int32Literal{
					Token: e.Token,
					Value: int32(e.Value),
				},
			}
		case types.Int64:
			// Upgrade uint16 to int64.
			e := expr.expr.(*Uint16Literal)
			return ExprValue{
				NodeKind: KindInt64Literal,
				TypeKind: types.Int64,
				expr: &Int64Literal{
					Token: e.Token,
					Value: int64(e.Value),
				},
			}
		case types.Int128:
			// Upgrade uint16 to int128.
			e := expr.expr.(*Uint16Literal)
			return ExprValue{
				NodeKind: KindInt128Literal,
				TypeKind: types.Int128,
				expr: &Int128Literal{
					Token: e.Token,
					Value: wide.Int128FromInt64(int64(e.Value)),
				},
			}
		case types.Float16:
			// Upgrade uint16 to float16.
			e := expr.expr.(*Uint16Literal)
			return ExprValue{
				NodeKind: KindFloat16Literal,
				TypeKind: types.Float16,
				expr: &Float16Literal{
					Token: e.Token,
					Value: f16.Fromfloat32(float32(e.Value)),
				},
			}
		case types.Float32:
			// Upgrade uint16 to float32.
			e := expr.expr.(*Uint16Literal)
			return ExprValue{
				NodeKind: KindFloat32Literal,
				TypeKind: types.Float32,
				expr: &Float32Literal{
					Token: e.Token,
					Value: float32(e.Value),
				},
			}
		case types.Float64:
			// Upgrade uint16 to float64.
			e := expr.expr.(*Uint16Literal)
			return ExprValue{
				NodeKind: KindFloat64Literal,
				TypeKind: types.Float64,
				expr: &Float64Literal{
					Token: e.Token,
					Value: float64(e.Value),
				},
			}
		}
	case KindUint32Literal:
		switch refType {
		case types.Uint64:
			// Upgrade uint32 to uint64.
			e := expr.expr.(*Uint32Literal)
			return ExprValue{
				NodeKind: KindUint64Literal,
				TypeKind: types.Uint64,
				expr: &Uint64Literal{
					Token: e.Token,
					Value: uint64(e.Value),
				},
			}
		case types.Uint128:
			// Upgrade uint32 to uint128.
			e := expr.expr.(*Uint32Literal)
			return ExprValue{
				NodeKind: KindUint128Literal,
				TypeKind: types.Uint128,
				expr: &Uint128Literal{
					Token: e.Token,
					Value: u128.From64(uint64(e.Value)),
				},
			}
		case types.Int64:
			// Upgrade uint32 to int64.
			e := expr.expr.(*Uint32Literal)
			return ExprValue{
				NodeKind: KindInt64Literal,
				TypeKind: types.Int64,
				expr: &Int64Literal{
					Token: e.Token,
					Value: int64(e.Value),
				},
			}
		case types.Int128:
			// Upgrade uint32 to int128.
			e := expr.expr.(*Uint32Literal)
			return ExprValue{
				NodeKind: KindInt128Literal,
				TypeKind: types.Int128,
				expr: &Int128Literal{
					Token: e.Token,
					Value: wide.Int128FromInt64(int64(e.Value)),
				},
			}
		case types.Float32:
			// Upgrade uint32 to float32.
			e := expr.expr.(*Uint32Literal)
			return ExprValue{
				NodeKind: KindFloat32Literal,
				TypeKind: types.Float32,
				expr: &Float32Literal{
					Token: e.Token,
					Value: float32(e.Value),
				},
			}
		case types.Float64:
			// Upgrade uint32 to float64.
			e := expr.expr.(*Uint32Literal)
			return ExprValue{
				NodeKind: KindFloat64Literal,
				TypeKind: types.Float64,
				expr: &Float64Literal{
					Token: e.Token,
					Value: float64(e.Value),
				},
			}
		}
	case KindUint64Literal:
		switch refType {
		case types.Uint128:
			// Upgrade uint64 to uint128.
			e := expr.expr.(*Uint64Literal)
			return ExprValue{
				NodeKind: KindUint128Literal,
				TypeKind: types.Uint128,
				expr: &Uint128Literal{
					Token: e.Token,
					Value: u128.From64(uint64(e.Value)),
				},
			}
		case types.Int128:
			// Upgrade uint64 to int128.
			e := expr.expr.(*Uint64Literal)
			return ExprValue{
				NodeKind: KindInt128Literal,
				TypeKind: types.Int128,
				expr: &Int128Literal{
					Token: e.Token,
					Value: wide.Int128FromInt64(int64(e.Value)),
				},
			}
		case types.Float64:
			// Upgrade uint64 to float64.
			e := expr.expr.(*Uint64Literal)
			return ExprValue{
				NodeKind: KindFloat64Literal,
				TypeKind: types.Float64,
				expr: &Float64Literal{
					Token: e.Token,
					Value: float64(e.Value),
				},
			}
		}
	}

	return expr
}
