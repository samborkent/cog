package parser

import (
	"context"
	"strings"

	"github.com/samborkent/cog/internal/ast"
	"github.com/samborkent/cog/internal/tokens"
	"github.com/samborkent/cog/internal/types"
)

func (p *Parser) parseForStatement(ctx context.Context) ast.NodeIndex {
	var label *ast.Identifier

	prev := p.lex.Peek(-1)

	if prev.Type == tokens.Identifier && p.lex.This().Type == tokens.Colon {
		label = &ast.Identifier{
			Token: prev,
		}

		p.lex.Step() // consume colon
	}

	forToken := p.lex.This()

	p.lex.Step() // consume for

	var (
		valueVar  *ast.Identifier
		indexVar  *ast.Identifier
		rangeExpr ast.ExprIndex
	)

	// TODO: add support for value and index variables

	switch p.lex.This().Type {
	case tokens.LBrace:
		// Infinite loop, no range.
	default:
		switch p.lex.Peek(1).Type {
		case tokens.In:
			if p.lex.This().Type != tokens.Identifier {
				p.error(p.lex.This(), "expected identifier for loop variable", "parseForStatement")
				return ast.ZeroNodeIndex
			}

			valueVar = &ast.Identifier{
				Token:     p.lex.This(),
				Qualifier: ast.QualifierImmutable,
			}

			p.lex.Step() // consume value variable
			p.lex.Step() // consume in keyword
		case tokens.Comma:
			if p.lex.This().Type != tokens.Identifier {
				p.error(p.lex.This(), "expected identifier for loop value variable", "parseForStatement")
				return ast.ZeroNodeIndex
			}

			// Skip _ value variable.
			if p.lex.This().Literal != "_" {
				valueVar = &ast.Identifier{
					Token:     p.lex.This(),
					Qualifier: ast.QualifierImmutable,
				}
			}

			p.lex.Step() // consume value variable
			p.lex.Step() // consume ,

			if p.lex.This().Type != tokens.Identifier {
				p.error(p.lex.This(), "expected identifier for loop index variable", "parseForStatement")
				return ast.ZeroNodeIndex
			}

			indexVar = &ast.Identifier{
				Token:     p.lex.This(),
				ValueType: types.Basics[types.Uint64],
				Qualifier: ast.QualifierImmutable,
			}

			p.lex.Step() // consume index

			if p.lex.This().Type != tokens.In {
				p.error(p.lex.This(), "expected in keyword after loop index variable", "parseForStatement")
				return ast.ZeroNodeIndex
			}

			p.lex.Step() // consume in keyword
		}

		expr := p.expression(ctx, types.None)
		if expr == ast.ZeroExprIndex {
			p.error(p.lex.This(), "expected range expression or loop body", "parseForStatement")
			return ast.ZeroNodeIndex
		}

		exprType := p.ast.Expr(expr).Type()

		if !types.IsIterator(exprType) {
			p.error(p.lex.This(), "cannot iterate over type "+exprType.String(), "parseForStatement")
			return ast.ZeroNodeIndex
		}

		if valueVar != nil {
			valueVar.ValueType = exprType
		}

		rangeExpr = expr
	}

	if valueVar != nil || indexVar != nil {
		// Add value variable to scope.
		p.symbols = NewEnclosedSymbolTable(p.symbols)

		if valueVar != nil {
			p.symbols.Define(valueVar)
			p.symbols.MarkUsed(valueVar.Token.Literal)
		}

		if indexVar != nil {
			p.symbols.Define(indexVar)
			p.symbols.MarkUsed(indexVar.Token.Literal)
		}
	}

	prevErrorCount := len(p.Errs)

	loop := p.parseBlockStatement(ctx)
	if loop == nil {
		p.error(p.lex.This(), "unable to parse for block", "parseForStatement")
		return ast.ZeroNodeIndex
	}

	if valueVar != nil || indexVar != nil {
		// Check for unused variables in the for scope.
		p.checkUnused()
		// Restore scope.
		p.symbols = p.symbols.Outer
	}

	// Logic for specific error when a untyped container literal is passed in loop range expression.
	if len(p.Errs) > prevErrorCount {
		for _, err := range p.Errs[prevErrorCount:] {
			if strings.Contains(err.Error(), "unknown token") {
				p.error(p.lex.This(), "untyped container literal not allowed in loop range expression", "parseIfStatement")
				return ast.ZeroNodeIndex
			}
		}
	}

	return p.ast.NewForStatement(forToken, label, valueVar, indexVar, rangeExpr, loop)
}
