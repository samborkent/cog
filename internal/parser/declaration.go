package parser

import (
	"context"

	"github.com/samborkent/cog/internal/ast"
	"github.com/samborkent/cog/internal/tokens"
	"github.com/samborkent/cog/internal/types"
)

func (p *Parser) parseTypedDeclaration(ctx context.Context, ident *ast.Identifier) ast.NodeIndex {
	declToken := p.prev()

	identType := p.parseCombinedType(ctx, ident.Exported, ident.Global)
	if identType == nil {
		return ast.ZeroNodeIndex
	}

	ident.ValueType = identType

	return p.parseDeclaration(ctx, declToken, ident)
}

func (p *Parser) parseDeclaration(ctx context.Context, declToken tokens.Token, ident *ast.Identifier) ast.NodeIndex {
	symbol, ok := p.symbols.Resolve(ident.Name)
	if ok && symbol.Scope != ScanScope && ident.Qualifier != ast.QualifierMethod {
		p.error(ident.Token, "cannot redeclare variable", "parseDeclaration")
		return ast.ZeroNodeIndex
	}

	if ident.Name == "main" {
		procType, isProc := ident.ValueType.(*types.Procedure)
		if !isProc || procType.Function || len(procType.Parameters) != 0 || procType.ReturnType != nil {
			p.error(ident.Token, `"main" can only be declared as proc()`, "parseDeclaration")
			return ast.ZeroNodeIndex
		}
	}

	if ident.ValueType == nil {
		ident.ValueType = types.None
	}

	assignment := &ast.Assignment{
		Token:      p.this(),
		Identifier: ident,
	}

	if !p.match(tokens.Assign, tokens.Declaration) {
		if ident.Qualifier == ast.QualifierImmutable {
			p.error(p.this(), "immutable declarations must be initialized", "parseDeclaration")
			return ast.ZeroNodeIndex
		}

		// Uninitialized variable
		p.symbols.Define(ident)

		return p.ast.NewDeclaration(declToken, assignment)
	}

	p.advance("parseDeclaration") // consume := or =

	expr := p.expression(ctx, ident.ValueType)
	if expr == ast.ZeroExprIndex {
		return ast.ZeroNodeIndex
	}

	assignment.Expr = expr
	exprType := p.ast.Expr(expr).Type()

	if ident.ValueType == types.None {
		ident.ValueType = exprType
		assignment.Identifier.ValueType = exprType
	}

	if ident.Qualifier != ast.QualifierMethod {
		p.symbols.Define(ident)
	}

	// Static result analysis: if the assigned expression's type matches the
	// result's value or error type, we know statically which variant it is.
	// Wrap in ResultLiteral so the transpiler emits the correct Go struct.
	if resultType, ok := ident.ValueType.Underlying().(*types.Result); ok {
		if state, isVariant := resultExprState(resultType, exprType); isVariant {
			assignment.Expr = p.ast.NewResultLiteral(assignment.Token, ident.ValueType, expr, exprType.Kind() == types.ErrorKind)
			p.symbols.MarkChecked(ident.Name, state)
		}
	}

	return p.ast.NewDeclaration(declToken, assignment)
}

// resultExprState checks whether an expression assigned to a result type
// is a valid value or error variant and returns the corresponding check state.
// Returns (state, true) if the expression matches a variant, or (0, false)
// if it matches the full result type (e.g. a function call returning T ! E).
func resultExprState(resolved *types.Result, exprType types.Type) (checkState, bool) {
	if exprType.Kind() == types.ErrorKind {
		return checkError, true
	}

	if types.Equal(exprType, resolved.Value) || types.AssignableTo(exprType, resolved.Value) {
		return checkValue, true
	}

	return 0, false
}
