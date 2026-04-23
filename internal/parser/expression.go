package parser

import (
	"fmt"

	"github.com/samborkent/cog/internal/ast"
	"github.com/samborkent/cog/internal/tokens"
	"github.com/samborkent/cog/internal/types"
)

func (p *Parser) parseLiteral(tokenType types.Type) ast.ExprValue {
	var (
		node     ast.Expr
		nodeKind ast.NodeKind
		typeKind types.Kind
		err      error
	)

	// Inferred types.
	if tokenType == types.None {
		switch p.this().Type {
		case tokens.FloatLiteral:
			node, err = ast.NewFloat64Literal(p.this())
			if err != nil {
				p.error(p.this(), err.Error(), "parseLiteral")
				return ast.ZeroExpr
			}

			nodeKind = ast.KindFloat64Literal
			typeKind = types.Float64
		case tokens.IntLiteral:
			node, err = ast.NewInt64Literal(p.this())
			if err != nil {
				p.error(p.this(), err.Error(), "parseLiteral")
				return ast.ZeroExpr
			}

			nodeKind = ast.KindInt64Literal
			typeKind = types.Int64
		case tokens.StringLiteral:
			node = ast.NewUTF8Literal(p.this())
			nodeKind = ast.KindUTF8Literal
			typeKind = types.UTF8
		default:
			p.error(p.this(), "unexpected token found in rhs of variable declaration", "parseLiteral")
			return ast.ZeroExpr
		}

		p.advance("parseLiteral") // consume literal

		return ast.NewExpr(nodeKind, typeKind, node)
	}

	// When expected type is a result, parse the literal using the value type.
	// The declaration/assignment will handle marking the check state.
	if r, ok := tokenType.Underlying().(*types.Result); ok {
		tokenType = r.Value
	}

	t, ok := tokenType.Underlying().(*types.Basic)
	if !ok {
		p.error(p.this(), fmt.Sprintf("expected basic or union type for literal, got %q", tokenType), "parseLiteral")
		return ast.ZeroExpr
	}

	switch t.Kind() {
	case types.ASCII:
		if p.this().Type != tokens.StringLiteral {
			p.error(p.this(), "ascii: expected string literal", "parseLiteral")
			return ast.ZeroExpr
		}

		node, err = ast.NewASCIILiteral(p.this())
		if err != nil {
			p.error(p.this(), err.Error(), "parseLiteral")
			return ast.ZeroExpr
		}

		nodeKind = ast.KindASCIILiteral
		typeKind = types.ASCII
	case types.Bool:
		if p.this().Type != tokens.True && p.this().Type != tokens.False {
			p.error(p.this(), "expected bool literal", "parseLiteral")
			return ast.ZeroExpr
		}

		node, err = ast.NewBoolLiteral(p.this())
		if err != nil {
			p.error(p.this(), err.Error(), "parseLiteral")
			return ast.ZeroExpr
		}

		nodeKind = ast.KindBoolLiteral
		typeKind = types.Bool
	case types.Float16:
		if p.this().Type != tokens.FloatLiteral && p.this().Type != tokens.IntLiteral {
			p.error(p.this(), "float16: expected number literal", "parseLiteral")
			return ast.ZeroExpr
		}

		node, err = ast.NewFloat16Literal(p.this())
		if err != nil {
			p.error(p.this(), err.Error(), "parseLiteral")
			return ast.ZeroExpr
		}

		nodeKind = ast.KindFloat16Literal
		typeKind = types.Float16
	case types.Float32:
		if p.this().Type != tokens.FloatLiteral && p.this().Type != tokens.IntLiteral {
			p.error(p.this(), "float32: expected float literal", "parseLiteral")
			return ast.ZeroExpr
		}

		node, err = ast.NewFloat32Literal(p.this())
		if err != nil {
			p.error(p.this(), err.Error(), "parseLiteral")
			return ast.ZeroExpr
		}

		nodeKind = ast.KindFloat32Literal
		typeKind = types.Float32
	case types.Float64:
		if p.this().Type != tokens.FloatLiteral && p.this().Type != tokens.IntLiteral {
			p.error(p.this(), "float64: expected float literal", "parseLiteral")
			return ast.ZeroExpr
		}

		node, err = ast.NewFloat64Literal(p.this())
		if err != nil {
			p.error(p.this(), err.Error(), "parseLiteral")
			return ast.ZeroExpr
		}

		nodeKind = ast.KindFloat64Literal
		typeKind = types.Float64
	case types.Int8:
		if p.this().Type != tokens.IntLiteral && p.this().Type != tokens.FloatLiteral {
			p.error(p.this(), "int8: expected int literal", "parseLiteral")
			return ast.ZeroExpr
		}

		node, err = ast.NewInt8Literal(p.this())
		if err != nil {
			p.error(p.this(), err.Error(), "parseLiteral")
			return ast.ZeroExpr
		}

		nodeKind = ast.KindInt8Literal
		typeKind = types.Int8
	case types.Int16:
		if p.this().Type != tokens.IntLiteral && p.this().Type != tokens.FloatLiteral {
			p.error(p.this(), "int16: expected int literal", "parseLiteral")
			return ast.ZeroExpr
		}

		node, err = ast.NewInt16Literal(p.this())
		if err != nil {
			p.error(p.this(), err.Error(), "parseLiteral")
			return ast.ZeroExpr
		}

		nodeKind = ast.KindInt16Literal
		typeKind = types.Int16
	case types.Int32:
		if p.this().Type != tokens.IntLiteral && p.this().Type != tokens.FloatLiteral {
			p.error(p.this(), "int32: expected int literal", "parseLiteral")
			return ast.ZeroExpr
		}

		node, err = ast.NewInt32Literal(p.this())
		if err != nil {
			p.error(p.this(), err.Error(), "parseLiteral")
			return ast.ZeroExpr
		}

		nodeKind = ast.KindInt32Literal
		typeKind = types.Int32
	case types.Int64:
		if p.this().Type != tokens.IntLiteral && p.this().Type != tokens.FloatLiteral {
			p.error(p.this(), "int64: expected int literal", "parseLiteral")
			return ast.ZeroExpr
		}

		node, err = ast.NewInt64Literal(p.this())
		if err != nil {
			p.error(p.this(), err.Error(), "parseLiteral")
			return ast.ZeroExpr
		}

		nodeKind = ast.KindInt64Literal
		typeKind = types.Int64
	case types.Int128:
		if p.this().Type != tokens.IntLiteral && p.this().Type != tokens.FloatLiteral {
			p.error(p.this(), "int128: expected int literal", "parseLiteral")
			return ast.ZeroExpr
		}

		node, err = ast.NewInt128Literal(p.this())
		if err != nil {
			p.error(p.this(), err.Error(), "parseLiteral")
			return ast.ZeroExpr
		}

		nodeKind = ast.KindInt128Literal
		typeKind = types.Int128
	case types.Uint8:
		if p.this().Type != tokens.IntLiteral && p.this().Type != tokens.FloatLiteral {
			p.error(p.this(), "uint8: expected int literal", "parseLiteral")
			return ast.ZeroExpr
		}

		node, err = ast.NewUint8Literal(p.this())
		if err != nil {
			p.error(p.this(), err.Error(), "parseLiteral")
			return ast.ZeroExpr
		}

		nodeKind = ast.KindUint8Literal
		typeKind = types.Uint8
	case types.Uint16:
		if p.this().Type != tokens.IntLiteral && p.this().Type != tokens.FloatLiteral {
			p.error(p.this(), "uint16: expected int literal", "parseLiteral")
			return ast.ZeroExpr
		}

		node, err = ast.NewUint16Literal(p.this())
		if err != nil {
			p.error(p.this(), err.Error(), "parseLiteral")
			return ast.ZeroExpr
		}

		nodeKind = ast.KindUint16Literal
		typeKind = types.Uint16
	case types.Uint32:
		if p.this().Type != tokens.IntLiteral && p.this().Type != tokens.FloatLiteral {
			p.error(p.this(), "uint32: expected int literal", "parseLiteral")
			return ast.ZeroExpr
		}

		node, err = ast.NewUint32Literal(p.this())
		if err != nil {
			p.error(p.this(), err.Error(), "parseLiteral")
			return ast.ZeroExpr
		}

		nodeKind = ast.KindUint32Literal
		typeKind = types.Uint32
	case types.Uint64:
		if p.this().Type != tokens.IntLiteral && p.this().Type != tokens.FloatLiteral {
			p.error(p.this(), "uint64: expected int literal", "parseLiteral")
			return ast.ZeroExpr
		}

		node, err = ast.NewUint64Literal(p.this())
		if err != nil {
			p.error(p.this(), err.Error(), "parseLiteral")
			return ast.ZeroExpr
		}

		nodeKind = ast.KindUint64Literal
		typeKind = types.Uint64
	case types.Uint128:
		if p.this().Type != tokens.IntLiteral && p.this().Type != tokens.FloatLiteral {
			p.error(p.this(), "uint128: expected int literal", "parseLiteral")
			return ast.ZeroExpr
		}

		node, err = ast.NewUint128Literal(p.this())
		if err != nil {
			p.error(p.this(), err.Error(), "parseLiteral")
			return ast.ZeroExpr
		}

		nodeKind = ast.KindUint128Literal
		typeKind = types.Uint128
	case types.UTF8:
		if p.this().Type != tokens.StringLiteral {
			p.error(p.this(), "utf8: expected string literal", "parseLiteral")
			return ast.ZeroExpr
		}

		node = ast.NewUTF8Literal(p.this())
		nodeKind = ast.KindUTF8Literal
		typeKind = types.UTF8
	default:
		p.error(p.this(), "unsupported type: "+tokenType.String(), "parseLiteral")
		return ast.ZeroExpr
	}

	p.advance("parseLiteral") // consume literal

	return ast.NewExpr(nodeKind, typeKind, node)
}
