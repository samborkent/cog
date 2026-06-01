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

	for p.lex.This().Type == tokens.LBracket {
		if ctx.Err() != nil || expr == ast.ZeroExprIndex {
			return ast.ZeroExprIndex
		}

		operator := p.lex.This()
		p.lex.Step() // consume [

		index := p.boolean(ctx, types.None)

		if p.lex.This().Type != tokens.RBracket {
			p.error(p.lex.This(), "expected ] after index expression", "expression")
			return ast.ZeroExprIndex
		}

		p.lex.Step() // consume ]

		exprType := p.ast.Expr(expr).Type()

		if !types.IsIndexable(exprType) {
			p.error(p.lex.This(), fmt.Sprintf("type %q is not indexable", exprType), "expression")
			return ast.ZeroExprIndex
		}

		// Rule 22: Index expressions borrow the container.
		if exprIdent, ok := p.ast.Expr(expr).(*ast.Identifier); ok &&
			exprIdent.Qualifier == ast.QualifierVariable {
			p.symbols.MarkBorrowed(exprIdent.Token.Literal)
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

		// Rule 22: Release borrow after index expression.
		if exprIdent, ok := p.ast.Expr(expr).(*ast.Identifier); ok &&
			exprIdent.Qualifier == ast.QualifierVariable {
			p.symbols.ReleaseBorrowed(exprIdent.Token.Literal)
		}

		expr = p.ast.AddExpr(indexExpr)
	}

	return expr
}

func (p *Parser) boolean(ctx context.Context, typeToken types.Type) ast.ExprIndex {
	expr := p.equality(ctx, typeToken)

	for p.match(p.lex.This(), tokens.And, tokens.Or) {
		operator := p.lex.This()
		p.lex.Step() // consume operator
		right := p.equality(ctx, types.Basics[types.Bool])

		if !types.IsBool(p.ast.Expr(expr).Type()) {
			p.error(p.lex.This(), "operator requires bool type", "boolean")
			return ast.ZeroExprIndex
		}

		// Allocate infix expression.
		expr = p.ast.NewInfix(operator, types.Basics[types.Bool], expr, right)
	}

	return expr
}

