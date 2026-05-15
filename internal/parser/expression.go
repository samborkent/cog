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
		switch p.lex.This().Type {
		case tokens.FloatLiteral:
			expr, err = p.ast.NewFloat64Literal(p.lex.This())
			if err != nil {
				p.error(p.lex.This(), err.Error(), "parseLiteral")
				return ast.ZeroExprIndex
			}
		case tokens.IntLiteral:
			expr, err = p.ast.NewInt64Literal(p.lex.This())
			if err != nil {
				p.error(p.lex.This(), err.Error(), "parseLiteral")
				return ast.ZeroExprIndex
			}
		case tokens.StringLiteral:
			expr = p.ast.NewUTF8Literal(p.lex.This())
		default:
			p.error(p.lex.This(), "unexpected token found in rhs of variable declaration", "parseLiteral")
			return ast.ZeroExprIndex
		}

		p.lex.Step() // consume literal

		return expr
	}

	// When expected type is a result, parse the literal using the value type.
	// The declaration/assignment will handle marking the check state.
	if r, ok := tokenType.Underlying().(*types.Result); ok {
		tokenType = r.Value
	}

	t, ok := tokenType.Underlying().(*types.Basic)
	if !ok {
		p.error(p.lex.This(), fmt.Sprintf("expected basic or union type for literal, got %q", tokenType), "parseLiteral")
		return ast.ZeroExprIndex
	}

	switch t.Kind() {
	case types.ASCII:
		if p.lex.This().Type != tokens.StringLiteral {
			p.error(p.lex.This(), "ascii: expected string literal", "parseLiteral")
			return ast.ZeroExprIndex
		}

		expr, err = p.ast.NewASCIILiteral(p.lex.This())
		if err != nil {
			p.error(p.lex.This(), err.Error(), "parseLiteral")
			return ast.ZeroExprIndex
		}
	case types.Bool:
		if p.lex.This().Type != tokens.True && p.lex.This().Type != tokens.False {
			p.error(p.lex.This(), "expected bool literal", "parseLiteral")
			return ast.ZeroExprIndex
		}

		expr = p.ast.NewBoolLiteral(p.lex.This())
	case types.Float16:
		if p.lex.This().Type != tokens.FloatLiteral && p.lex.This().Type != tokens.IntLiteral {
			p.error(p.lex.This(), "float16: expected number literal", "parseLiteral")
			return ast.ZeroExprIndex
		}

		expr, err = p.ast.NewFloat16Literal(p.lex.This())
		if err != nil {
			p.error(p.lex.This(), err.Error(), "parseLiteral")
			return ast.ZeroExprIndex
		}
	case types.Float32:
		if p.lex.This().Type != tokens.FloatLiteral && p.lex.This().Type != tokens.IntLiteral {
			p.error(p.lex.This(), "float32: expected float literal", "parseLiteral")
			return ast.ZeroExprIndex
		}

		expr, err = p.ast.NewFloat32Literal(p.lex.This())
		if err != nil {
			p.error(p.lex.This(), err.Error(), "parseLiteral")
			return ast.ZeroExprIndex
		}
	case types.Float64:
		if p.lex.This().Type != tokens.FloatLiteral && p.lex.This().Type != tokens.IntLiteral {
			p.error(p.lex.This(), "float64: expected float literal", "parseLiteral")
			return ast.ZeroExprIndex
		}

		expr, err = p.ast.NewFloat64Literal(p.lex.This())
		if err != nil {
			p.error(p.lex.This(), err.Error(), "parseLiteral")
			return ast.ZeroExprIndex
		}
	case types.Int8:
		if p.lex.This().Type != tokens.IntLiteral && p.lex.This().Type != tokens.FloatLiteral {
			p.error(p.lex.This(), "int8: expected int literal", "parseLiteral")
			return ast.ZeroExprIndex
		}

		expr, err = p.ast.NewInt8Literal(p.lex.This())
		if err != nil {
			p.error(p.lex.This(), err.Error(), "parseLiteral")
			return ast.ZeroExprIndex
		}
	case types.Int16:
		if p.lex.This().Type != tokens.IntLiteral && p.lex.This().Type != tokens.FloatLiteral {
			p.error(p.lex.This(), "int16: expected int literal", "parseLiteral")
			return ast.ZeroExprIndex
		}

		expr, err = p.ast.NewInt16Literal(p.lex.This())
		if err != nil {
			p.error(p.lex.This(), err.Error(), "parseLiteral")
			return ast.ZeroExprIndex
		}
	case types.Int32:
		if p.lex.This().Type != tokens.IntLiteral && p.lex.This().Type != tokens.FloatLiteral {
			p.error(p.lex.This(), "int32: expected int literal", "parseLiteral")
			return ast.ZeroExprIndex
		}

		expr, err = p.ast.NewInt32Literal(p.lex.This())
		if err != nil {
			p.error(p.lex.This(), err.Error(), "parseLiteral")
			return ast.ZeroExprIndex
		}
	case types.Int64:
		if p.lex.This().Type != tokens.IntLiteral && p.lex.This().Type != tokens.FloatLiteral {
			p.error(p.lex.This(), "int64: expected int literal", "parseLiteral")
			return ast.ZeroExprIndex
		}

		expr, err = p.ast.NewInt64Literal(p.lex.This())
		if err != nil {
			p.error(p.lex.This(), err.Error(), "parseLiteral")
			return ast.ZeroExprIndex
		}
	case types.Int128:
		if p.lex.This().Type != tokens.IntLiteral && p.lex.This().Type != tokens.FloatLiteral {
			p.error(p.lex.This(), "int128: expected int literal", "parseLiteral")
			return ast.ZeroExprIndex
		}

		expr, err = p.ast.NewInt128Literal(p.lex.This())
		if err != nil {
			p.error(p.lex.This(), err.Error(), "parseLiteral")
			return ast.ZeroExprIndex
		}
	case types.Uint8:
		if p.lex.This().Type != tokens.IntLiteral && p.lex.This().Type != tokens.FloatLiteral {
			p.error(p.lex.This(), "uint8: expected int literal", "parseLiteral")
			return ast.ZeroExprIndex
		}

		expr, err = p.ast.NewUint8Literal(p.lex.This())
		if err != nil {
			p.error(p.lex.This(), err.Error(), "parseLiteral")
			return ast.ZeroExprIndex
		}
	case types.Uint16:
		if p.lex.This().Type != tokens.IntLiteral && p.lex.This().Type != tokens.FloatLiteral {
			p.error(p.lex.This(), "uint16: expected int literal", "parseLiteral")
			return ast.ZeroExprIndex
		}

		expr, err = p.ast.NewUint16Literal(p.lex.This())
		if err != nil {
			p.error(p.lex.This(), err.Error(), "parseLiteral")
			return ast.ZeroExprIndex
		}
	case types.Uint32:
		if p.lex.This().Type != tokens.IntLiteral && p.lex.This().Type != tokens.FloatLiteral {
			p.error(p.lex.This(), "uint32: expected int literal", "parseLiteral")
			return ast.ZeroExprIndex
		}

		expr, err = p.ast.NewUint32Literal(p.lex.This())
		if err != nil {
			p.error(p.lex.This(), err.Error(), "parseLiteral")
			return ast.ZeroExprIndex
		}
	case types.Uint64:
		if p.lex.This().Type != tokens.IntLiteral && p.lex.This().Type != tokens.FloatLiteral {
			p.error(p.lex.This(), "uint64: expected int literal", "parseLiteral")
			return ast.ZeroExprIndex
		}

		expr, err = p.ast.NewUint64Literal(p.lex.This())
		if err != nil {
			p.error(p.lex.This(), err.Error(), "parseLiteral")
			return ast.ZeroExprIndex
		}
	case types.Uint128:
		if p.lex.This().Type != tokens.IntLiteral && p.lex.This().Type != tokens.FloatLiteral {
			p.error(p.lex.This(), "uint128: expected int literal", "parseLiteral")
			return ast.ZeroExprIndex
		}

		expr, err = p.ast.NewUint128Literal(p.lex.This())
		if err != nil {
			p.error(p.lex.This(), err.Error(), "parseLiteral")
			return ast.ZeroExprIndex
		}
	case types.UTF8:
		if p.lex.This().Type != tokens.StringLiteral {
			p.error(p.lex.This(), "utf8: expected string literal", "parseLiteral")
			return ast.ZeroExprIndex
		}

		expr = p.ast.NewUTF8Literal(p.lex.This())
	default:
		p.error(p.lex.This(), "unsupported type: "+tokenType.String(), "parseLiteral")
		return ast.ZeroExprIndex
	}

	p.lex.Step() // consume literal

	return expr
}
