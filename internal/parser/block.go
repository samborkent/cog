package parser

import (
	"context"

	"github.com/samborkent/cog/internal/ast"
	"github.com/samborkent/cog/internal/tokens"
)

// TODO: base on heuristics.
const blockPreacclocationSize = 8

func (p *Parser) parseBlockStatement(ctx context.Context) *ast.Block {
	startToken := p.lex.This()
	stmts := make([]ast.NodeIndex, 0, blockPreacclocationSize)

	var endToken tokens.Token

	p.lex.Step() // consume '{'

	// Enter scope.
	p.symbols = NewEnclosedSymbolTable(p.symbols)

	for t := range p.lex.Range(ctx) {
		if t.Type == tokens.RBrace {
			endToken = t
			break
		}

		stmt := p.parseStatement(ctx)
		if stmt != ast.ZeroNodeIndex {
			stmts = append(stmts, stmt)
		} else {
			// Synchronize to recover from errors within a block.
			p.synchronize(ctx)
		}
	}

	if p.lex.This().Type != tokens.RBrace {
		p.error(p.lex.This(), "expected '}' to close block", "parseBlock")
		return nil
	}

	p.lex.Step() // consume '}'

	// Restore scope
	p.symbols = p.symbols.Outer

	block := ast.New[ast.Block](p.ast)
	block.Start = startToken
	block.End = endToken
	block.Statements = stmts

	return block
}
