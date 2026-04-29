package parser

import (
	"context"
	"fmt"

	"github.com/samborkent/cog/internal/ast"
	"github.com/samborkent/cog/internal/types"
)

func (p *Parser) parseAssignment(ctx context.Context, ident *ast.Identifier) ast.NodeIndex {
	symbol, ok := p.symbols.Resolve(ident.Name)
	if !ok {
		p.error(p.prev(), "unknown identifier", "parseAssignment")
		return ast.ZeroNodeIndex
	}

	switch symbol.Identifier.Qualifier {
	case ast.QualifierImmutable:
		p.error(p.prev(), "cannot reassign a constant", "parseAssignment")

		// Skip until next line.
		p.advance("parseAssignment error =") // consume =
		_ = p.expression(ctx, symbol.Type())

		return ast.ZeroNodeIndex
	case ast.QualifierType:
		p.error(p.prev(), "cannot assign to a type identifier", "parseAssignment")
		return ast.ZeroNodeIndex
	}

	assignmentToken := p.this()

	p.advance("parseAssignment") // consume '='

	expr := p.expression(ctx, symbol.Type())
	if expr == ast.ZeroExprIndex {
		return ast.ZeroNodeIndex
	}

	exprType := p.ast.Expr(expr).Type()

	if symbol.Identifier.Name != "_" &&
		// TODO: check if this is required, as [Parser.expression] should already enforce this, expect if symbol type is [types.None].
		!types.Equal(symbol.Type(), exprType) &&
		!types.AssignableTo(exprType, symbol.Type()) {
		p.error(assignmentToken, fmt.Sprintf("type mismatch: cannot assign %q to variable of type %q", exprType, symbol.Type()), "parseAssignment")
		return ast.ZeroNodeIndex
	}

	if symbol.Identifier.Name != "_" &&
		(symbol.Identifier.ValueType == nil || symbol.Identifier.ValueType == types.None) {
		symbol.Identifier.ValueType = exprType
	}

	if symbol.Identifier.Name != "_" && symbol.Type() == types.None {
		// TODO: this seems unnecessary, as we do the same just above.
		p.symbols.Update(ident.Name, exprType)
	}

	// Static result analysis: if the assigned expression's type matches the
	// result's value or error type, we know statically which variant it is.
	// Wrap in ResultLiteral so the transpiler emits the correct Go struct.
	if resultType, ok := exprType.Underlying().(*types.Result); ok {
		if state, isVariant := resultExprState(resultType, exprType); isVariant {
			expr = p.ast.NewResultLiteral(assignmentToken, exprType, expr, exprType.Kind() == types.ErrorKind)

			p.symbols.MarkChecked(ident.Name, state)
		} else {
			// Reassignment from an unknown result variant invalidates previous checks.
			p.symbols.ClearChecked(ident.Name)
		}
	}

	return p.ast.NewAssignment(assignmentToken, symbol.Identifier, expr)
}
