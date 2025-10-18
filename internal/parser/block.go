package parser

import (
	"context"

	"github.com/samborkent/cog/internal/ast"
	"github.com/samborkent/cog/internal/tokens"
)

func (p *Parser) parseBlock(ctx context.Context) *ast.Block {
	node := &ast.Block{
		Start:      p.this(),
		Statements: make([]ast.Statement, 0),
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

		stmt := p.parseStatement(ctx)
		if stmt != nil {
			node.Statements = append(node.Statements, stmt)
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
