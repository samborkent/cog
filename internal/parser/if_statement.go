package parser

import (
	"context"

	"github.com/samborkent/cog/internal/ast"
	"github.com/samborkent/cog/internal/tokens"
	"github.com/samborkent/cog/internal/types"
)

func (p *Parser) parseIfStatement(ctx context.Context) *ast.IfStatement {
	node := &ast.IfStatement{
		Token: p.this(),
	}

	p.advance("parseIfStatement if") // consume if

	expr := p.expression(ctx, types.None)
	if expr == nil {
		p.error(p.this(), "unable to parse bool expression in if condition", "parseIfStatement")
		return nil
	}

	if p.prev().Type != tokens.Question && expr.Type().Kind() != types.Bool {
		p.error(p.this(), "expected option or bool expression in if condition", "parseIfStatement")
		return nil
	}

	node.Condition = expr

	consequence := p.parseBlock(ctx)
	if consequence == nil {
		p.error(p.this(), "unable to parse if block", "parseIfStatement")
		return nil
	}

	node.Consequence = consequence

	if p.this().Type == tokens.Else {
		if ctx.Err() != nil {
			return nil
		}

		p.advance("parseIfStatement else") // consume 'else'

		alternative := p.parseBlock(ctx)
		if alternative == nil {
			p.error(p.this(), "unable to parse else block", "parseIfStatement")
			return nil
		}

		node.Alternative = alternative
	}

	return node
}
