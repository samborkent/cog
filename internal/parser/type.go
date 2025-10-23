package parser

import (
	"context"
	"fmt"

	"github.com/samborkent/cog/internal/ast"
	"github.com/samborkent/cog/internal/tokens"
	"github.com/samborkent/cog/internal/types"
)

func (p *Parser) parseCombinedType(ctx context.Context, exported bool) types.Type {
	if p.match(tokens.Function, tokens.Procedure) {
		return p.parseProcedureType(ctx)
	}

	typ := p.parseType(ctx)

	switch p.this().Type {
	case tokens.BitAnd:
		// Tuple
		tuple := &types.Tuple{
			Types:    make([]types.Type, 1, types.TupleMaxTypes),
			Exported: exported,
		}

		// Put parsed type as first type.
		tuple.Types[0] = typ

		for p.this().Type == tokens.BitAnd {
			p.advance("parseCombinedType tuple &") // consume &

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

		typ = &types.Alias{
			Name:     typeSymbol.Identifier.Name,
			Derived:  typeSymbol.Identifier.ValueType,
			Exported: typeSymbol.Identifier.Exported,
		}
	}

	p.advance("parseType type") // consume type

	if p.this().Type == tokens.Question {
		// Optional type
		p.advance("parseType ?") // consume ?

		if typ.Kind() == types.OptionKind {
			p.error(p.this(), "nested optional types are not allowed", "parseType")
			return nil
		}

		return &types.Option{
			Value: typ,
		}
	}

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

func (p *Parser) parseProcedureType(ctx context.Context) *types.Procedure {
	procType := &types.Procedure{
		Function:   p.this().Type == tokens.Function,
		Parameters: make([]*types.Parameter, 0),
	}

	p.advance("parseProcedureType proc/func")

	if p.this().Type != tokens.LParen {
		p.error(p.this(), fmt.Sprintf("expected '(' after %q in type", p.prev().Type))
		return nil
	}

	// Flag to keep track of if any of the parameters is optional.
	// When a parameter is marked as optional, all following parameters must also be optional.
	haveOptional := false

	for i := 0; !p.match(tokens.RParen, tokens.EOF); i++ {
		if ctx.Err() != nil {
			return nil
		}

		if p.this().Type != tokens.Identifier {
			p.error(p.this(), "expected parameter identifier", "parseParameters")
			return nil
		}

		var optional bool

		ident := &ast.Identifier{
			Token: p.this(),
			Name:  p.this().Literal,
		}

		param := &types.Parameter{
			Name: p.this().Literal,
		}

		if param.Name == "ctx" && (procType.Function || i > 0) {
			p.error(p.this(), "'ctx' identifier is reserved for the first input parameter of procedures", "parseParameters")
			return nil
		}

		p.advance("parseParameters loop identifier") // consume identifier

		if p.this().Type == tokens.Question {
			param.Optional = true
			haveOptional = true

			p.advance("parseParameters loop ?") // consume ?
		} else if haveOptional {
			// This parameter is not optional, but a previous parameter was, this is not allowed.
			p.error(p.prev(), "all input parameters following an optional parameter must also be optional", "parseParameters")
			return nil
		}

		if p.this().Type != tokens.Colon {
			p.error(p.this(), "expected ':' after input parameter identifier", "parseParameters")
			return nil
		}

		p.advance("parseParameters loop :") // consume :

		paramType := p.parseCombinedType(ctx, false)
		if paramType == nil {
			p.error(p.this(), "unknown parameter type", "parseParameters")
			return nil
		}

		if param.Name == "ctx" && paramType.Kind() != types.Context {
			p.error(p.this(), "input parameter 'ctx' must be of type 'context'", "parseParameters")
			return nil
		} else if paramType.Kind() == types.Context && (procType.Function || param.Name != "ctx") {
			p.error(p.this(), "context type may only be used as first input parameter of procedures", "parseParameters")
			return nil
		}

		param.Type = paramType
		ident.ValueType = paramType

		// TODO: ensure we want to make function parameters always constant (read-only)
		p.symbols.Define(ident, SymbolKindConstant)

		if p.this().Type == tokens.Assign {
			if !optional {
				p.error(p.this(), "default values are only allowed for optional input parameters", "parseParameters")
				return nil
			}

			// Default parameter value assignment
			p.advance("parseParameters loop =") // consume '='

			expr := p.expression(ctx, paramType)
			if expr != nil {
				param.Default = expr
			}
		}

		procType.Parameters = append(procType.Parameters, param)

		if p.this().Type == tokens.Comma {
			p.advance("parseParameters loop ,") // consume ','
		}
	}

	return procType
}
