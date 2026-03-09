package parser

import (
	"context"

	"github.com/samborkent/cog/internal/ast"
)

func (p *Parser) parseForStatement(ctx context.Context) *ast.ForStatement {
	node := &ast.ForStatement{
		Token: p.this(),
	}

	p.advance("parseForStatement for") // consume for

	loop := p.parseBlockStatement(ctx)
	if loop == nil {
		p.error(p.this(), "unable to parse for block", "parseIfStatement")
		return nil
	}

	node.Loop = loop

	return node
}
