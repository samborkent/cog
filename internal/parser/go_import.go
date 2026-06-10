package parser

import (
	"context"

	"github.com/samborkent/cog/internal/ast"
	"github.com/samborkent/cog/internal/tokens"
	"github.com/samborkent/cog/internal/types"
)

const importPreallocationSize = 4

func (p *Parser) parseGoImport(ctx context.Context) ast.NodeIndex {
	importToken := p.lex.This()
	imports := make([]*ast.Identifier, 0, importPreallocationSize)

	p.lex.Step() // consume 'goimport'

	if p.lex.This().Type != tokens.LParen {
		p.error(p.lex.This(), "expected '(' after goimport", "parseGoImport")
		return ast.ZeroNodeIndex
	}

	p.lex.Step() // consume '('

	for p.lex.This().Type != tokens.EOF && ctx.Err() == nil {
		t := p.lex.This()
		if t.Type == tokens.RParen {
			break
		}

		if t.Type == tokens.Newline {
			p.lex.Step()
			continue
		}

		if t.Type != tokens.StringLiteral {
			p.error(t, "found non-string token in goimport list: "+t.Literal, "parseGoImport")
			return ast.ZeroNodeIndex
		}

		_, ok := p.symbols.ResolveGoImport(t.Literal)
		if ok {
			p.error(t, "cannot redeclare Go imports", "parseGoImport")
			return ast.ZeroNodeIndex
		}

		ident := &ast.Identifier{
			Token: t,
		}

		imports = append(imports, ident)
		p.symbols.DefineGoImport(ident)

		p.lex.Step()
	}

	p.lex.Step() // consume ')'

	return p.ast.NewGoImport(importToken, imports)
}

func (p *Parser) parseGoCallExpression(ctx context.Context) ast.ExprIndex {
	if p.inPureFunc {
		p.error(p.lex.This(), "cannot use @go inside func (pure function)", "parseGoCallExpression")
	}

	expr := ast.New[ast.GoCallExpression](p.ast)
	expr.Token = p.lex.This()

	p.lex.Step() // consume @go

	if p.lex.This().Type != tokens.Dot {
		p.error(p.lex.This(), "expected '.' after @go", "parseGoCallExpression")
		return ast.ZeroExprIndex
	}

	p.lex.Step() // consume .

	if p.lex.This().Type != tokens.Identifier {
		p.error(p.lex.This(), "expected identifier after '.' in @go call", "parseGoCallExpression")
		return ast.ZeroExprIndex
	}

	_, ok := p.symbols.ResolveGoImport(p.lex.This().Literal)
	if !ok {
		p.error(p.lex.This(), "undefined Go import", "parseGoCallExpression")
	}

	// TODO: handle identifiers.
	expr.Import = &ast.Identifier{
		Token: p.lex.This(),
	}

	p.lex.Step() // consume import identifier

	if p.lex.This().Type != tokens.Dot {
		p.error(p.lex.This(), "expected '.' after Go import", "parseGoCallExpression")
		return ast.ZeroExprIndex
	}

	p.lex.Step() // consume .

	if p.lex.This().Type != tokens.Identifier {
		p.error(p.lex.This(), "expected call after '.' in Go import", "parseGoCallExpression")
		return ast.ZeroExprIndex
	}

	callIdent := &ast.Identifier{
		Token:     p.lex.This(),
		ValueType: types.None, // TODO: figure out how to deal with Go types and type conversion
	}

	p.lex.Step() // consume call identifier

	// TODO: also support imported variables / constants
	if p.lex.This().Type != tokens.LParen {
		p.error(p.lex.This(), "expected '(' after call in Go import", "parseGoCallExpression")
		return ast.ZeroExprIndex
	}

	expr.CallIdentifier = callIdent
	expr.Arguments = p.parseCallArguments(ctx, nil)

	if !ok {
		return ast.ZeroExprIndex
	}

	return p.ast.AddExpr(expr)
}
