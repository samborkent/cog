package parser

import (
	"context"

	"github.com/samborkent/cog/internal/ast"
	"github.com/samborkent/cog/internal/tokens"
)

// TODO: base on heuristics.
const blockPreacclocationSize = 8

func (p *Parser) parseBlockStatement(ctx context.Context) *ast.Block {
	startToken := p.this()
	stmts := make([]ast.NodeIndex, 0, blockPreacclocationSize)

	var endToken tokens.Token

	p.advance("parseBlock") // consume '{'

	// Enter scope.
	p.symbols = NewEnclosedSymbolTable(p.symbols)

	for p.this().Type != tokens.EOF {
		if ctx.Err() != nil {
			return nil
		}

		if p.this().Type == tokens.RBrace {
			endToken = p.this()
			break
		}

		prev := p.i

		stmt := p.parseStatement(ctx)
		if stmt != ast.ZeroNodeIndex {
			stmts = append(stmts, stmt)
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

	block := ast.New[ast.Block](p.ast)
	block.Start = startToken
	block.End = endToken
	block.Statements = stmts

	return block
}
