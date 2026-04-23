package parser

import (
	"context"
	"fmt"

	"github.com/samborkent/cog/internal/ast"
	"github.com/samborkent/cog/internal/tokens"
	"github.com/samborkent/cog/internal/types"
)

func (p *Parser) parseTypeAlias(ctx context.Context, ident *ast.Identifier) ast.NodeValue {
	if p.symbols.Outer == nil {
		// Ensure type already exists (created during find globals sweep)
		_, ok := p.symbols.Resolve(ident.Name)
		if !ok {
			p.error(p.this(), fmt.Sprintf("missing global type symbol %q", ident.Name), "parseTypeAlias")
			return ast.ZeroNode
		}
	}

	ident.Qualifier = ast.QualifierType

	typeDecl := &ast.Type{
		Token:      p.this(),
		Identifier: ident,
	}

	// Parse optional type parameters: <T ~ any, K ~ comparable>
	var typeParams []*types.Alias

	if p.this().Type == tokens.LT {
		typeParams = p.parseTypeParams(ctx)
		if typeParams == nil {
			return ast.ZeroNode
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
		return ast.ZeroNode
	}

	// Carry over methods registered during the global scan:
	// findGlobalMethod attached methods to the original *Struct, but
	// parseCombinedType just created a new *Struct that will replace it.
	if newStruct, ok := typ.(*types.Struct); ok {
		if existing, ok := p.symbols.Resolve(ident.Name); ok {
			if oldStruct, ok := existing.Identifier.ValueType.(*types.Struct); ok {
				newStruct.Methods = oldStruct.Methods
			}
		}
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

	if iface, ok := typeDecl.Identifier.ValueType.(*types.Interface); ok {
		// Register interface methods as methods on the type for method call resolution.
		for _, method := range iface.Methods {
			p.symbols.DefineMethod(typeDecl.Identifier.Name, &ast.Identifier{
				Name:      method.Name,
				ValueType: method.Procedure,
				Qualifier: ast.QualifierMethod,
			})
		}
	}

	return ast.NewNode(ast.KindType, typeDecl)
}
