package parser

import (
	"context"

	"github.com/samborkent/cog/internal/ast"
	"github.com/samborkent/cog/internal/tokens"
	"github.com/samborkent/cog/internal/types"
)

func (p *Parser) parseGoImport() ast.NodeValue {
	node := &ast.GoImport{
		Token:   p.this(),
		Imports: make([]*ast.Identifier, 0),
	}

	p.advance("parseGoImport goimport") // consume 'goimport'

	if p.this().Type != tokens.LParen {
		p.error(p.this(), "expected '(' after goimport", "parseGoImport")
		return ast.ZeroNode
	}

	p.advance("parseGoImport (") // consume '('

	for ; p.this().Type != tokens.RParen && p.this().Type != tokens.EOF; p.advance("parseGoImport loop") {
		if p.this().Type != tokens.StringLiteral {
			p.error(p.this(), "found non-string token in goimport list: "+p.this().Literal, "parseGoImport")
			return ast.ZeroNode
		}

		_, ok := p.symbols.ResolveGoImport(p.this().Literal)
		if ok {
			p.error(p.this(), "cannot redeclare Go imports", "parseGoImport")
			return ast.ZeroNode
		}

		ident := &ast.Identifier{
			Token: p.this(),
			Name:  p.this().Literal,
		}

		node.Imports = append(node.Imports, ident)
		p.symbols.DefineGoImport(ident)
	}

	p.advance("parseGoImport )") // consume ')'

	return ast.NewNode(ast.KindGoImport, node)
}

func (p *Parser) parseGoCallExpression(ctx context.Context) ast.ExprValue {
	node := &ast.GoCallExpression{
		Token: p.this(),
	}

	p.advance("parseGoCallExpression @go") // consume @go

	if p.this().Type != tokens.Dot {
		p.error(p.this(), "expected '.' after @go", "parseGoCallExpression")
		return ast.ZeroExpr
	}

	p.advance("parseGoCallExpression .") // consume .

	if p.this().Type != tokens.Identifier {
		p.error(p.this(), "expected identifier after '.' in @go call", "parseGoCallExpression")
		return ast.ZeroExpr
	}

	_, ok := p.symbols.ResolveGoImport(p.this().Literal)
	if !ok {
		p.error(p.this(), "undefined Go import", "parseGoCallExpression")
	}

	node.Import = &ast.Identifier{
		Token: p.this(),
		Name:  p.this().Literal,
	}

	p.advance("parseGoCallExpression import") // consume import identifier

	if p.this().Type != tokens.Dot {
		p.error(p.this(), "expected '.' after Go import", "parseGoCallExpression")
		return ast.ZeroExpr
	}

	p.advance("parseGoCallExpression import .") // consume .

	if p.this().Type != tokens.Identifier {
		p.error(p.this(), "expected call after '.' in Go import", "parseGoCallExpression")
		return ast.ZeroExpr
	}

	callIdent := &ast.Identifier{
		Token:     p.this(),
		Name:      p.this().Literal,
		ValueType: types.None, // TODO: figure out how to deal with Go types and type conversion
	}

	p.advance("parseGoCallExpression import call") // consume call identifier

	// TODO: also support imported variables / constants
	if p.this().Type != tokens.LParen {
		p.error(p.this(), "expected '(' after call in Go import", "parseGoCallExpression")
		return ast.ZeroExpr
	}

	node.CallIdentifier = callIdent
	node.Arguments = p.parseCallArguments(ctx, nil)

	if !ok {
		return ast.ZeroExpr
	}

	// TODO: resolve Go type
	return ast.NewExpr(ast.KindGoCallExpression, node.Type().Kind(), node)
}
