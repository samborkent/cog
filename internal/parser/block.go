package parser

import (
	"context"

	"github.com/samborkent/cog/internal/ast"
	"github.com/samborkent/cog/internal/tokens"
)

// TODO: optimize based on statistics.
const blockSizeEstimate = 8

func (p *Parser) parseBlockStatement(ctx context.Context) *ast.Block {
	node := &ast.Block{
		Start:      p.this(),
		Statements: make([]ast.NodeValue, 0, blockSizeEstimate),
	}

	p.advance("parseBlock") // consume '{'

	// Enter scope.
	p.symbols = NewEnclosedSymbolTable(p.symbols)

	for p.this().Type != tokens.EOF {
		if ctx.Err() != nil {
			return nil
		}

		if p.this().Type == tokens.RBrace {
			node.End = p.this()
			break
		}

		prev := p.i

		stmt := p.parseStatement(ctx)
		if stmt != ast.ZeroNode {
			node.Statements = append(node.Statements, stmt)
		} else {
			// Synchronize to recover from errors within a block.
			p.synchronize()
		}

		// Guard against infinite loops: if no progress was made, force advance.
		if p.i == prev {
			p.advance("parseBlock recovery")
		}
	}

	if p.this().Type != tokens.RBrace {
		p.error(p.this(), "expected '}' to close block", "parseBlock")
		return nil
	}

	p.advance("parseBlock }") // consume '}'

	// Restore scope
	p.symbols = p.symbols.Outer

	return node
}
