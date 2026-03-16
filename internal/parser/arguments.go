package parser

import (
	"context"

	"github.com/samborkent/cog/internal/tokens"
	"github.com/samborkent/cog/internal/types"
)

func (p *Parser) parseTypeArguments(ctx context.Context) []types.Type {
	if p.this().Type != tokens.LT {
		p.error(p.this(), "expected opening < for type arguments", "parseTypeArguments")
		return nil
	}

	p.advance("parseTypeArguments <") // consume <

	typ := p.parseCombinedType(ctx, false)
	if typ == nil {
		return nil
	}

	args := []types.Type{typ}

	for p.this().Type == tokens.Comma {
		p.advance("parseTypeArguments ,") // consume ,

		typ := p.parseCombinedType(ctx, false)
		if typ == nil {
			return nil
		}

		args = append(args, typ)
	}

	if p.this().Type != tokens.GT {
		p.error(p.this(), "expected closing > for type arguments", "parseTypeArguments")
		return nil
	}

	p.advance("parseTypeArguments >") // consume >

	return args
}
