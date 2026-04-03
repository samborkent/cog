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
	if c, ok := types.LookupConstraint(p.this().Type.String()); ok {
		return c
	}
	if p.this().Type == tokens.Identifier {
		if c, ok := types.LookupConstraint(p.this().Literal); ok {
			return c
		}
	}
	return nil
}

func (p *Parser) parseTypeArguments(ctx context.Context) []types.Type {
	if p.this().Type != tokens.LT {
		p.error(p.this(), "expected opening < for type arguments", "parseTypeArguments")
		return nil
	}

	p.advance("parseTypeArguments <") // consume <

	typ := p.parseCombinedType(ctx, false, false)
	if typ == nil {
		return nil
	}

	args := []types.Type{typ}

	for p.this().Type == tokens.Comma {
		p.advance("parseTypeArguments ,") // consume ,

		typ := p.parseCombinedType(ctx, false, false)
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

// parseTypeParams parses a generic type parameter list in declaration position:
//
//	<T ~ any>
//	<T ~ any, K ~ comparable>
//	<T ~ string | int>
//
// Each parameter is an identifier, followed by ~, followed by one or more
// constraint keywords separated by |.
func (p *Parser) parseTypeParams(ctx context.Context) []*types.TypeParam {
	if p.this().Type != tokens.LT {
		p.error(p.this(), "expected opening < for type parameters", "parseTypeParams")
		return nil
	}

	p.advance("parseTypeParams <") // consume <

	var params []*types.TypeParam

	for !p.match(tokens.GT, tokens.EOF) {
		if ctx.Err() != nil {
			return nil
		}

		if p.this().Type != tokens.Identifier {
			p.error(p.this(), "expected type parameter name", "parseTypeParams")
			return nil
		}

		name := p.this().Literal
		p.advance("parseTypeParams name") // consume parameter name

		if p.this().Type != tokens.Tilde {
			p.error(p.this(), "expected ~ after type parameter name", "parseTypeParams")
			return nil
		}

		p.advance("parseTypeParams ~") // consume ~

		// Parse constraint(s): keyword constraints or concrete types separated by |.
		// Try keyword constraint first, fall back to concrete type via parseType.
		constraint := p.resolveConstraintToken()
		if constraint != nil {
			p.advance("parseTypeParams constraint") // consume keyword constraint
		} else {
			constraint = p.parseType(ctx)
			if constraint == nil {
				p.error(p.this(), fmt.Sprintf("expected constraint or type, got %q", p.this().Type.String()), "parseTypeParams")
				return nil
			}
		}

		constraints := []types.Type{constraint}

		for p.this().Type == tokens.Pipe {
			p.advance("parseTypeParams |") // consume |

			constraint := p.resolveConstraintToken()
			if constraint != nil {
				p.advance("parseTypeParams constraint") // consume keyword constraint
			} else {
				constraint = p.parseType(ctx)
				if constraint == nil {
					p.error(p.this(), fmt.Sprintf("expected constraint or type after |, got %q", p.this().Type.String()), "parseTypeParams")
					return nil
				}
			}

			constraints = append(constraints, constraint)
		}

		params = append(params, &types.TypeParam{
			Name:        name,
			Constraints: constraints,
		})

		if p.this().Type == tokens.Comma {
			p.advance("parseTypeParams ,") // consume ,
			continue
		}

		break
	}

	if p.this().Type != tokens.GT {
		p.error(p.this(), "expected closing > for type parameters", "parseTypeParams")
		return nil
	}

	p.advance("parseTypeParams >") // consume >

	return params
}
