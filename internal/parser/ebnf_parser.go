package parser

import (
	"context"
	"fmt"

	"github.com/samborkent/cog/internal/ast"
	"github.com/samborkent/cog/internal/tokens"
	"github.com/samborkent/cog/internal/types"
)

func (p *Parser) expression(ctx context.Context, typeToken types.Type) ast.ExprIndex {
	expr := p.boolean(ctx, typeToken)

	for p.match(tokens.LBracket) {
		if ctx.Err() != nil || expr == ast.ZeroExprIndex {
			return ast.ZeroExprIndex
		}

		operator := p.this()
		p.advance("expression [") // consume [

		index := p.boolean(ctx, types.None)

		if p.this().Type != tokens.RBracket {
			p.error(p.this(), "expected ] after index expression", "expression")
			return ast.ZeroExprIndex
		}

		p.advance("expression ]") // consume ]

		exprType := p.ast.Expr(expr).Type()

		if !types.IsIndexable(exprType) {
			p.error(p.this(), fmt.Sprintf("type %q is not indexable", exprType), "expression")
			return ast.ZeroExprIndex
		}

		// Allocate index expression.
		indexExpr := ast.New[ast.Index](p.ast)
		indexExpr.Token = operator
		indexExpr.Expr = expr
		indexExpr.Index = index

		switch t := exprType.Underlying().(type) {
		case *types.Array:
			indexExpr.ElemType = t.Element
		case *types.Slice:
			indexExpr.ElemType = t.Element
		case *types.Map:
			indexExpr.ElemType = t.Value
		case *types.Set:
			indexExpr.ElemType = t.Element
		case *types.Basic:
			if exprType.Kind() == types.ASCII {
				indexExpr.ElemType = types.Basics[types.Uint8]
			} else if exprType.Kind() == types.UTF8 {
				// TODO: implement rune type and string indexing
				indexExpr.ElemType = types.Basics[types.Int32]
			} else {
				panic(fmt.Sprintf("unexpected basic type that is indexable but not string: %v", exprType))
			}
		default:
			panic(fmt.Sprintf("unexpected indexable type that is not array, slice, map, set, or string: %v", exprType))
		}

		expr = p.ast.AddExpr(indexExpr)
	}

	return expr
}

func (p *Parser) boolean(ctx context.Context, typeToken types.Type) ast.ExprIndex {
	expr := p.equality(ctx, typeToken)

	for p.match(tokens.And, tokens.Or) {
		if ctx.Err() != nil || expr == ast.ZeroExprIndex {
			return ast.ZeroExprIndex
		}

		operator := p.this()
		p.advance("boolean operator") // consume operator
		right := p.equality(ctx, types.Basics[types.Bool])

		if !types.IsBool(p.ast.Expr(expr).Type()) {
			p.error(p.this(), "operator requires bool type", "boolean")
			return ast.ZeroExprIndex
		}

		// Allocate infix expression.
		expr = p.ast.NewInfix(operator, types.Basics[types.Bool], expr, right)
	}

	return expr
}

func (p *Parser) equality(ctx context.Context, typeToken types.Type) ast.ExprIndex {
	expr := p.comparison(ctx, typeToken)

	for p.match(tokens.Equal, tokens.NotEqual) {
		if ctx.Err() != nil || expr == ast.ZeroExprIndex {
			return ast.ZeroExprIndex
		}

		operator := p.this()
		p.advance("equality operator") // consume operator
		rightIndex := p.comparison(ctx, types.None)

		left := p.ast.Expr(expr)
		right := p.ast.Expr(rightIndex)

		// TODO: do we need to equalize for equality?
		if left.Type().Kind() != right.Type().Kind() {
			ast.EqualizeInfixTypes(left, right)
		}

		expr = p.ast.NewInfix(operator, types.Basics[types.Bool], expr, rightIndex)
	}

	return expr
}

func (p *Parser) comparison(ctx context.Context, typeToken types.Type) ast.ExprIndex {
	expr := p.term(ctx, typeToken)

	for p.match(tokens.GT, tokens.GTEqual, tokens.LT, tokens.LTEqual) {
		if ctx.Err() != nil || expr == ast.ZeroExprIndex {
			return ast.ZeroExprIndex
		}

		operator := p.this()
		p.advance("comparison operator") // consume operator
		// TODO: should we pass expr.Type()?
		rightIndex := p.term(ctx, types.None)

		if !types.IsNumber(p.ast.Expr(expr).Type()) {
			p.error(p.this(), "operator requires numeric type", "comparison")
			return ast.ZeroExprIndex
		}

		left := p.ast.Expr(expr)
		right := p.ast.Expr(rightIndex)

		// TODO: do we need to equalize for comparison?
		if left.Type().Kind() != right.Type().Kind() {
			ast.EqualizeInfixTypes(left, right)
		}

		expr = p.ast.NewInfix(operator, types.Basics[types.Bool], expr, rightIndex)
	}

	return expr
}

