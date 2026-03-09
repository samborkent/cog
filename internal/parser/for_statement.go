package parser

import (
	"context"
	"strings"

	"github.com/samborkent/cog/internal/ast"
	"github.com/samborkent/cog/internal/tokens"
	"github.com/samborkent/cog/internal/types"
)

func (p *Parser) parseForStatement(ctx context.Context) *ast.ForStatement {
	node := &ast.ForStatement{
		Token: p.this(),
	}

	p.advance("parseForStatement for") // consume for

	// TODO: add support for in keyword
	// TODO: add support for value and index variables
	switch p.this().Type {
	case tokens.LBrace:
		// Infinite loop, no range.
	case tokens.LBracket, tokens.Map, tokens.Set:
		p.error(p.this(), "cannot iterate over container literal, assign to identifier first", "parseForStatement")
		return nil
	default:
		expr := p.expression(ctx, types.None)
		if expr == nil {
			p.error(p.this(), "expected range expression or loop body", "parseForStatement")
			return nil
		}

		if !types.IsIterator(expr.Type()) {
			p.error(p.this(), "cannot iterate over type "+expr.Type().String(), "parseForStatement")
			return nil
		}

		node.Range = expr
	}

	prevErrorCount := len(p.Errs)

	loop := p.parseBlockStatement(ctx)
	if loop == nil {
		p.error(p.this(), "unable to parse for block", "parseIfStatement")
		return nil
	}

	// Logic for specific error when a untyped container literal is passed in loop range expression.
	if len(p.Errs) > prevErrorCount {
		for _, err := range p.Errs[prevErrorCount:] {
			if strings.Contains(err.Error(), "unknown token") {
				p.error(p.this(), "untyped container literal not allowed in loop range expression", "parseIfStatement")
				return nil
			}
		}
	}

	node.Loop = loop

	return node
}
