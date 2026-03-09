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

	// TODO: add other types
	// TODO: add support for in keyword
	// TODO: add support for value and index variables
	if p.this().Type == tokens.IntLiteral {
		expr := p.parseLiteral(types.None)
		if expr == nil {
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
