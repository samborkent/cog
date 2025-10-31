package parser

import (
	"context"
	"fmt"

	"github.com/samborkent/cog/internal/ast"
	"github.com/samborkent/cog/internal/tokens"
	"github.com/samborkent/cog/internal/types"
)

func (p *Parser) parseTypeAlias(ctx context.Context, ident *ast.Identifier, constant bool) *ast.Type {
	if p.symbols.Outer == nil {
		// Ensure type already exists (created during find globals sweep)
		_, ok := p.symbols.Resolve(ident.Name)
		if !ok {
			p.error(p.this(), fmt.Sprintf("missing global type symbol %q", ident.Name), "parseTypeAlias")
			return nil
		}
	}

	typeDecl := &ast.Type{
		Token:      p.this(),
		Identifier: ident,
	}

	p.advance("parseTypeAlias export ident ~") // consume ~

	var typ types.Type

	if p.this().Type == tokens.Enum {
		p.advance("parseTypeAlias enum") // consume enum

		if p.this().Type != tokens.LBracket {
			p.error(p.this(), "expected [ in enum declaration", "parseTypeAlias")
			return nil
		}

		p.advance("parseTypeAlias enum [") // consume [

		enumValType := p.parseCombinedType(ctx, ident.Exported, constant)

		if p.this().Type != tokens.RBracket {
			p.error(p.this(), "expected ] in enum declaration", "parseTypeAlias")
			return nil
		}

		p.advance("parseTypeAlias enum ]") // consume ]

		if p.this().Type != tokens.LBrace {
			p.error(p.this(), "expected { after enum declaration", "parseTypeAlias")
			return nil
		}

		enumLiteral := &ast.EnumLiteral{
			Token:     p.this(),
			ValueType: enumValType,
			Values:    make([]*ast.EnumValue, 0),
		}

		p.advance("parseTypeAlias {") // consume {

		for p.this().Type != tokens.RBrace {
			if ctx.Err() != nil {
				return nil
			}

			if p.this().Type != tokens.Identifier {
				p.error(p.this(), "expected identifier in enum literal", "parseTypeAlias")
				return nil
			}

			enumIdent := &ast.Identifier{
				Token:     p.this(),
				Name:      p.this().Literal,
				ValueType: enumValType,
				Exported:  ident.Exported,
			}

			p.symbols.DefineEnumValue(ident.Name, enumIdent)

			p.advance("parseTypeAlias enum literal identifier") // consume identifier

			if p.this().Type != tokens.Declaration {
				p.error(p.this(), "expected := in enum literal", "parseTypeAlias")
				return nil
			}

			p.advance("parseTypeAlias enum literal :=") // consume :=

			enumExpr := p.expression(ctx, enumValType)
			if enumExpr != nil {
				enumLiteral.Values = append(enumLiteral.Values, &ast.EnumValue{
					Identifier: enumIdent,
					Value:      enumExpr,
				})
			}

			if p.this().Type == tokens.Comma {
				p.advance("parseTypeAlias enum literal ,") // consume ,
			}
		}

		p.advance("parseTypeAlias }") // consume }

		typeDecl.Literal = enumLiteral

		typ = &types.Enum{
			Value: enumValType,
		}
	} else {
		typ = p.parseCombinedType(ctx, ident.Exported, constant)
		if typ == nil {
			return nil
		}
	}

	typeDecl.Identifier.ValueType = typ
	typeDecl.Alias = typ

	// Define type if in inner scope
	if p.symbols.Outer != nil {
		p.symbols.Define(typeDecl.Identifier, SymbolKindType)
	}

	return typeDecl
}
