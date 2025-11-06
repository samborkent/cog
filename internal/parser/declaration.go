package parser

import (
	"context"

	"github.com/samborkent/cog/internal/ast"
	"github.com/samborkent/cog/internal/tokens"
	"github.com/samborkent/cog/internal/types"
)

func (p *Parser) parseTypedDeclaration(ctx context.Context, ident *ast.Identifier) *ast.Declaration {
	identType := p.parseCombinedType(ctx, ident.Exported)
	if identType == nil {
		return nil
	}

	ident.ValueType = identType

	node := p.parseDeclaration(ctx, ident)
	if node == nil {
		return nil
	}

	return node
}

func (p *Parser) parseDeclaration(ctx context.Context, ident *ast.Identifier) *ast.Declaration {
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
	}

	if !p.match(tokens.Assign, tokens.Declaration) {
		if ident.Qualifier == ast.QualifierImmutable {
			p.error(p.this(), "immutable declarations must be initialized", "parseDeclaration")
			return nil
		}

		// Uninitialized variable
		p.symbols.Define(ident)

		return node
	}

	p.advance("parseDeclaration") // consume := or =

	expr := p.expression(ctx, ident.ValueType)
	if expr == nil {
		return nil
	}

	node.Assignment.Expression = expr

	if ident.ValueType == types.None {
		exprType := expr.Type()

		ident.ValueType = exprType
		node.Assignment.Identifier.ValueType = exprType
	}

	p.symbols.Define(ident)

	return node
}