func (p *Parser) equality(ctx context.Context, typeToken types.Type) ast.ExprIndex {
	expr := p.comparison(ctx, typeToken)

	for p.match(p.lex.This(), tokens.Equal, tokens.NotEqual) {
		left := p.ast.Expr(expr)

		operator := p.lex.This()
		p.lex.Step() // consume operator
		rightIndex := p.comparison(ctx, types.None)

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

	for p.match(p.lex.This(), tokens.GT, tokens.GTEqual, tokens.LT, tokens.LTEqual) {
		operator := p.lex.This()
		p.lex.Step() // consume operator
		// TODO: should we pass expr.Type()?
		rightIndex := p.term(ctx, types.None)

		if !types.IsNumber(p.ast.Expr(expr).Type()) {
			p.error(operator, "operator requires numeric type", "comparison")
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
	if expr == ast.ZeroExprIndex {
		return ast.ZeroExprIndex
	}

	for p.match(p.lex.This(), tokens.Minus, tokens.Plus) {
		operator := p.lex.This()
		p.lex.Step() // consume operator

		exprType := p.ast.Expr(expr).Type()
		right := p.factor(ctx, exprType)

		// TODO: this is a hack due to lack of known Go typing at compile time, figure out a better solution.
		if exprType != types.None {
			if operator.Type == tokens.Plus {
				if !types.IsSummable(exprType) {
					p.error(operator, fmt.Sprintf("operator requires numeric or string type, got %q", exprType), "term")
					return ast.ZeroExprIndex
				}
			} else {
				// Minus
				if !types.IsNumber(exprType) {
					p.error(operator, fmt.Sprintf("operator requires numeric type, got %q", exprType), "term")
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

	for p.match(p.lex.This(), tokens.Asterisk, tokens.Divide) {
		operator := p.lex.This()
		p.lex.Step() // consume operator

		exprType := p.ast.Expr(expr).Type()
		right := p.unary(ctx, exprType)

		if !types.IsNumber(exprType) {
			p.error(operator, "operator requires numeric type", "factor")
			return ast.ZeroExprIndex
		}

		expr = p.ast.NewInfix(operator, exprType, expr, right)
	}

	return expr
}

func (p *Parser) unary(ctx context.Context, typeToken types.Type) ast.ExprIndex {
	if p.match(p.lex.This(), tokens.Not, tokens.Minus, tokens.BitAnd, tokens.Asterisk) {
		// Previous operator is stored, to disallow double references.
		prevOperator := p.lex.Peek(-1)
		if prevOperator.Type == tokens.LParen && p.lex.Peek(-2).Type == tokens.BitAnd {
			prevOperator = p.lex.Peek(-2)
		}

		operator := p.lex.This()
		p.lex.Step() // consume operator

		exprType := typeToken

		if operator.Type == tokens.BitAnd {
			// Special reference handling.
			if prevOperator.Type == tokens.BitAnd {
				p.error(operator, "double reference is not allowed", "unary")
				return ast.ZeroExprIndex
			}

			if typeToken != types.None && typeToken.Kind() == types.ReferenceKind {
				// If a type is specified, we need to pass the reference underlying type to the expression parsing.
				refType, ok := typeToken.(*types.Reference)
				if !ok {
					p.error(p.lex.This(), "unable to assert reference type", "unary")
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
			p.error(operator, "operator requires bool type", "unary")
			return ast.ZeroExprIndex
		} else if operator.Type == tokens.Minus && !types.IsSigned(rightType) {
			p.error(operator, "operator requires signed numeric type", "unary")
			return ast.ZeroExprIndex
		} else if operator.Type == tokens.Asterisk && rightType.Kind() != types.ReferenceKind {
			p.error(operator, "dereference requires reference type", "unary")
			return ast.ZeroExprIndex
		} else if operator.Type == tokens.BitAnd {
			// Rule 2: & consumes var variable.
			if ident, ok := p.ast.Expr(right).(*ast.Identifier); ok && ident.Qualifier == ast.QualifierVariable {
				if !p.symbols.IsAlive(ident.Token.Literal) {
					p.error(operator, fmt.Sprintf("cannot take reference of %s: variable is consumed", ident.Token.Literal), "unary")
					return ast.ZeroExprIndex
				}
				p.symbols.MarkConsumed(ident.Token.Literal)
			}
		} else if operator.Type == tokens.Asterisk {
			// Rule 3: * deref consumes var & reference.
			if ident, ok := p.ast.Expr(right).(*ast.Identifier); ok && ident.Qualifier == ast.QualifierVariable {
				if !p.symbols.IsAlive(ident.Token.Literal) {
					p.error(operator, fmt.Sprintf("cannot dereference %s: variable is consumed", ident.Token.Literal), "unary")
					return ast.ZeroExprIndex
				}
				p.symbols.MarkConsumed(ident.Token.Literal)
			}
		}

		return p.ast.NewPrefix(operator, rightType, right)
	}

	if (typeToken == nil || typeToken == types.None) && p.lex.This().Type == tokens.Identifier {
		// TODO: get rid of double lookup for identifiers
		symbol, ok := p.symbols.Resolve(p.lex.This().Literal)
		if !ok {
			// If this is an imported package name, skip the type pre-lookup;
			// primary() will handle it via parsePkgSelector.
			if _, isImport := p.symbols.ResolveCogImport(p.lex.This().Literal); !isImport {
				p.error(p.lex.This(), "undefined identifier", "primary")
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

	if p.lex.This().Type == tokens.Question {
		token := p.lex.This()
		p.lex.Step() // consume ?

		// ? works on both option and result types.
		if typeToken.Kind() != types.OptionKind && typeToken.Kind() != types.ResultKind {
			p.error(token, "? operator requires option or result type", "unary")
			return ast.ZeroExprIndex
		}

		return p.ast.NewSuffix(token, typeToken, expr)
	}

	if p.lex.This().Type == tokens.Not {
		token := p.lex.This()
		p.lex.Step() // consume !

		if typeToken.Kind() != types.ResultKind {
			p.error(token, "! operator requires result type", "unary")
			return ast.ZeroExprIndex
		}

		// Must-check: cannot extract error without checking ? first.
		if ident, ok := p.ast.Expr(expr).(*ast.Identifier); ok {
			if !p.symbols.IsErrorChecked(ident.Token.Literal) {
				p.error(ident.Token, "must check "+ident.Token.Literal+" before accessing error", "unary")
				return ast.ZeroExprIndex
			}
		}

		return p.ast.NewSuffix(token, typeToken, expr)
	}

	if ident, ok := p.ast.Expr(expr).(*ast.Identifier); ok {
		kind := typeToken.Kind()

		// Must-check analysis: bare access to option/result requires prior ? check.
		if (kind == types.OptionKind || kind == types.ResultKind) && !p.symbols.IsValueChecked(ident.Token.Literal) {
			p.error(ident.Token, "must check "+ident.Token.Literal+" before accessing value", "unary")
			return ast.ZeroExprIndex
		}

		// Consumed variable check: use of a consumed var is an error.
		if ident.Qualifier == ast.QualifierVariable && !p.symbols.IsAlive(ident.Token.Literal) {
			p.error(ident.Token, fmt.Sprintf("cannot use %s: variable is consumed", ident.Token.Literal), "unary")
			return ast.ZeroExprIndex
		}
	}

	return expr
}
