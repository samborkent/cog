package parser

import (
	"context"

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

	loop := p.parseBlockStatement(ctx)
	if loop == nil {
		p.error(p.this(), "unable to parse for block", "parseIfStatement")
		return nil
	}

	node.Loop = loop

	return node
}
