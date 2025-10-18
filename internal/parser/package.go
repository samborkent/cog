package parser

import (
	"github.com/samborkent/cog/internal/ast"
	"github.com/samborkent/cog/internal/tokens"
	"github.com/samborkent/cog/internal/types"
)

func (p *Parser) parsePackage() *ast.Package {
	node := &ast.Package{
		Token: p.this(),
	}

	p.advance("parsePackage") // consume package

	if p.this().Type != tokens.Identifier {
		p.error(p.this(), "missing package identifier", "parsePackage")
		return nil
	}

	node.Identifier = &ast.Identifier{
		Token:     p.this(),
		Name:      p.this().Literal,
		ValueType: types.None,
	}

	p.advance("parsePackage identifier") // consume identifier

	return node
}
