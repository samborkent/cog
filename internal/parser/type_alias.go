package parser

import (
	"context"
	"fmt"

	"github.com/samborkent/cog/internal/ast"
	"github.com/samborkent/cog/internal/tokens"
	"github.com/samborkent/cog/internal/types"
)

func (p *Parser) parseTypeAlias(ctx context.Context, ident *ast.Identifier) *ast.Type {
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

	// Parse optional type parameters: <T ~ any, K ~ comparable>
	var typeParams []*types.TypeParam

	if p.this().Type == tokens.LT {
		typeParams = p.parseTypeParams(ctx)
		if typeParams == nil {
			return nil
		}

		typeDecl.TypeParameters = typeParams
	}

	p.advance("parseTypeAlias export ident ~") // consume ~

	// If there are type params, push them into an enclosed scope so that
	// type parameter names (e.g. T) are resolvable in the alias body.
	if len(typeParams) > 0 {
		outer := p.symbols
		p.symbols = NewEnclosedSymbolTable(outer)

		for _, tp := range typeParams {
			p.symbols.Define(&ast.Identifier{
				Name:      tp.Name,
				ValueType: tp,
				Qualifier: ast.QualifierType,
			})
		}

		defer func() { p.symbols = outer }()
	}

	typ := p.parseCombinedType(ctx, ident.Exported, ident.Global)
	if typ == nil {
		return nil
	}

	typeDecl.Identifier.ValueType = typ
	typeDecl.Alias = typ

	// Store type params on the alias for transpilation.
	if len(typeParams) > 0 {
		if alias, ok := typ.(*types.Alias); ok {
			alias.TypeParams = typeParams
		} else {
			// TODO: check if we need this
			// // Wrap in an alias so type params can be carried.
			// wrapped := &types.Alias{
			// 	Name:       ident.Name,
			// 	Derived:    typ,
			// 	Exported:   ident.Exported,
			// 	Global:     ident.Global,
			// 	TypeParams: typeParams,
			// }
			// typeDecl.Alias = wrapped
			// typeDecl.Identifier.ValueType = wrapped
		}
	}

	// Define type if in inner scope
	// TODO: find out why we had these restrictions.
	// if p.symbols.Outer != nil && len(typeParams) == 0 {
	p.symbols.Define(typeDecl.Identifier)
	// }

	return typeDecl
}
