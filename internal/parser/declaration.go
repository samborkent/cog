package parser

import (
	"context"

	"github.com/samborkent/cog/internal/ast"
	"github.com/samborkent/cog/internal/tokens"
	"github.com/samborkent/cog/internal/types"
)

func (p *Parser) parseTypedDeclaration(ctx context.Context, ident *ast.Identifier, constant bool) *ast.Declaration {
	typeToken := p.this()

	identType, ok := types.Lookup[typeToken.Type]
	if !ok {
		symbol, ok := p.symbols.Resolve(p.this().Literal)
		if !ok {
			p.error(p.this(), "expected type", "parseTypedDeclaration")
			return nil
		}

		identType = &types.Alias{
			Name:     p.this().Literal,
			Derived:  symbol.Type(),
			Exported: symbol.Identifier.Exported,
		}
	}

	p.advance("parseTypedDeclaration type") // consume type

	if p.this().Type == tokens.Question {
		p.advance("parseTypedDeclaration type ?") // consume ?

		if identType.Kind() == types.OptionKind {
			p.error(p.this(), "nested optional types are not allowed", "parseType")
			return nil
		}

		identType = &types.Option{Value: identType}
	}

	// Check if type is an alias.
	_, ok = identType.(*types.Alias)
	if ok {
		ident.ValueType = identType
	} else {
		switch identType.Kind() {
		case types.EnumKind:
			if !constant {
				p.error(ident.Token, "enum declarations must be constant", "parseTypedDeclaration")
				return nil
			}

			if p.this().Type != tokens.LBracket {
				p.error(p.this(), "expected [ after enum type", "parseTypedDeclaration")
				return nil
			}

			p.advance("parseTypedDeclaration enum [") // consume [

			valType, ok := types.Lookup[p.this().Type]
			if !ok {
				symbol, ok := p.symbols.Resolve(p.this().Literal)
				if !ok {
					p.error(p.this(), "expected enum value type", "parseTypedDeclaration")
					return nil
				}

				valType = &types.Alias{
					Name:     p.this().Literal,
					Derived:  symbol.Type(),
					Exported: symbol.Identifier.Exported,
				}
			}

			p.advance("parseTypedDeclaration enum value type") // consume elem type

			if p.this().Type != tokens.RBracket {
				p.error(p.this(), "expected ] after enum value type", "parseTypedDeclaration")
				return nil
			}

			p.advance("parseTypedDeclaration enum ]") // consume ]

			ident.ValueType = &types.Enum{Value: valType}
		case types.SetKind:
			if p.this().Type != tokens.LBracket {
				p.error(p.this(), "expected [ after set type", "parseTypedDeclaration")
				return nil
			}

			p.advance("parseTypedDeclaration set [") // consume [

			elemType, ok := types.Lookup[p.this().Type]
			if !ok {
				symbol, ok := p.symbols.Resolve(p.this().Literal)
				if !ok {
					p.error(p.this(), "expected set element type", "parseTypedDeclaration")
					return nil
				}

				elemType = &types.Alias{
					Name:     p.this().Literal,
					Derived:  symbol.Type(),
					Exported: symbol.Identifier.Exported,
				}
			}

			p.advance("parseTypedDeclaration set element type") // consume elem type

			if p.this().Type != tokens.RBracket {
				p.error(p.this(), "expected ] after set element type", "parseTypedDeclaration")
				return nil
			}

			p.advance("parseTypedDeclaration set ]") // consume ]

			ident.ValueType = &types.Set{Element: elemType}
		default:
			ident.ValueType = identType
		}
	}

	node := p.parseDeclaration(ctx, ident, constant)
	if node == nil {
		return nil
	}

	return node
}

func (p *Parser) parseDeclaration(ctx context.Context, ident *ast.Identifier, constant bool) *ast.Declaration {
	symbol, ok := p.symbols.Resolve(ident.Name)
	if ok && symbol.Scope != ScanScope {
		p.error(ident.Token, "cannot redeclare variable", "parseDeclaration")
		return nil
	}

	if ident.ValueType == nil {
		ident.ValueType = types.None
	}

	node := &ast.Declaration{
		Assignment: &ast.Assignment{
			Token:      p.this(),
			Identifier: ident,
		},
		Constant: constant,
		Type:     ident.ValueType,
	}

	if p.this().Type != tokens.Assign {
		// Empty declaration.
		if constant {
			p.error(p.this(), "constant declarations must be initialized", "parseDeclaration")
			return nil
		}

		return node
	}

	p.advance("parseDeclaration") // consume ':=' or '='

	startToken := p.this()

	expr := p.expression(ctx, ident.ValueType)
	if expr == nil {
		p.error(startToken, "unable to parse expression", "parseDeclaration")
		return nil
	}

	node.Assignment.Expression = expr

	if ident.ValueType == types.None {
		exprType := expr.Type()

		ident.ValueType = exprType
		node.Assignment.Identifier.ValueType = exprType
		node.Type = exprType
	}

	kind := SymbolKindVariable
	if constant {
		kind = SymbolKindConstant
	}

	p.symbols.Define(ident, kind)

	if node.Assignment.Expression.Type().Underlying().Kind() == types.EnumKind {
		if enumLiteral, ok := node.Assignment.Expression.(*ast.EnumLiteral); ok {
			for _, val := range enumLiteral.Values {
				p.symbols.DefineEnumValue(ident.Name, val.Identifier)
			}
		}
	}

	return node
}