func (p *Parser) term(ctx context.Context, typeToken types.Type) ast.ExprIndex {
	expr := p.factor(ctx, typeToken)

	for p.match(tokens.Minus, tokens.Plus) {
		if ctx.Err() != nil || expr == ast.ZeroExprIndex {
			return ast.ZeroExprIndex
		}

		operator := p.this()
		p.advance("term operator") // consume operator

		exprType := p.ast.Expr(expr).Type()
		right := p.factor(ctx, exprType)

		// TODO: this is a hack due to lack of known Go typing at compile time, figure out a better solution.
		if exprType != types.None {
			if operator.Type == tokens.Plus {
				if !types.IsSummable(exprType) {
					p.error(p.this(), fmt.Sprintf("operator requires numeric or string type, got %q", exprType), "term")
					return ast.ZeroExprIndex
				}
			} else {
				// Minus
				if !types.IsNumber(exprType) {
					p.error(p.this(), fmt.Sprintf("operator requires numeric type, got %q", exprType), "term")
					return ast.ZeroExprIndex
				}
			}
		}

		expr = p.ast.NewInfix(operator, exprType, expr, right)
	}

	return expr
}

func (p *Parser) factor(ctx context.Context, typeToken types.Type) ast.ExprIndex {
	expr := p.unary(ctx, typeToken)

	for p.match(tokens.Asterisk, tokens.Divide) {
		if ctx.Err() != nil || expr == ast.ZeroExprIndex {
			return ast.ZeroExprIndex
		}

		operator := p.this()
		p.advance("factor operator") // consume operator

		exprType := p.ast.Expr(expr).Type()
		right := p.unary(ctx, exprType)

		if !types.IsNumber(exprType) {
			p.error(p.this(), "operator requires numeric type", "factor")
			return ast.ZeroExprIndex
		}

		expr = p.ast.NewInfix(operator, exprType, expr, right)
	}

	return expr
}

func (p *Parser) unary(ctx context.Context, typeToken types.Type) ast.ExprIndex {
	if p.match(tokens.Not, tokens.Minus, tokens.BitAnd) {
		// Previous operator is stored, to disallow double references.
		prevOperator := p.prev()
		if prevOperator.Type == tokens.LParen && p.i >= 2 && p.tokens[p.i-2].Type == tokens.BitAnd {
			prevOperator = p.tokens[p.i-2]
		}

		operator := p.this()
		p.advance("unary operator") // consume operator

		exprType := typeToken

		if operator.Type == tokens.BitAnd {
			// Special reference handling.
			if prevOperator.Type == tokens.BitAnd {
				p.error(p.this(), "double reference is not allowed", "unary")
				return ast.ZeroExprIndex
			}

			if typeToken != types.None && typeToken.Kind() == types.ReferenceKind {
				// If a type is specified, we need to pass the reference underlying type to the expression parsing.
				refType, ok := typeToken.(*types.Reference)
				if !ok {
					p.error(p.this(), "unable to assert reference type", "unary")
					return ast.ZeroExprIndex
				}

				exprType = refType.Value
			}
		}

		right := p.unary(ctx, exprType)
		if right == ast.ZeroExprIndex {
			return ast.ZeroExprIndex
		}

		rightType := p.ast.Expr(right).Type()

		if operator.Type == tokens.Not && !types.IsBool(rightType) {
			p.error(p.this(), "operator requires bool type", "unary")
			return ast.ZeroExprIndex
		} else if operator.Type == tokens.Minus && !types.IsSigned(rightType) {
			p.error(p.this(), "operator requires signed numeric type", "unary")
			return ast.ZeroExprIndex
		}

		return p.ast.NewPrefix(operator, rightType, right)
	}

	if (typeToken == nil || typeToken == types.None) && p.this().Type == tokens.Identifier {
		// TODO: get rid of double lookup for identifiers
		symbol, ok := p.symbols.Resolve(p.this().Literal)
		if !ok {
			// If this is an imported package name, skip the type pre-lookup;
			// primary() will handle it via parsePkgSelector.
			if _, isImport := p.symbols.ResolveCogImport(p.this().Literal); !isImport {
				p.error(p.this(), "undefined identifier", "primary")
				return ast.ZeroExprIndex
			}
		} else {
			typeToken = symbol.Type()
		}
	}

	expr := p.primary(ctx, typeToken)
	if expr == ast.ZeroExprIndex {
		return ast.ZeroExprIndex
	}

	if p.this().Type == tokens.Question {
		token := p.this()
		p.advance("unary ?") // consume ?

		// ? works on both option and result types.
		if typeToken.Kind() != types.OptionKind && typeToken.Kind() != types.ResultKind {
			p.error(token, "? operator requires option or result type", "unary")
			return ast.ZeroExprIndex
		}

		return p.ast.NewSuffix(token, typeToken, expr)
	}

	if p.this().Type == tokens.Not {
		token := p.this()
		p.advance("unary !") // consume !

		if typeToken.Kind() != types.ResultKind {
			p.error(token, "! operator requires result type", "unary")
			return ast.ZeroExprIndex
		}

		// Must-check: cannot extract error without checking ? first.
		if ident, ok := p.ast.Expr(expr).(*ast.Identifier); ok {
			if !p.symbols.IsErrorChecked(ident.Name) {
				p.error(ident.Token, "must check "+ident.Name+" before accessing error", "unary")
				return ast.ZeroExprIndex
			}
		}

		return p.ast.NewSuffix(token, typeToken, expr)
	}

	// Must-check analysis: bare access to option/result requires prior ? check.
	if ident, ok := p.ast.Expr(expr).(*ast.Identifier); ok {
		kind := typeToken.Kind()

		if (kind == types.OptionKind || kind == types.ResultKind) && !p.symbols.IsValueChecked(ident.Name) {
			p.error(ident.Token, "must check "+ident.Name+" before accessing value", "unary")
			return ast.ZeroExprIndex
		}
	}

	return expr
}
