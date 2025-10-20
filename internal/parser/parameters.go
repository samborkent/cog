package parser

import (
	"context"

	"github.com/samborkent/cog/internal/ast"
	"github.com/samborkent/cog/internal/tokens"
)

func (p *Parser) parseParameters(ctx context.Context, returnParams bool) []*ast.Parameter {
	params := make([]*ast.Parameter, 0)

	for !p.match(tokens.RParen, tokens.EOF) {
		if ctx.Err() != nil {
			return nil
		}

		var ident *ast.Identifier

		if !returnParams && p.this().Type != tokens.Identifier {
			p.error(p.this(), "expected parameter identifier", "parseParameters")
			return nil
		}

		if !returnParams || (returnParams && p.this().Type == tokens.Identifier) {
			ident = &ast.Identifier{
				Token: p.this(),
				Name:  p.this().Literal,
			}

			p.advance("parseParameters loop identifier") // consume identifier

			if p.this().Type != tokens.Colon {
				p.error(p.this(), "expected ':' after parameter identifier", "parseParameters")
				return nil
			}

			p.advance("parseParameters loop :") // consume ':'
		}

		identType := p.parseCombinedType(ctx, false)
		if identType == nil {
			p.error(p.this(), "unknown parameter type", "parseParameters")
			return nil
		}

		if ident != nil {
			ident.ValueType = identType

			// TODO: ensure we want to make function parameters always constant (read-only)
			p.symbols.Define(ident, SymbolKindConstant)
		}

		param := &ast.Parameter{
			Identifier: ident,
			ValueType:  identType,
		}

		if p.this().Type == tokens.Assign {
			if returnParams {
				if p.next().Type != tokens.LBrace {
					p.error(p.this(), "return parameters cannot have default values", "parseParameters")
					return nil
				}

				// Only single parameter
				return []*ast.Parameter{param}
			}

			// Default parameter value assignment
			p.advance("parseParameters loop =") // consume '='

			expr := p.expression(ctx, identType)
			if expr != nil {
				param.Default = expr
			}
		}

		params = append(params, param)

		if p.this().Type == tokens.Comma {
			p.advance("parseParameters loop ,") // consume ','
		}
	}

	return params
}
