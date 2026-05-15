package parser

import (
	"github.com/samborkent/cog/internal/ast"
	"github.com/samborkent/cog/internal/tokens"
	"github.com/samborkent/cog/internal/types"
)

func (p *Parser) parsePackage() *ast.Package {
	node := &ast.Package{
		Token: p.lex.This(),
	}

	p.lex.Step() // consume package

	if p.lex.This().Type != tokens.Identifier {
		p.error(p.lex.This(), "missing package identifier", "parsePackage")
		return nil
	}

	node.Identifier = &ast.Identifier{
		Token:     p.lex.This(),
		ValueType: types.None,
	}

	p.lex.Step() // consume identifier

	return node
}
