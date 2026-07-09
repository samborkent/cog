package parser

import (
	"context"
	"fmt"
	"strings"

	"github.com/samborkent/cog/internal/ast"
	"github.com/samborkent/cog/internal/types"
)

func (p *Parser) parseAssignment(ctx context.Context, ident *ast.Identifier) ast.NodeIndex {
	symbol, ok := p.symbols.ResolveIdent(ident.String())
	if !ok {
		// Field assignment: resolve just the struct variable name.
		// TODO: wtf is this?
		name := ident.String()
		if idx := strings.IndexByte(name, '.'); idx >= 0 {
			symbol, ok = p.symbols.ResolveIdent(name[:idx])
		}

		if !ok {
			p.error(p.lex.Peek(-1), "unknown identifier", "parseAssignment")
			return ast.ZeroNodeIndex
		}
	}

	if symbol.Identifier.Qualifier == ast.QualifierImmutable {
		p.error(p.lex.Peek(-1), "cannot reassign a constant", "parseAssignment")

		// Skip until next line.
		p.lex.Step() // consume =
		_ = p.expression(ctx, symbol.Type())

		return ast.ZeroNodeIndex
	}

	assignmentToken := p.lex.This()

	p.lex.Step() // consume '='

	// Use the identifier's embedded type hint if available (e.g. for field assignments),
	// otherwise fall back to the symbol's type.
	typeHint := symbol.Type()
	if ident.ValueType != nil && ident.ValueType != types.None {
		typeHint = ident.ValueType
	}

	expr := p.expression(ctx, typeHint)
	if expr == ast.ZeroExprIndex {
		return ast.ZeroNodeIndex
	}

	exprType := p.ast.Expr(expr).Type()

	// Rule 4: var to var assignment consumes source.
	if sourceIdent, ok := p.ast.Expr(expr).(*ast.Identifier); ok &&
		sourceIdent.Qualifier == ast.QualifierVariable &&
		sourceIdent.Token.Literal != "_" {
		if !p.symbols.IsAlive(sourceIdent.Token.Literal) {
			p.error(assignmentToken, fmt.Sprintf("cannot assign %s: variable is consumed", sourceIdent.Token.Literal), "parseAssignment")
			return ast.ZeroNodeIndex
		}
		p.symbols.MarkConsumed(sourceIdent.Token.Literal)
	}

	// Field selector as RHS consumes that specific field.
	if selector, ok := p.ast.Expr(expr).(*ast.Selector); ok {
		if len(selector.Fields) > 0 {
			structVar := selector.Fields[0].Token.Literal
			fieldName := selector.Fields[len(selector.Fields)-1].Token.Literal

			sym, ok := p.symbols.ResolveIdent(structVar)
			if ok && sym.Identifier.Qualifier == ast.QualifierVariable {
				if err := p.symbols.MarkFieldConsumed(structVar, fieldName); err != nil {
					p.error(assignmentToken, err.Error(), "parseAssignment")
					return ast.ZeroNodeIndex
				}
			}
		}
	}

	if symbol.Identifier.String() != "_" &&
		// TODO: check if this is required, as [Parser.expression] should already enforce this, expect if symbol type is [types.None].
		!types.Equal(typeHint, exprType) &&
		!types.AssignableTo(exprType, typeHint) {
		p.error(assignmentToken, fmt.Sprintf("type mismatch: cannot assign %q to variable of type %q", exprType, typeHint), "parseAssignment")
		return ast.ZeroNodeIndex
	}

	// Static result analysis: if the assigned expression's type matches the
	// result's value or error type, we know statically which variant it is.
	// Wrap in ResultLiteral so the transpiler emits the correct Go struct.
	if resultType, ok := symbol.Identifier.Type().Underlying().(*types.Result); ok {
		if state, isVariant := resultExprState(resultType, exprType); isVariant {
			expr = p.ast.NewResultLiteral(assignmentToken, exprType, expr, exprType.Kind() == types.ErrorKind)

			p.symbols.MarkChecked(ident.String(), state)
		} else {
			// Reassignment from an unknown result variant invalidates previous checks.
			p.symbols.ClearChecked(ident.String())
		}
	}

	if symbol.Identifier.String() != "_" && (symbol.Identifier.ValueType == nil || symbol.Identifier.ValueType == types.None) {
		symbol.Identifier.ValueType = exprType
	}

	return p.ast.NewAssignment(assignmentToken, symbol.Identifier, expr)
}
