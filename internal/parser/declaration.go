package parser

import (
	"context"

	"github.com/samborkent/cog/internal/ast"
	"github.com/samborkent/cog/internal/tokens"
	"github.com/samborkent/cog/internal/types"
)

func (p *Parser) parseTypedDeclaration(ctx context.Context, ident *ast.Identifier, constant bool) *ast.Declaration {
	identType := p.parseCombinedType(ctx, ident.Exported, constant)
	if identType == nil {
		return nil
	}

	ident.ValueType = identType

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
	}

	kind := SymbolKindVariable
	if constant {
		kind = SymbolKindConstant
	}

	if p.this().Type != tokens.Assign && p.this().Type != tokens.Declaration {
		if constant {
			p.error(p.this(), "constant declarations must be initialized", "parseDeclaration")
			return nil
		}

		// Uninitialized variable
		p.symbols.Define(ident, kind)

		return node
	}

	p.advance("parseDeclaration") // consume := or =

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
