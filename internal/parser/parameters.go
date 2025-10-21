package parser

import (
	"context"

	"github.com/samborkent/cog/internal/ast"
	"github.com/samborkent/cog/internal/tokens"
	"github.com/samborkent/cog/internal/types"
)

func (p *Parser) parseParameters(ctx context.Context, procedure bool) []*ast.Parameter {
	params := make([]*ast.Parameter, 0)

	// Flag to keep track of any of the parameters is optional.
	// When a parameter is marked as optional, all following parameters must also be optional.
	haveOptional := false

	for i := 0; !p.match(tokens.RParen, tokens.EOF); i++ {
		if ctx.Err() != nil {
			return nil
		}

		var ident *ast.Identifier

		if p.this().Type != tokens.Identifier {
			p.error(p.this(), "expected parameter identifier", "parseParameters")
			return nil
		}

		var optional bool

		ident = &ast.Identifier{
			Token: p.this(),
			Name:  p.this().Literal,
		}

		if ident.Name == "ctx" && (!procedure || i > 0) {
			p.error(p.this(), "'ctx' identifier is reserved for the first input parameter of procedures", "parseParameters")
			return nil
		}

		p.advance("parseParameters loop identifier") // consume identifier

		if p.this().Type == tokens.Question {
			optional = true
			haveOptional = true

			p.advance("parseParameters loop ?") // consume ?
		} else if haveOptional {
			// This parameter is not optional, but a previous parameter was, this is not allowed.
			p.error(p.prev(), "all input parameters following an optional parameter must also be optional", "parseParameters")
			return nil
		}

		if p.this().Type != tokens.Colon {
			p.error(p.this(), "expected ':' after return parameter identifier", "parseParameters")
			return nil
		}

		p.advance("parseParameters loop :") // consume :

		identType := p.parseCombinedType(ctx, false)
		if identType == nil {
			p.error(p.this(), "unknown parameter type", "parseParameters")
			return nil
		}

		if ident.Name == "ctx" && identType.Kind() != types.Context {
			p.error(p.this(), "input parameter 'ctx' must be of type 'context'", "parseParameters")
			return nil
		} else if identType.Kind() == types.Context && (!procedure || ident.Name != "ctx") {
			p.error(p.this(), "context type may only be used as first input parameter of procedures", "parseParameters")
			return nil
		}

		ident.ValueType = identType

		// TODO: ensure we want to make function parameters always constant (read-only)
		p.symbols.Define(ident, SymbolKindConstant)

		param := &ast.Parameter{
			Identifier: ident,
			ValueType:  identType,
			Optional:   optional,
		}

		if p.this().Type == tokens.Assign {
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
