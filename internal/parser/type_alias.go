package parser

import (
	"context"
	"fmt"

	"github.com/samborkent/cog/internal/ast"
	"github.com/samborkent/cog/internal/tokens"
	"github.com/samborkent/cog/internal/types"
)

func (p *Parser) parseTypeAlias(ctx context.Context, ident *ast.Identifier) ast.NodeIndex {
	if !p.globalsPass && p.symbols.Outer == nil {
		// Ensure type already exists (created during globals pass)
		_, ok := p.symbols.Resolve(ident.Token.Literal)
		if !ok {
			p.error(p.lex.This(), fmt.Sprintf("missing global type symbol %q", ident.Token.Literal), "parseTypeAlias")
			return ast.ZeroNodeIndex
		}
	}

	ident.Qualifier = ast.QualifierType

	typeToken := p.lex.This()

	// Parse optional type parameters: <T ~ any, K ~ comparable>
	var typeParams []*types.Alias

	if p.lex.This().Type == tokens.LT {
		tp := p.parseTypeParams(ctx)
		if tp == nil {
			return ast.ZeroNodeIndex
		}

		typeParams = tp
	}

	p.lex.Step() // consume ~

	// Save the global scope before pushing type param scope, so that
	// DefineGlobal registers the type on the correct (outer) scope.
	globalScope := p.symbols

	// If there are type params, push them into an enclosed scope so that
	// type parameter names (e.g. T) are resolvable in the alias body.
	if len(typeParams) > 0 {
		p.symbols = NewEnclosedSymbolTable(globalScope)

		for _, tp := range typeParams {
			p.symbols.Define(&ast.Identifier{
				Token: tokens.Token{
					Type:    tokens.Identifier,
					Literal: tp.Name,
				},
				ValueType: tp,
				Qualifier: ast.QualifierType,
			})
		}

		defer func() { p.symbols = globalScope }()
	}

	typ := p.parseCombinedType(ctx, ident.Exported, ident.Global)
	if typ == nil {
		return ast.ZeroNodeIndex
	}

	// Carry over methods registered during the global scan:
	// findGlobalMethod attached methods to the original *Struct, but
	// parseCombinedType just created a new *Struct that will replace it.
	if newStruct, ok := typ.(*types.Struct); ok {
		if existing, ok := globalScope.Resolve(ident.Token.Literal); ok {
			if oldStruct, ok := existing.Identifier.ValueType.(*types.Struct); ok {
				newStruct.Methods = oldStruct.Methods
			}
		}
	}

	ident.ValueType = typ

	// Store type params on the alias for transpilation.
	if len(typeParams) > 0 {
		if alias, ok := typ.(*types.Alias); ok {
			alias.TypeParams = typeParams
		} else {
			// Wrap in an alias so type params can be carried.
			wrapped := &types.Alias{
				Name:       ident.Token.Literal,
				Derived:    typ,
				Exported:   ident.Exported,
				Global:     ident.Global,
				TypeParams: typeParams,
			}
			ident.ValueType = wrapped
		}
	}

	// Define type if in inner scope
	// TODO: find out why we had these restrictions.
	// if p.symbols.Outer != nil && len(typeParams) == 0 {
	if p.globalsPass {
		globalScope.DefineGlobal(ident)
	} else {
		p.symbols.Define(ident)
	}
	// }

	if iface, ok := ident.ValueType.(*types.Interface); ok {
		// Register interface methods as methods on the type for method call resolution.
		for _, method := range iface.Methods {
			p.symbols.DefineMethod(ident.Token.Literal, &ast.Identifier{
				Token: tokens.Token{
					Type:    tokens.Identifier,
					Literal: method.Name,
				},
				ValueType: method.Procedure,
				Qualifier: ast.QualifierMethod,
			})
		}
	}

	return p.ast.NewType(typeToken, ident, typeParams, typ)
}
