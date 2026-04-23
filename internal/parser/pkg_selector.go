package parser

import (
	"context"
	"fmt"

	"github.com/samborkent/cog/internal/ast"
	"github.com/samborkent/cog/internal/tokens"
	"github.com/samborkent/cog/internal/types"
)

// parsePkgSelector parses an imported package selector expression: pkg.Symbol
// The cursor is on the package name identifier.
func (p *Parser) parsePkgSelector(ctx context.Context, imp *CogImport) ast.ExprValue {
	pkgToken := p.this()

	p.advance("parsePkgSelector pkg") // consume package name

	if p.this().Type != tokens.Dot {
		p.error(p.this(), "expected '.' after package name", "parsePkgSelector")
		return ast.ZeroExpr
	}

	p.advance("parsePkgSelector .") // consume '.'

	if p.this().Type != tokens.Identifier {
		p.error(p.this(), "expected identifier after package selector", "parsePkgSelector")
		return ast.ZeroExpr
	}

	symbolName := p.this().Literal

	// Look up the exported symbol from the imported package.
	sym, ok := imp.Exports[symbolName]
	if !ok {
		p.error(p.this(), fmt.Sprintf("package %q has no exported symbol %q", imp.Name, symbolName), "parsePkgSelector")
		return ast.ZeroExpr
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

	// If followed by '(', this is a function call: pkg.Func(args)
	if p.this().Type == tokens.LParen {
		procType, isProc := sym.Identifier.ValueType.(*types.Procedure)
		if !isProc {
			p.error(p.this(), fmt.Sprintf("%s.%s is not callable", imp.Name, symbolName), "parsePkgSelector")
			return ast.ZeroExpr
		}

		return ast.NewExpr(ast.KindCall, procType.ReturnType.Kind(), &ast.Call{
			Expr:       ast.NewExpr(ast.KindIdentifier, fieldIdent.ValueType.Kind(), fieldIdent),
			Package:    imp.Name,
			Arguments:  p.parseCallArguments(ctx, procType),
			ReturnType: procType.ReturnType,
		})
	}

	// Otherwise it's a value/type selector: pkg.Value
	return ast.NewExpr(ast.KindSelector, fieldIdent.ValueType.Kind(), &ast.Selector{
		Token: pkgToken,
		Expr:  ast.NewExpr(ast.KindIdentifier, pkgIdent.ValueType.Kind(), pkgIdent),
		Field: fieldIdent,
	})
}
