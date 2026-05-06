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
	pkgToken := p.this()

	p.advance("parsePkgSelector pkg") // consume package name

	if p.this().Type != tokens.Dot {
		p.error(p.this(), "expected '.' after package name", "parsePkgSelector")
		return ast.ZeroExprIndex
	}

	selToken := p.this()

	p.advance("parsePkgSelector .") // consume '.'

	if p.this().Type != tokens.Identifier {
		p.error(p.this(), "expected identifier after package selector", "parsePkgSelector")
		return ast.ZeroExprIndex
	}

	symbolName := p.this().Literal

	// Look up the exported symbol from the imported package.
	sym, ok := imp.Exports[symbolName]
	if !ok {
		p.error(p.this(), fmt.Sprintf("package %q has no exported symbol %q", imp.Name, symbolName), "parsePkgSelector")
		return ast.ZeroExprIndex
	}

	pkgIdent := &ast.Identifier{
		Token:     pkgToken,
		Name:      imp.Name,
		ValueType: types.None,
	}

	fieldIdent := &ast.Identifier{
		Token:     p.this(),
		Name:      symbolName,
		ValueType: sym.Identifier.ValueType,
		Exported:  true,
	}

	p.advance("parsePkgSelector symbol") // consume symbol identifier

	sel := p.ast.NewSelector(selToken, pkgIdent, fieldIdent)

	// If followed by '(', this is a function call: pkg.Func(args)
	if p.this().Type == tokens.LParen {
		procType, isProc := sym.Identifier.ValueType.(*types.Procedure)
		if !isProc {
			p.error(p.this(), fmt.Sprintf("%s.%s is not callable", imp.Name, symbolName), "parsePkgSelector")
			return ast.ZeroExprIndex
		}

		return p.ast.NewCall(p.this(), sel, p.parseCallArguments(ctx, procType), procType.ReturnType)
	}

	// Otherwise it's a value/type selector: pkg.Value
	return sel
}
