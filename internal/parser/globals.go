package parser

import (
	"context"

	"github.com/samborkent/cog/internal/ast"
	"github.com/samborkent/cog/internal/tokens"
	"github.com/samborkent/cog/internal/types"
)

func (p *Parser) findGlobals(ctx context.Context) {
tokenLoop:
	for p.this().Type != tokens.EOF {
		exported := false

		if p.this().Type == tokens.Export {
			p.advance("findGlobals export") // consume export
			exported = true
		}

		qualifier := ast.QualifierImmutable

		switch p.this().Type {
		case tokens.Dynamic:
			qualifier = ast.QualifierDynamic
			p.advance("findGlobals dyn") // consume dyn
		case tokens.Variable:
			qualifier = ast.QualifierVariable
			p.advance("findGlobals var") // consume var
		}

		switch p.this().Type {
		case tokens.GoImport:
			p.advance("findGlobals goimport") // consume goimport

			if p.this().Type == tokens.LParen {
				p.skipGrouped(ctx)
			}
		case tokens.Identifier:
			switch p.next().Type {
			case tokens.Colon, tokens.Declaration:
				p.findGlobalDecl(ctx, exported, qualifier)
			case tokens.Tilde:
				p.findGlobalType(ctx, exported)
			default:
				p.advance("findGlobals") // consume token
			}
		case tokens.Package:
			p.advance("findGlobals package") // consume package

			if p.this().Type == tokens.Identifier {
				p.advance("findGlobals package identifier") // consume package identifier
			}
		case tokens.EOF:
			break tokenLoop
		default:
			p.advance("findGlobals") // consume token
		}
	}

	p.i = 0
	p.Errs = p.Errs[:0]
}

func (p *Parser) findGlobalDecl(ctx context.Context, exported bool, qualifier ast.Qualifier) {
	if p.this().Type != tokens.Identifier {
		return
	}

	_, ok := p.symbols.Resolve(p.this().Literal)
	if ok {
		// Report redeclare error and advance past the identifier to avoid an infinite loop
		p.error(p.this(), "cannot redeclare variable", "findGlobalDecl")
		p.advance("findGlobalDecl redeclare") // consume identifier to make progress
		return
	}

	ident := &ast.Identifier{
		Token:     p.this(),
		Name:      p.this().Literal,
		Exported:  exported,
		Qualifier: qualifier,
	}

	p.advance("findGlobalDecl identifier") // consume identifier

	switch p.this().Type {
	case tokens.Colon:
		p.advance("findGlobalDecl :") // consume :

		ident.ValueType = p.parseCombinedType(ctx, exported)

		p.symbols.DefineGlobal(ident)

		if p.this().Type == tokens.Assign {
			p.advance("findGlobalDecl =") // consume =

			if p.this().Type == tokens.LBrace {
				p.skipScope(ctx)
			} else {
				_ = p.expression(ctx, ident.ValueType)
			}
		}
	case tokens.Declaration:
		p.advance("findGlobalDecl :=") // consume :=
		p.symbols.DefineGlobal(ident)

		if p.this().Type == tokens.LBrace {
			p.skipScope(ctx)
		} else {
			_ = p.expression(ctx, types.None)
		}
	default:
		return
	}
}

func (p *Parser) findGlobalType(ctx context.Context, exported bool) {
	ident := &ast.Identifier{
		Token:    p.this(),
		Name:     p.this().Literal,
		Exported: exported,
	}

	p.advance("findGlobalType identifier") // consume identifier

	_, ok := p.symbols.Resolve(ident.Name)
	if ok {
		p.error(p.this(), "cannot redeclare type", "findGlobalType")
		return
	}

	p.advance("findGlobalType ~") // consume ~

	if p.this().Type == tokens.Enum {
		p.advance("findGlobalType enum") // consume enum

		if p.this().Type != tokens.LBracket {
			p.error(p.this(), "expected [ in enum declaration", "findGlobalType")
			return
		}

		p.advance("findGlobalType enum [") // consume [

		enumValType := p.parseCombinedType(ctx, exported)

		enumType := &types.Enum{ValueType: enumValType}

		if p.this().Type != tokens.RBracket {
			p.error(p.this(), "expected ] in enum declaration", "findGlobalType")
			return
		}

		p.advance("findGlobalType enum ]") // consume ]

		if p.this().Type != tokens.LBrace {
			p.error(p.this(), "expected { in enum literal", "findGlobalType")
			return
		}

		p.advance("findGlobalType enum literal {") // consume {

		for p.this().Type != tokens.RBrace {
			if ctx.Err() != nil {
				return
			}

			if p.this().Type != tokens.Identifier {
				p.error(p.this(), "expected identifier in enum literal", "findGlobalType")
				return
			}

			valIdent := &ast.Identifier{
				Token:     p.this(),
				Name:      p.this().Literal,
				ValueType: enumValType,
				Exported:  exported,
			}

			p.symbols.DefineEnumValue(ident.Name, valIdent)

			p.advance("findGlobalType enum literal identifier") // consume identifier

			if p.this().Type != tokens.Declaration {
				p.error(p.this(), "expected := in enum literal", "findGlobalType")
				return
			}

			p.advance("findGlobalType enum literal :=") // consume :=

			enumVal := p.expression(ctx, enumValType)
			if enumVal != nil {
				enumType.Values = append(enumType.Values, &types.EnumValue{
					Name:  valIdent.Name,
					Value: enumVal,
				})
			}

			if p.this().Type == tokens.Comma {
				p.advance("findGlobalType enum literal ,") // consume ,
			}
		}

		ident.ValueType = enumType
		p.symbols.DefineGlobal(ident)

		return
	}

	alias := p.parseCombinedType(ctx, ident.Exported)
	if alias == nil {
		return
	}

	ident.ValueType = alias

	p.symbols.DefineGlobal(ident)
}

func (p *Parser) skipScope(ctx context.Context) {
	braceIndex := 0

	for {
		if ctx.Err() != nil {
			return
		}

		switch p.this().Type {
		case tokens.LBrace:
			braceIndex++
		case tokens.RBrace:
			braceIndex--
		}

		p.advance("skipScope " + p.this().Literal)

		if braceIndex == 0 {
			return
		}
	}
}

func (p *Parser) skipGrouped(ctx context.Context) {
	parenIndex := 0

	for {
		if ctx.Err() != nil {
			return
		}

		switch p.this().Type {
		case tokens.LParen:
			parenIndex++
		case tokens.RParen:
			parenIndex--
		}

		p.advance("skipGrouped " + p.this().Literal)

		if parenIndex == 0 {
			return
		}
	}
}
