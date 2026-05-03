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
	InfixType   types.Type
	Left, Right ExprIndex
}

func (a *AST) NewInfix(operator tokens.Token, infixType types.Type, left, right ExprIndex) ExprIndex {
	infixExpr := New[Infix](a)
	infixExpr.Operator = operator
	infixExpr.InfixType = infixType
	infixExpr.Left = left
	infixExpr.Right = right
	return a.AddExpr(infixExpr)
}

func (e *Infix) Pos() (uint32, uint16) {
	return e.Operator.Ln, e.Operator.Col
}

func (e *Infix) Hash() uint64 {
	return hash(e)
}

func (e *Infix) StringTo(out *strings.Builder, a *AST) {
	_ = out.WriteByte('(')
	a.exprs[e.Left].StringTo(out, a)
	_ = out.WriteByte(' ')
	_, _ = out.WriteString(e.Operator.Type.String())
	_ = out.WriteByte(' ')
	a.exprs[e.Right].StringTo(out, a)
	_ = out.WriteByte(')')
}

func (e *Infix) Type() types.Type {
	// Return bool type for comparison operators
	switch e.Operator.Type {
	case tokens.And, tokens.Or,
		tokens.Equal, tokens.NotEqual,
		tokens.GT, tokens.GTEqual, tokens.LT, tokens.LTEqual:
		return types.Basics[types.Bool]
	}

	return e.InfixType
}

func EqualizeInfixTypes(left, right Expr) {
	if types.Equal(left.Type(), right.Type()) {
		return
	}

	// Handle default inferred literal types.
	switch left.Type().Kind() {
	case types.ASCII:
		if r, ok := right.(*UTF8Literal); ok {
			right = &ASCIILiteral{
				Token: r.Token,
				Value: ascii(r.Value),
			}

			return
		}
	case types.Float16:
		if r, ok := right.(*Float64Literal); ok {
			right = &Float16Literal{
				Token: r.Token,
				Value: f16.Fromfloat32(float32(r.Value)),
			}

			return
		}
	case types.Float32:
		if r, ok := right.(*Float64Literal); ok {
			right = &Float32Literal{
				Token: r.Token,
				Value: float32(r.Value),
			}

			return
		}
	case types.Int8:
		if r, ok := right.(*Int64Literal); ok && (r.Value >= math.MinInt8 && r.Value <= math.MaxInt8) {
			right = &Int8Literal{
				Token: r.Token,
				Value: int8(r.Value),
			}

			return
		}
	case types.Int16:
		if r, ok := right.(*Int64Literal); ok && (r.Value >= math.MinInt16 && r.Value <= math.MaxInt16) {
			right = &Int16Literal{
				Token: r.Token,
				Value: int16(r.Value),
			}

			return
		}
	case types.Int32:
		if r, ok := right.(*Int64Literal); ok && (r.Value >= math.MinInt32 && r.Value <= math.MaxInt32) {
			right = &Int32Literal{
				Token: r.Token,
				Value: int32(r.Value),
			}

			return
		}
	case types.Int128:
		if r, ok := right.(*Int64Literal); ok {
			right = &Int128Literal{
				Token: r.Token,
				Value: wide.Int128FromInt64(r.Value),
			}

			return
		}
	case types.Uint8:
		if r, ok := right.(*Int64Literal); ok && (r.Value >= 0 && r.Value <= math.MaxUint8) {
			right = &Uint8Literal{
				Token: r.Token,
				Value: uint8(r.Value),
			}

			return
		}
	case types.Uint16:
		if r, ok := right.(*Int64Literal); ok && (r.Value >= 0 && r.Value <= math.MaxUint16) {
			right = &Uint16Literal{
				Token: r.Token,
				Value: uint16(r.Value),
			}

			return
		}
	case types.Uint32:
		if r, ok := right.(*Int64Literal); ok && (r.Value >= 0 && r.Value <= math.MaxUint32) {
			right = &Uint32Literal{
				Token: r.Token,
				Value: uint32(r.Value),
			}

			return
		}
	case types.Uint64:
		if r, ok := right.(*Int64Literal); ok && r.Value >= 0 {
			right = &Uint64Literal{
				Token: r.Token,
				Value: uint64(r.Value),
			}

			return
		}
	case types.Uint128:
		if r, ok := right.(*Int64Literal); ok && r.Value >= 0 {
			right = &Uint128Literal{
				Token: r.Token,
				Value: u128.From64(uint64(r.Value)),
			}

			return
		}
	}

	left = upgradeLiteralType(left, right)
	right = upgradeLiteralType(right, left)
}

func upgradeLiteralType(expr Expr, ref Expr) Expr {
	refType := ref.Type().Kind()

	if expr.Type().Kind() == refType {
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
				Value: Complex32To64(e.Value),
			}
		case types.Complex128:
			// Upgrade complex32 to complex128.
			return &Complex128Literal{
				Token: e.Token,
				Value: complex128(Complex32To64(e.Value)),
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
				Value: wide.Int128FromInt64(int64(e.Value)),
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
				Value: wide.Int128FromInt64(int64(e.Value)),
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
				Value: wide.Int128FromInt64(int64(e.Value)),
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
				Value: wide.Int128FromInt64(e.Value),
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
				Value: wide.Int128FromInt64(int64(e.Value)),
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
				Value: wide.Int128FromInt64(int64(e.Value)),
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
				Value: wide.Int128FromInt64(int64(e.Value)),
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
				Value: wide.Int128FromInt64(int64(e.Value)),
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
