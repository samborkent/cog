package parser

import (
	"context"

	"github.com/samborkent/cog/internal/ast"
	"github.com/samborkent/cog/internal/tokens"
	"github.com/samborkent/cog/internal/types"
)

func (p *Parser) parseParameters(ctx context.Context, procedure, returnParams bool) []*ast.Parameter {
	params := make([]*ast.Parameter, 0)

	for i := 0; !p.match(tokens.RParen, tokens.EOF); i++ {
		if ctx.Err() != nil {
			return nil
		}

		var ident *ast.Identifier

		if !returnParams && p.this().Type != tokens.Identifier {
			p.error(p.this(), "expected parameter identifier", "parseParameters")
			return nil
		}

		var optional bool

		if !returnParams || (returnParams && p.this().Type == tokens.Identifier) {
			ident = &ast.Identifier{
				Token: p.this(),
				Name:  p.this().Literal,
			}

			if (!procedure && ident.Name == "ctx") || (returnParams && ident.Name == "ctx") {
				p.error(p.this(), "ctx is a reserved name for the first context input parameter of procedures", "parseParameters")
				return nil
			}

			p.advance("parseParameters loop identifier") // consume identifier

			if returnParams {
				if p.this().Type != tokens.Colon {
					p.error(p.this(), "expected ':' after return parameter identifier", "parseParameters")
					return nil
				}
			} else {
				if !p.match(tokens.Colon, tokens.Optional) {
					p.error(p.this(), "expected ':' or ':?' after input parameter identifier", "parseParameters")
					return nil
				}

				optional = p.this().Type == tokens.Optional
			}

			p.advance("parseParameters loop : / :?") // consume : or :?
		}

		identType := p.parseCombinedType(ctx, false)
		if identType == nil {
			p.error(p.this(), "unknown parameter type", "parseParameters")
			return nil
		}

		if ident != nil {
			if !returnParams && (ident.Name == "ctx" && identType.Kind() != types.Context) {
				p.error(p.this(), "ctx is a reserved name for the first context input parameter of procedures", "parseParameters")
				return nil
			} else if identType.Kind() == types.Context && (!procedure || ident.Name != "ctx") {
				p.error(p.this(), "context type may only be used for the first context input parameter of procedures named ctx", "parseParameters")
				return nil
			}

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

			if !optional {
				p.error(p.this(), "default values are only allowed for optional input parameters", "parseParameters")
				return nil
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
