package parser

import (
	"context"
	"fmt"

	"github.com/samborkent/cog/internal/ast"
	"github.com/samborkent/cog/internal/tokens"
	"github.com/samborkent/cog/internal/types"
)

// TODO: remove this and treat like regular selector?

// parsePkgSelector parses an imported package selector expression: pkg.Symbol
// The cursor is on the package name identifier.
func (p *Parser) parsePkgSelector(ctx context.Context, imp *CogImport) ast.ExprIndex {
	pkgToken := p.lex.This()

	p.lex.Step() // consume package name

	if p.lex.This().Type != tokens.Dot {
		p.error(p.lex.This(), "expected '.' after package name", "parsePkgSelector")
		return ast.ZeroExprIndex
	}

	selToken := p.lex.This()

	p.lex.Step() // consume '.'

	if p.lex.This().Type != tokens.Identifier {
		p.error(p.lex.This(), "expected identifier after package selector", "parsePkgSelector")
		return ast.ZeroExprIndex
	}

	symbolName := p.lex.This().Literal

	// Look up the exported symbol from the imported package.
	// TODO: this only looks up values. Usage of exported types currently doesn't work.
	sym, ok := imp.ExportValues[symbolName]
	if !ok {
		p.error(p.lex.This(), fmt.Sprintf("package %q has no exported symbol %q", imp.Name, symbolName), "parsePkgSelector")
		return ast.ZeroExprIndex
	}

	pkgToken.Literal = imp.Name // ensure package token is the import name
	pkgIdent := &ast.Identifier{
		Token:     pkgToken,
		ValueType: types.None,
	}

	fieldToken := p.lex.This()
	fieldToken.Literal = symbolName // ensure field token is the symbol name

	fieldIdent := &ast.Identifier{
		Token:     fieldToken,
		ValueType: sym.Identifier.ValueType,
		Exported:  true,
	}

	p.lex.Step() // consume symbol identifier

	sel := p.ast.NewSelector(selToken, pkgIdent, fieldIdent)

	// If followed by '(', this is a function call: pkg.Func(args)
	if p.lex.This().Type == tokens.LParen {
		procType, isProc := sym.Identifier.ValueType.(*types.Procedure)
		if !isProc {
			p.error(p.lex.This(), fmt.Sprintf("%s.%s is not callable", imp.Name, symbolName), "parsePkgSelector")
			return ast.ZeroExprIndex
		}

		if p.inPureFunc && !procType.Function {
			p.error(p.lex.This(), "calling proc inside func is not allowed", "primary")
			return ast.ZeroExprIndex
		}

		return p.ast.NewCall(p.lex.This(), sel, p.parseCallArguments(ctx, procType), procType.ReturnType)
	}

	// Otherwise it's a value/type selector: pkg.Value
	return sel
}
