package parser

import (
	"fmt"

	"github.com/samborkent/cog/internal/ast"
	"github.com/samborkent/cog/internal/tokens"
	"github.com/samborkent/cog/internal/types"
)

func (p *Parser) parseLiteral(tokenType types.Type) ast.ExprIndex {
	var (
		expr ast.ExprIndex
		err  error
	)

	// Inferred types.
	if tokenType == types.None {
		switch p.this().Type {
		case tokens.FloatLiteral:
			expr, err = p.ast.NewFloat64Literal(p.this())
			if err != nil {
				p.error(p.this(), err.Error(), "parseLiteral")
				return ast.ZeroExprIndex
			}
		case tokens.IntLiteral:
			expr, err = p.ast.NewInt64Literal(p.this())
			if err != nil {
				p.error(p.this(), err.Error(), "parseLiteral")
				return ast.ZeroExprIndex
			}
		case tokens.StringLiteral:
			expr = p.ast.NewUTF8Literal(p.this())
		default:
			p.error(p.this(), "unexpected token found in rhs of variable declaration", "parseLiteral")
			return ast.ZeroExprIndex
		}

		p.advance("parseLiteral") // consume literal

		return expr
	}

	// When expected type is a result, parse the literal using the value type.
	// The declaration/assignment will handle marking the check state.
	if r, ok := tokenType.Underlying().(*types.Result); ok {
		tokenType = r.Value
	}

	t, ok := tokenType.Underlying().(*types.Basic)
	if !ok {
		p.error(p.this(), fmt.Sprintf("expected basic or union type for literal, got %q", tokenType), "parseLiteral")
		return ast.ZeroExprIndex
	}

	switch t.Kind() {
	case types.ASCII:
		if p.this().Type != tokens.StringLiteral {
			p.error(p.this(), "ascii: expected string literal", "parseLiteral")
			return ast.ZeroExprIndex
		}

		expr, err = p.ast.NewASCIILiteral(p.this())
		if err != nil {
			p.error(p.this(), err.Error(), "parseLiteral")
			return ast.ZeroExprIndex
		}
	case types.Bool:
		if p.this().Type != tokens.True && p.this().Type != tokens.False {
			p.error(p.this(), "expected bool literal", "parseLiteral")
			return ast.ZeroExprIndex
		}

		expr = p.ast.NewBoolLiteral(p.this())
	case types.Float16:
		if p.this().Type != tokens.FloatLiteral && p.this().Type != tokens.IntLiteral {
			p.error(p.this(), "float16: expected number literal", "parseLiteral")
			return ast.ZeroExprIndex
		}

		expr, err = p.ast.NewFloat16Literal(p.this())
		if err != nil {
			p.error(p.this(), err.Error(), "parseLiteral")
			return ast.ZeroExprIndex
		}
	case types.Float32:
		if p.this().Type != tokens.FloatLiteral && p.this().Type != tokens.IntLiteral {
			p.error(p.this(), "float32: expected float literal", "parseLiteral")
			return ast.ZeroExprIndex
		}

		expr, err = p.ast.NewFloat32Literal(p.this())
		if err != nil {
			p.error(p.this(), err.Error(), "parseLiteral")
			return ast.ZeroExprIndex
		}
	case types.Float64:
		if p.this().Type != tokens.FloatLiteral && p.this().Type != tokens.IntLiteral {
			p.error(p.this(), "float64: expected float literal", "parseLiteral")
			return ast.ZeroExprIndex
		}

		expr, err = p.ast.NewFloat64Literal(p.this())
		if err != nil {
			p.error(p.this(), err.Error(), "parseLiteral")
			return ast.ZeroExprIndex
		}
	case types.Int8:
		if p.this().Type != tokens.IntLiteral && p.this().Type != tokens.FloatLiteral {
			p.error(p.this(), "int8: expected int literal", "parseLiteral")
			return ast.ZeroExprIndex
		}

		expr, err = p.ast.NewInt8Literal(p.this())
		if err != nil {
			p.error(p.this(), err.Error(), "parseLiteral")
			return ast.ZeroExprIndex
		}
	case types.Int16:
		if p.this().Type != tokens.IntLiteral && p.this().Type != tokens.FloatLiteral {
			p.error(p.this(), "int16: expected int literal", "parseLiteral")
			return ast.ZeroExprIndex
		}

		expr, err = p.ast.NewInt16Literal(p.this())
		if err != nil {
			p.error(p.this(), err.Error(), "parseLiteral")
			return ast.ZeroExprIndex
		}
	case types.Int32:
		if p.this().Type != tokens.IntLiteral && p.this().Type != tokens.FloatLiteral {
			p.error(p.this(), "int32: expected int literal", "parseLiteral")
			return ast.ZeroExprIndex
		}

		expr, err = p.ast.NewInt32Literal(p.this())
		if err != nil {
			p.error(p.this(), err.Error(), "parseLiteral")
			return ast.ZeroExprIndex
		}
	case types.Int64:
		if p.this().Type != tokens.IntLiteral && p.this().Type != tokens.FloatLiteral {
			p.error(p.this(), "int64: expected int literal", "parseLiteral")
			return ast.ZeroExprIndex
		}

		expr, err = p.ast.NewInt64Literal(p.this())
		if err != nil {
			p.error(p.this(), err.Error(), "parseLiteral")
			return ast.ZeroExprIndex
		}
	case types.Int128:
		if p.this().Type != tokens.IntLiteral && p.this().Type != tokens.FloatLiteral {
			p.error(p.this(), "int128: expected int literal", "parseLiteral")
			return ast.ZeroExprIndex
		}

		expr, err = p.ast.NewInt128Literal(p.this())
		if err != nil {
			p.error(p.this(), err.Error(), "parseLiteral")
			return ast.ZeroExprIndex
		}
	case types.Uint8:
		if p.this().Type != tokens.IntLiteral && p.this().Type != tokens.FloatLiteral {
			p.error(p.this(), "uint8: expected int literal", "parseLiteral")
			return ast.ZeroExprIndex
		}

		expr, err = p.ast.NewUint8Literal(p.this())
		if err != nil {
			p.error(p.this(), err.Error(), "parseLiteral")
			return ast.ZeroExprIndex
		}
	case types.Uint16:
		if p.this().Type != tokens.IntLiteral && p.this().Type != tokens.FloatLiteral {
			p.error(p.this(), "uint16: expected int literal", "parseLiteral")
			return ast.ZeroExprIndex
		}

		expr, err = p.ast.NewUint16Literal(p.this())
		if err != nil {
			p.error(p.this(), err.Error(), "parseLiteral")
			return ast.ZeroExprIndex
		}
	case types.Uint32:
		if p.this().Type != tokens.IntLiteral && p.this().Type != tokens.FloatLiteral {
			p.error(p.this(), "uint32: expected int literal", "parseLiteral")
			return ast.ZeroExprIndex
		}

		expr, err = p.ast.NewUint32Literal(p.this())
		if err != nil {
			p.error(p.this(), err.Error(), "parseLiteral")
			return ast.ZeroExprIndex
		}
	case types.Uint64:
		if p.this().Type != tokens.IntLiteral && p.this().Type != tokens.FloatLiteral {
			p.error(p.this(), "uint64: expected int literal", "parseLiteral")
			return ast.ZeroExprIndex
		}

		expr, err = p.ast.NewUint64Literal(p.this())
		if err != nil {
			p.error(p.this(), err.Error(), "parseLiteral")
			return ast.ZeroExprIndex
		}
	case types.Uint128:
		if p.this().Type != tokens.IntLiteral && p.this().Type != tokens.FloatLiteral {
			p.error(p.this(), "uint128: expected int literal", "parseLiteral")
			return ast.ZeroExprIndex
		}

		expr, err = p.ast.NewUint128Literal(p.this())
		if err != nil {
			p.error(p.this(), err.Error(), "parseLiteral")
			return ast.ZeroExprIndex
		}
	case types.UTF8:
		if p.this().Type != tokens.StringLiteral {
			p.error(p.this(), "utf8: expected string literal", "parseLiteral")
			return ast.ZeroExprIndex
		}

		expr = p.ast.NewUTF8Literal(p.this())
	default:
		p.error(p.this(), "unsupported type: "+tokenType.String(), "parseLiteral")
		return ast.ZeroExprIndex
	}

	p.advance("parseLiteral") // consume literal

	return expr
}
