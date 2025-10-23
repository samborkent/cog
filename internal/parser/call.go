package parser

import (
	"context"

	"github.com/samborkent/cog/internal/ast"
	"github.com/samborkent/cog/internal/tokens"
	"github.com/samborkent/cog/internal/types"
)

func (p *Parser) parseCallArguments(ctx context.Context, procedure *ast.Procedure) []ast.Expression {
	if p.this().Type != tokens.LParen {
		p.error(p.this(), "expected '(' after call identifier", "parseCallArguments")
		return nil
	}

	p.advance("parseCallArguments (") // consume '('

	if p.this().Type == tokens.RParen {
		return nil
	}

	args := []ast.Expression{}

	var procType *types.Procedure
	var ok bool

	if procedure != nil {
		procType, ok = procedure.Identifier.ValueType.(*types.Procedure)
		if !ok {
			panic("failed to assert procedure type")
		}
	}

	for i := 0; p.this().Type != tokens.RParen && p.this().Type != tokens.EOF; i++ {
		if ctx.Err() != nil {
			return nil
		}

		var arg ast.Expression

		if procedure == nil {
			arg = p.expression(ctx, types.None)
			if arg == nil {
				return nil
			}
		} else {
			arg = p.expression(ctx, procType.Parameters[i].Type)
			if arg == nil {
				return nil
			}
		}

		args = append(args, arg)

		if p.this().Type == tokens.Comma {
			p.advance("parseCallArguments ,") // consume ','
		}
	}

	if p.this().Type != tokens.RParen {
		p.error(p.this(), "expected ')' after function call arguments", "parseCallArguments")
		return nil
	}

	p.advance("parseCallArguments )") // consume ')'

	return args
}
