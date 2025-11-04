package parser

import (
	"context"
	"fmt"

	"github.com/samborkent/cog/internal/ast"
	"github.com/samborkent/cog/internal/types"
)

func (p *Parser) parseAssignment(ctx context.Context, ident *ast.Identifier) *ast.Assignment {
	symbol, ok := p.symbols.Resolve(ident.Name)
	if !ok {
		p.error(p.prev(), "unknown identifier", "parseAssignment")
		return nil
	}

	switch symbol.Identifier.Qualifier {
	case ast.QualifierImmutable:
		p.error(p.prev(), "cannot reassign a constant", "parseAssignment")

		// Skip until next line.
		p.advance("parseAssignment error =") // consume =
		_ = p.expression(ctx, symbol.Type())

		return nil
	case ast.QualifierType:
		if ident.Name == "_" {
			// Ignore no-op assignment
			break
		}

		p.error(p.prev(), "cannot assign to a type identifier", "parseAssignment")
		return nil
	}

	node := &ast.Assignment{
		Token:      p.this(),
		Identifier: symbol.Identifier,
	}

	p.advance("parseAssignment") // consume '='

	expr := p.expression(ctx, symbol.Type())
	if expr == nil {
		return nil
	}

	node.Expression = expr

	if symbol.Identifier.Name != "_" && !types.Equal(symbol.Type(), expr.Type()) {
		p.error(node.Token, fmt.Sprintf("type mismatch: cannot assign %q to variable of type %q", expr.Type(), symbol.Type()), "parseAssignment")
		return nil
	}

	if node.Identifier.ValueType == nil || node.Identifier.ValueType == types.None {
		node.Identifier.ValueType = expr.Type()
	}

	if symbol.Type() == types.None {
		p.symbols.Update(ident.Name, expr.Type())
	}

	return node
}
