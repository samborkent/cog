package parser

import (
	"context"
	"fmt"

	"github.com/samborkent/cog/internal/tokens"
	"github.com/samborkent/cog/internal/types"
)

// resolveConstraintToken looks up the current token as a constraint.
// Keywords are matched by type name; identifiers by literal (for "int", "uint", etc.).
func (p *Parser) resolveConstraintToken() types.Type {
	if c, ok := types.LookupConstraint(p.lex.This().Type.String()); ok {
		return c
	}

	if p.lex.This().Type == tokens.Identifier {
		if c, ok := types.LookupConstraint(p.lex.This().Literal); ok {
			return c
		}
	}

	return nil
}

func (p *Parser) parseTypeArguments(ctx context.Context) []types.Type {
	if p.lex.This().Type != tokens.LT {
		p.error(p.lex.This(), "expected opening < for type arguments", "parseTypeArguments")
		return nil
	}

	p.lex.Step() // consume <

	typ := p.parseCombinedType(ctx, false, false)
	if typ == nil {
		return nil
	}

	args := []types.Type{typ}

	for p.lex.This().Type == tokens.Comma {
		p.lex.Step() // consume ,

		typ := p.parseCombinedType(ctx, false, false)
		if typ == nil {
			return nil
		}

		args = append(args, typ)
	}

	if p.lex.This().Type != tokens.GT {
		p.error(p.lex.This(), "expected closing > for type arguments", "parseTypeArguments")
		return nil
	}

	p.lex.Step() // consume >

	return args
}

// parseTypeParams parses a generic type parameter list in declaration position:
//
//	<T ~ any>
//	<T ~ any, K ~ comparable>
//	<T ~ string | int>
//
// Each parameter is an identifier, followed by ~, followed by one or more
// constraint keywords separated by |.
func (p *Parser) parseTypeParams(ctx context.Context) []*types.Alias {
	if p.lex.This().Type != tokens.LT {
		p.error(p.lex.This(), "expected opening < for type parameters", "parseTypeParams")
		return nil
	}

	p.lex.Step() // consume <

	var params []*types.Alias

	for !p.match(p.lex.This(), tokens.GT) {
		if ctx.Err() != nil {
			return nil
		}

		if p.lex.This().Type != tokens.Identifier {
			p.error(p.lex.This(), "expected type parameter name", "parseTypeParams")
			return nil
		}

		name := p.lex.This().Literal
		p.lex.Step() // consume parameter name

		if p.lex.This().Type != tokens.Tilde {
			p.error(p.lex.This(), "expected ~ after type parameter name", "parseTypeParams")
			return nil
		}

		p.lex.Step() // consume ~

		// Parse constraint(s): keyword constraints or concrete types separated by |.
		// Try keyword constraint first, fall back to concrete type via parseType.
		constraint := p.resolveConstraintToken()
		if constraint != nil {
			p.lex.Step() // consume keyword constraint
		} else {
			constraint = p.parseType(ctx)
			if constraint == nil {
				p.error(p.lex.This(), fmt.Sprintf("expected constraint or type, got %q", p.lex.This().Type.String()), "parseTypeParams")
				return nil
			}
		}

		constraints := []types.Type{constraint}

		for p.lex.This().Type == tokens.Pipe {
			p.lex.Step() // consume |

			constraint := p.resolveConstraintToken()
			if constraint != nil {
				p.lex.Step() // consume keyword constraint
			} else {
				constraint = p.parseType(ctx)
				if constraint == nil {
					p.error(p.lex.This(), fmt.Sprintf("expected constraint or type after |, got %q", p.lex.This().Type.String()), "parseTypeParams")
					return nil
				}
			}

			constraints = append(constraints, constraint)
		}

		var constraintType types.Type
		if len(constraints) == 1 {
			constraintType = constraints[0]
		} else {
			constraintType = &types.Union{Variants: constraints}
		}

		params = append(params, &types.Alias{
			Name:       name,
			Constraint: constraintType,
		})

		if p.lex.This().Type == tokens.Comma {
			p.lex.Step() // consume ,
			continue
		}

		break
	}

	if p.lex.This().Type != tokens.GT {
		p.error(p.lex.This(), "expected closing > for type parameters", "parseTypeParams")
		return nil
	}

	p.lex.Step() // consume >

	return params
}
