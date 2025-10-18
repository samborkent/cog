package parser

import (
	"context"

	"github.com/samborkent/cog/internal/ast"
	"github.com/samborkent/cog/internal/tokens"
	"github.com/samborkent/cog/internal/types"
)

func (p *Parser) parseParameters(ctx context.Context) []*ast.Parameter {
	if p.this().Type != tokens.LParen {
		p.error(p.this(), "expected '(' after procedure identifier", "parseParameters")
		return nil
	}

	p.advance("parseParameters (") // consume '('

	params := make([]*ast.Parameter, 0)

tokenLoop:
	for ; p.this().Type != tokens.RParen && p.this().Type != tokens.EOF; p.advance("parseParameters loop") {
		if ctx.Err() != nil {
			return nil
		}

		switch p.this().Type {
		case tokens.Identifier:
			ident := &ast.Identifier{
				Token: p.this(),
				Name:  p.this().Literal,
			}

			p.advance("parseParameters loop identifier") // consume identifier

			if p.this().Type != tokens.Colon {
				p.error(p.this(), "expected ':' after parameter identifier", "parseParameters")
				continue tokenLoop
			}

			p.advance("parseParameters loop :") // consume ':'

			identType, ok := types.Lookup[p.this().Type]
			if !ok {
				p.error(p.this(), "unknown parameter type", "parseParameters")
				continue tokenLoop
			}

			ident.ValueType = identType

			// TODO: ensure we want to make function parameters always constant (read-only)
			p.symbols.Define(ident, SymbolKindConstant)

			param := &ast.Parameter{
				Identifier: ident,
			}

			p.advance("parseParameters loop type") // consume type token

			if p.this().Type == tokens.Assign {
				// Default parameter value assignment
				p.advance("parseParameters loop =") // consume '='

				expr := p.expression(ctx, identType)
				if expr != nil {
					param.Default = expr
				}
			}

			params = append(params, param)

			switch p.this().Type {
			case tokens.RParen, tokens.EOF:
				break tokenLoop
			case tokens.Comma:
				p.advance("parseParameters loop ,") // consume ','
			default:
				p.error(p.this(), "unexpected token found in parameter list", "parseParameters")
			}
		default:
			p.error(p.this(), "expected parameter identifier", "parseParameters")
			continue tokenLoop
		}
	}

	p.advance("parseParameters )") // consume ')'

	return params
}
