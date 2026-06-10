package parser

import (
	"context"
	"fmt"

	"github.com/samborkent/cog/internal/ast"
	"github.com/samborkent/cog/internal/tokens"
	"github.com/samborkent/cog/internal/types"
)

func (p *Parser) parseTypedDeclaration(ctx context.Context, ident *ast.Identifier) ast.NodeIndex {
	declToken := p.lex.Peek(-1)

	// Handle qualifier after colon: c : var Type or c : dyn Type
	switch p.lex.This().Type {
	case tokens.Variable:
		if p.symbols.Outer == nil && !p.scriptMode {
			p.error(p.lex.This(), "variable declarations are not allowed in package scope", "parseTypedDeclaration")
			return ast.ZeroNodeIndex
		}
		ident.Qualifier = ast.QualifierVariable
		p.lex.Step() // consume var
	case tokens.Dynamic:
		if p.symbols.Outer != nil {
			p.error(p.lex.This(), "dynamic scope variable declarations are only allowed in package scope", "parseTypedDeclaration")
			return ast.ZeroNodeIndex
		}
		ident.Qualifier = ast.QualifierDynamic
		p.lex.Step() // consume dyn
	}

	// If the qualifier was consumed and the next token is = or :=,
	// this is an untyped var/dyn declaration (x : var = expr).
	if ident.Qualifier != ast.QualifierImmutable &&
		(p.lex.This().Type == tokens.Assign || p.lex.This().Type == tokens.Declaration) {
		return p.parseDeclaration(ctx, declToken, ident)
	}

	identType := p.parseCombinedType(ctx, ident.Exported, ident.Global)
	if identType == nil {
		return ast.ZeroNodeIndex
	}

	ident.ValueType = identType

	return p.parseDeclaration(ctx, declToken, ident)
}

func (p *Parser) parseDeclaration(ctx context.Context, declToken tokens.Token, ident *ast.Identifier) ast.NodeIndex {
	symbol, ok := p.symbols.ResolveIdent(ident.Token.Literal)
	if ok {
		if symbol.Scope != ScanScope {
			p.error(ident.Token, "cannot redeclare variable", "parseDeclaration")
			return ast.ZeroNodeIndex
		}

		// During globalsPass, ScanScope with a resolved type means duplicate.
		if p.globalsPass && !types.IsNone(symbol.Identifier.ValueType) {
			p.error(ident.Token, "cannot redeclare variable", "parseDeclaration")
			return ast.ZeroNodeIndex
		}
	}

	if ident.Token.Literal == "main" {
		procType, isProc := ident.ValueType.(*types.Procedure)
		if !isProc || procType.Function || len(procType.Parameters) != 0 || procType.ReturnType != nil {
			p.error(ident.Token, `"main" can only be declared as proc()`, "parseDeclaration")
			return ast.ZeroNodeIndex
		}
	}

	if ident.ValueType == nil {
		ident.ValueType = types.None
	}

	assignToken := p.lex.This()
	assignment := &ast.Assignment{
		Token:      assignToken,
		Identifier: ident,
	}

	if !p.match(assignToken, tokens.Assign, tokens.Declaration) {
		if ident.Qualifier == ast.QualifierImmutable {
			p.error(assignToken, "immutable declarations must be initialized", "parseDeclaration")
			return ast.ZeroNodeIndex
		}

		// Uninitialized variable
		if p.globalsPass {
			p.symbols.DefineGlobalIdent(ident)
		} else {
			p.symbols.DefineIdent(ident)
		}

		return p.ast.NewDeclaration(declToken, assignment)
	}

	p.lex.Step() // consume := or =

	expr := p.expression(ctx, ident.ValueType)
	if expr == ast.ZeroExprIndex {
		return ast.ZeroNodeIndex
	}

	assignment.Expr = expr
	exprType := p.ast.Expr(expr).Type()

	// Rule 4: var to var assignment (via declaration) consumes source.
	if sourceIdent, ok := p.ast.Expr(expr).(*ast.Identifier); ok &&
		sourceIdent.Qualifier == ast.QualifierVariable &&
		sourceIdent.Token.Literal != "_" {
		if !p.symbols.IsAlive(sourceIdent.Token.Literal) {
			p.error(assignToken, fmt.Sprintf("cannot assign %s: variable is consumed", sourceIdent.Token.Literal), "parseDeclaration")
			return ast.ZeroNodeIndex
		}
		p.symbols.MarkConsumed(sourceIdent.Token.Literal)
	}

	// Field selector as RHS in declaration consumes that field.
	if selector, ok := p.ast.Expr(expr).(*ast.Selector); ok {
		if len(selector.Fields) > 0 {
			structVar := selector.Fields[0].Token.Literal
			fieldName := selector.Fields[len(selector.Fields)-1].Token.Literal

			sym, ok := p.symbols.ResolveIdent(structVar)
			if ok && sym.Identifier.Qualifier == ast.QualifierVariable {
				if err := p.symbols.MarkFieldConsumed(structVar, fieldName); err != nil {
					p.error(assignToken, err.Error(), "parseDeclaration")
					return ast.ZeroNodeIndex
				}
			}
		}
	}

	if ident.ValueType == types.None {
		ident.ValueType = exprType
		assignment.Identifier.ValueType = exprType
	}

	if p.globalsPass {
		p.symbols.DefineGlobalIdent(ident)
	} else {
		p.symbols.DefineIdent(ident)
	}

	// Static result analysis: if the assigned expression's type matches the
	// result's value or error type, we know statically which variant it is.
	// Wrap in ResultLiteral so the transpiler emits the correct Go struct.
	if resultType, ok := ident.ValueType.Underlying().(*types.Result); ok {
		if state, isVariant := resultExprState(resultType, exprType); isVariant {
			assignment.Expr = p.ast.NewResultLiteral(assignment.Token, ident.ValueType, expr, exprType.Kind() == types.ErrorKind)
			p.symbols.MarkChecked(ident.Token.Literal, state)
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
