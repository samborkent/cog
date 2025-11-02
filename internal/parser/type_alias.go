package parser

import (
	"context"
	"fmt"

	"github.com/samborkent/cog/internal/ast"
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

	typ := p.parseCombinedType(ctx, ident.Exported)
	if typ == nil {
		return nil
	}

	typeDecl.Identifier.ValueType = typ
	typeDecl.Alias = typ

	// Define type if in inner scope
	if p.symbols.Outer != nil {
		p.symbols.Define(typeDecl.Identifier, SymbolKindType)
	}

	return typeDecl
}
