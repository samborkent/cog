package parser

import (
	"context"
	"fmt"

	"github.com/samborkent/cog/internal/ast"
	"github.com/samborkent/cog/internal/types"
)

func (p *Parser) parseAssignment(ctx context.Context, ident *ast.Identifier) ast.NodeValue {
	symbol, ok := p.symbols.Resolve(ident.Name)
	if !ok {
		p.error(p.prev(), "unknown identifier", "parseAssignment")
		return ast.ZeroNode
	}

	switch symbol.Identifier.Qualifier {
	case ast.QualifierImmutable:
		p.error(p.prev(), "cannot reassign a constant", "parseAssignment")

		// Skip until next line.
		p.advance("parseAssignment error =") // consume =
		_ = p.expression(ctx, symbol.Type())

		return ast.ZeroNode
	case ast.QualifierType:
		p.error(p.prev(), "cannot assign to a type identifier", "parseAssignment")
		return ast.ZeroNode
	}

	node := &ast.Assignment{
		Token:      p.this(),
		Identifier: symbol.Identifier,
	}

	p.advance("parseAssignment") // consume '='

	expr := p.expression(ctx, symbol.Type())
	if expr == ast.ZeroExpr {
		return ast.ZeroNode
	}

	// TODO: fill in node kind
	node.Expr = expr

	if symbol.Identifier.Name != "_" && !types.Equal(symbol.Type(), expr.Type()) && !types.AssignableTo(expr.Type(), symbol.Type()) {
		p.error(node.Token, fmt.Sprintf("type mismatch: cannot assign %q to variable of type %q", expr.Type(), symbol.Type()), "parseAssignment")
		return ast.ZeroNode
	}

	if symbol.Identifier.Name != "_" && (node.Identifier.ValueType == nil || node.Identifier.ValueType == types.None) {
		node.Identifier.ValueType = expr.Type()
	}

	if symbol.Identifier.Name != "_" && symbol.Type() == types.None {
		p.symbols.Update(ident.Name, expr.Type())
	}

	// Static result analysis: if the assigned expression's type matches the
	// result's value or error type, we know statically which variant it is.
	// Wrap in ResultLiteral so the transpiler emits the correct Go struct.
	if resultType, ok := symbol.Type().Underlying().(*types.Result); ok {
		if state, isVariant := resultExprState(resultType, expr); isVariant {
			node.Expr = wrapResultLiteral(node.Token, symbol.Type(), expr)

			p.symbols.MarkChecked(ident.Name, state)
		} else {
			// Reassignment from an unknown result variant invalidates previous checks.
			p.symbols.ClearChecked(ident.Name)
		}
	}

	return ast.NewNode(ast.KindAssignment, node)
}
