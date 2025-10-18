package parser

import (
	"context"

	"github.com/samborkent/cog/internal/tokens"
	"github.com/samborkent/cog/internal/types"
)

func (p *Parser) parseCombinedType(ctx context.Context, exported bool) types.Type {
	typ := p.parseType(ctx)

	switch p.this().Type {
	case tokens.Comma:
		// Tuple
		tuple := &types.Tuple{
			Types:    make([]types.Type, 1, types.TupleMaxTypes),
			Exported: exported,
		}

		// Put parsed type as first type.
		tuple.Types[0] = typ

		for p.this().Type == tokens.Comma {
			p.advance("parseCombinedType tuple ,") // consume ,

			next := p.parseType(ctx)
			if next != nil {
				tuple.Types = append(tuple.Types, next)
			}
		}

		return tuple
	case tokens.Pipe:
		// Union
		union := &types.Union{
			Either:   typ,
			Exported: exported,
		}

		if p.this().Type != tokens.Pipe {
			p.error(p.this(), "expected | in union type declaration", "parseCombinedType")
			return nil
		}

		p.advance("parseCombinedType union |") // consume |

		next := p.parseType(ctx)
		if next != nil {
			union.Or = next
		}

		if p.this().Type == tokens.Pipe {
			p.error(p.this(), "union can only contain two types", "parseCombinedType")
			return nil
		}

		return union
	}

	return typ
}

func (p *Parser) parseType(ctx context.Context) types.Type {
	// TODO: also parse set type
	switch p.this().Type {
	case tokens.Struct:
		return p.parseStruct(ctx)
	}

	typ, ok := types.Lookup[p.this().Type]
	if !ok {
		// Non-basic type, try to find in symbol table.
		typeSymbol, ok := p.symbols.Resolve(p.this().Literal)
		if !ok || typeSymbol.Kind != SymbolKindType {
			p.error(p.this(), "unknown type found in type declaration", "parseStatement")
			return nil
		}

		p.advance("parseType alias") // consume type

		return &types.Alias{
			Name:     typeSymbol.Identifier.Name,
			Derived:  typeSymbol.Identifier.ValueType,
			Exported: typeSymbol.Identifier.Exported,
		}
	}

	p.advance("parseType type") // consume type

	return typ
}

func (p *Parser) parseStruct(ctx context.Context) types.Type {
	p.advance("parseStruct struct") // consume struct

	if p.this().Type != tokens.LBrace {
		p.error(p.this(), "expected { after struct declaration", "parseStruct")
		return nil
	}

	p.advance("parseStruct {") // consume {

	fields := []*types.Field{}

	for p.this().Type != tokens.RBrace {
		if ctx.Err() != nil {
			return nil
		}

		switch p.this().Type {
		case tokens.Export:
			p.advance("parseStruct export") // consume export

			if p.this().Type == tokens.LParen {
				p.advance("parseStruct export (") // consume (

				for p.this().Type != tokens.RParen {
					field := p.parseField(ctx, true)
					if field == nil {
						return nil
					}

					fields = append(fields, field)
				}

				p.advance("parseStruct export )") // consume )
				continue
			}

			field := p.parseField(ctx, true)
			if field == nil {
				return nil
			}

			fields = append(fields, field)
		case tokens.Identifier:
			field := p.parseField(ctx, false)
			if field == nil {
				return nil
			}

			fields = append(fields, field)
		default:
			p.error(p.this(), "unexpected token found in struct declaration", "parseStruct")
			return nil
		}
	}

	p.advance("parseStruct }") // consume }

	return &types.Struct{
		Fields: fields,
	}
}

func (p *Parser) parseField(ctx context.Context, exported bool) *types.Field {
	field := &types.Field{
		Name:     p.this().Literal,
		Exported: exported,
	}

	p.advance("parseField identifier") // consume identifier

	if p.this().Type != tokens.Colon {
		p.error(p.this(), "expected : after field name in struct declaration", "parseStruct")
		return nil
	}

	p.advance("parseField :") // consume :

	field.Type = p.parseCombinedType(ctx, exported)

	return field
}
