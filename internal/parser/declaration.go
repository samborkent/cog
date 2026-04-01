package parser

import (
	"context"

	"github.com/samborkent/cog/internal/ast"
	"github.com/samborkent/cog/internal/tokens"
	"github.com/samborkent/cog/internal/types"
)

// resolveResult unwraps aliases to find an underlying Result type.
func resolveResult(t types.Type) (*types.Result, bool) {
	if r, ok := t.(*types.Result); ok {
		return r, true
	}

	if a, ok := t.(*types.Alias); ok {
		return resolveResult(a.Underlying())
	}

	return nil, false
}

func (p *Parser) parseTypedDeclaration(ctx context.Context, ident *ast.Identifier) *ast.Declaration {
	identType := p.parseCombinedType(ctx, ident.Exported)
	if identType == nil {
		return nil
	}

	ident.ValueType = identType

	node := p.parseDeclaration(ctx, ident)
	if node == nil {
		return nil
	}

	return node
}

func (p *Parser) parseDeclaration(ctx context.Context, ident *ast.Identifier) *ast.Declaration {
	symbol, ok := p.symbols.Resolve(ident.Name)
	if ok && symbol.Scope != ScanScope {
		p.error(ident.Token, "cannot redeclare variable", "parseDeclaration")
		return nil
	}

	if ident.Name == "main" {
		procType, isProc := ident.ValueType.(*types.Procedure)
		if !isProc || procType.Function || len(procType.Parameters) != 0 || procType.ReturnType != nil {
			p.error(ident.Token, `"main" can only be declared as proc()`, "parseDeclaration")
			return nil
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
			return nil
		}

		// Uninitialized variable
		p.symbols.Define(ident)

		return node
	}

	p.advance("parseDeclaration") // consume := or =

	expr := p.expression(ctx, ident.ValueType)
	if expr == nil {
		return nil
	}

	node.Assignment.Expression = expr

	if ident.ValueType == types.None {
		exprType := expr.Type()

		ident.ValueType = exprType
		node.Assignment.Identifier.ValueType = exprType
	}

	p.symbols.Define(ident)

	// Static result analysis: if the assigned expression's type matches the
	// result's value or error type, we know statically which variant it is.
	// Wrap in ResultLiteral so the transpiler emits the correct Go struct.
	if resultType, ok := resolveResult(ident.ValueType); ok {
		if types.Equal(expr.Type(), resultType.Error) {
			node.Assignment.Expression = &ast.ResultLiteral{
				Token:      node.Assignment.Token,
				ResultType: ident.ValueType,
				Value:      expr,
				IsError:    true,
			}
			p.symbols.MarkChecked(ident.Name, checkError)
		} else if !types.Equal(expr.Type(), ident.ValueType) {
			// Expression type is not the full result type (not a function call
			// returning the same result), so it must be the value variant.
			node.Assignment.Expression = &ast.ResultLiteral{
				Token:      node.Assignment.Token,
				ResultType: ident.ValueType,
				Value:      expr,
				IsError:    false,
			}
			p.symbols.MarkChecked(ident.Name, checkValue)
		}
	}

	return node
}
