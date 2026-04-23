package ast

import (
	"testing"

	"github.com/samborkent/cog/internal/types"
)

type Expr interface {
	Node
	Type() types.Type
}

type ExprValue struct {
	NodeKind NodeKind
	TypeKind types.Kind
	expr     Expr
}

var ZeroExpr ExprValue

func NewExpr[T Expr](kind NodeKind, typeKind types.Kind, expr T) ExprValue {
	return ExprValue{
		NodeKind: kind,
		TypeKind: typeKind,
		expr:     expr,
	}
}

func (v ExprValue) Expr(t *testing.T) Expr {
	t.Helper()
	return v.expr
}

func (v ExprValue) String() string {
	return v.expr.String()
}

func (v ExprValue) Type() types.Type {
	if v == ZeroExpr {
		return types.None
	}

	return v.expr.Type()
}

func (v ExprValue) AsArrayLiteral() *ArrayLiteral {
	return v.expr.(*ArrayLiteral)
}

func (v ExprValue) AsCall() *Call {
	return v.expr.(*Call)
}

func (v ExprValue) AsEitherLiteral() *EitherLiteral {
	return v.expr.(*EitherLiteral)
}

func (v ExprValue) AsFloat16Literal() *Float16Literal {
	return v.expr.(*Float16Literal)
}

func (v ExprValue) AsIdentifier() *Identifier {
	return v.expr.(*Identifier)
}

func (v ExprValue) AsMapLiteral() *MapLiteral {
	return v.expr.(*MapLiteral)
}

func (v ExprValue) AsPrefix() *Prefix {
	return v.expr.(*Prefix)
}

func (v ExprValue) AsProcedureLiteral() *ProcedureLiteral {
	return v.expr.(*ProcedureLiteral)
}

func (v ExprValue) AsResultLiteral() *ResultLiteral {
	return v.expr.(*ResultLiteral)
}

func (v ExprValue) AsSelector() *Selector {
	return v.expr.(*Selector)
}

func (v ExprValue) AsSetLiteral() *SetLiteral {
	return v.expr.(*SetLiteral)
}

func (v ExprValue) AsStructLiteral() *StructLiteral {
	return v.expr.(*StructLiteral)
}

func (v ExprValue) AsSuffix() *Suffix {
	return v.expr.(*Suffix)
}

func (v ExprValue) AsTupleLiteral() *TupleLiteral {
	return v.expr.(*TupleLiteral)
}
