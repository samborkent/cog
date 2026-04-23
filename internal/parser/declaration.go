package parser

import (
	"context"

	"github.com/samborkent/cog/internal/ast"
	"github.com/samborkent/cog/internal/tokens"
	"github.com/samborkent/cog/internal/types"
)

func (p *Parser) parseTypedDeclaration(ctx context.Context, ident *ast.Identifier) ast.NodeValue {
	identType := p.parseCombinedType(ctx, ident.Exported, ident.Global)
	if identType == nil {
		return ast.ZeroNode
	}

	ident.ValueType = identType

	return p.parseDeclaration(ctx, ident)
}

func (p *Parser) parseDeclaration(ctx context.Context, ident *ast.Identifier) ast.NodeValue {
	symbol, ok := p.symbols.Resolve(ident.Name)
	if ok && symbol.Scope != ScanScope && ident.Qualifier != ast.QualifierMethod {
		p.error(ident.Token, "cannot redeclare variable", "parseDeclaration")
		return ast.ZeroNode
	}

	if ident.Name == "main" {
		procType, isProc := ident.ValueType.(*types.Procedure)
		if !isProc || procType.Function || len(procType.Parameters) != 0 || procType.ReturnType != nil {
			p.error(ident.Token, `"main" can only be declared as proc()`, "parseDeclaration")
			return ast.ZeroNode
		}
	}

	if ident.ValueType == nil {
		ident.ValueType = types.None
	}

	node := &ast.Declaration{
		Assignment: &ast.Assignment{
			Token:      p.this(),
			Identifier: ident,
		},
	}

	if !p.match(tokens.Assign, tokens.Declaration) {
		if ident.Qualifier == ast.QualifierImmutable {
			p.error(p.this(), "immutable declarations must be initialized", "parseDeclaration")
			return ast.ZeroNode
		}

		// Uninitialized variable
		p.symbols.Define(ident)

		return ast.NewNode(ast.KindDeclaration, node)
	}

	p.advance("parseDeclaration") // consume := or =

	expr := p.expression(ctx, ident.ValueType)
	if expr == ast.ZeroExpr {
		return ast.ZeroNode
	}

	node.Assignment.Expr = expr

	if ident.ValueType == types.None {
		exprType := expr.Type()

		ident.ValueType = exprType
		node.Assignment.Identifier.ValueType = exprType
	}

	if ident.Qualifier != ast.QualifierMethod {
		p.symbols.Define(ident)
	}

	// Static result analysis: if the assigned expression's type matches the
	// result's value or error type, we know statically which variant it is.
	// Wrap in ResultLiteral so the transpiler emits the correct Go struct.
	if resultType, ok := ident.ValueType.Underlying().(*types.Result); ok {
		if state, isVariant := resultExprState(resultType, expr); isVariant {
			node.Assignment.Expr = wrapResultLiteral(node.Assignment.Token, ident.ValueType, expr)
			p.symbols.MarkChecked(ident.Name, state)
		}
	}

	return ast.NewNode(ast.KindDeclaration, node)
}

// resultExprState checks whether an expression assigned to a result type
// is a valid value or error variant and returns the corresponding check state.
// Returns (state, true) if the expression matches a variant, or (0, false)
// if it matches the full result type (e.g. a function call returning T ! E).
func resultExprState(resolved *types.Result, expr ast.ExprValue) (checkState, bool) {
	if expr.TypeKind == types.ErrorKind {
		return checkError, true
	}

	if types.Equal(expr.Type(), resolved.Value) || types.AssignableTo(expr.Type(), resolved.Value) {
		return checkValue, true
	}

	return 0, false
}

// wrapResultLiteral wraps an expression in a ResultLiteral for assignment
// to a result-typed variable. It determines whether the expression is the
// error or value variant based on the expression type's kind.
// Returns nil if the expression type doesn't match either variant.
func wrapResultLiteral(tok tokens.Token, resultType types.Type, expr ast.ExprValue) ast.ExprValue {
	isError := expr.Type().Kind() == types.ErrorKind

	return ast.NewExpr(ast.KindResultLiteral, types.ResultKind, &ast.ResultLiteral{
		Token:      tok,
		ResultType: resultType,
		Value:      expr,
		IsError:    isError,
	})
}
