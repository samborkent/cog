package parser

import (
	"context"

	"github.com/samborkent/cog/internal/ast"
	"github.com/samborkent/cog/internal/tokens"
)

func (p *Parser) parseConstant(ctx context.Context, exported bool) *ast.Declaration {
	p.advance("parseConstant const") // consume 'const'

	if p.this().Type != tokens.Identifier {
		p.error(p.this(), "constant declaration is missing identifier", "parseStatement")
		return nil
	}

	ident := &ast.Identifier{
		Token:    p.this(),
		Name:     p.this().Literal,
		Exported: exported,
	}

	p.advance("parseConstant const ident") // consume identifier

	switch p.this().Type {
	case tokens.Colon:
		p.advance("parseConstant const :") // consume ':'

		decl := p.parseTypedDeclaration(ctx, ident, true)
		if decl == nil {
			return nil
		}

		return decl
	case tokens.Declaration:
		decl := p.parseDeclaration(ctx, ident, true)
		if decl == nil {
			return nil
		}

		return decl
	default:
		p.error(p.this(), "unexpected token found after constant declaration", "parseStatement")
		return nil
	}
}
