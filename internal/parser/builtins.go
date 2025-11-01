package parser

import (
	"context"
	"fmt"

	"github.com/samborkent/cog/internal/ast"
	"github.com/samborkent/cog/internal/tokens"
	"github.com/samborkent/cog/internal/types"
)

type BuiltinParser func(ctx context.Context, t tokens.Token, tokenType types.Type) *ast.Builtin

func (p *Parser) parseBuiltinIf(ctx context.Context, t tokens.Token, tokenType types.Type) *ast.Builtin {
	if p.this().Type != tokens.LParen {
		p.error(p.this(), "expected '(' after @if", "parseIf")
		return nil
	}

	p.advance("parseIf (") // consume (

	// condition := p.expression(ctx, types.Basics[types.Bool])
	condition := p.expression(ctx, types.None)
	if condition == nil {
		return nil
	}

	if p.this().Type != tokens.Comma {
		p.error(p.this(), "expected ',' after condition in @if", "parseIf")
		return nil
	}

	p.advance("parseIf , condition") // consume ,

	thenExpr := p.expression(ctx, tokenType)
	if thenExpr == nil {
		return nil
	}

	args := []ast.Expression{condition, thenExpr}

	if p.this().Type == tokens.Comma {
		p.advance("parseIf , then") // consume ,

		elseExpr := p.expression(ctx, tokenType)
		if elseExpr == nil {
			return nil
		}

		if thenExpr.Type() != elseExpr.Type() {
			p.error(t, fmt.Sprintf("type mismatch in @if branches: then is %q, else is %q", thenExpr.Type(), elseExpr.Type()), "parseIf")
			return nil
		}

		args = append(args, elseExpr)
	}

	if p.this().Type != tokens.RParen {
		p.error(p.this(), "expected ')' after else expression in @if", "parseIf")
		return nil
	}

	p.advance("parseIf )") // consume ')'

	return &ast.Builtin{
		Token:      t,
		Name:       "if",
		ReturnType: thenExpr.Type(),
		Arguments:  args,
	}
}

func (p *Parser) parseBuiltinPrint(ctx context.Context, t tokens.Token, tokenType types.Type) *ast.Builtin {
	if p.this().Type != tokens.LParen {
		p.error(p.this(), "expected '(' after @print", "parsePrint")
		return nil
	}

	p.advance("parsePrint (") // consume (

	arg := p.expression(ctx, tokenType)
	if arg == nil {
		return nil
	}

	// TODO: implement string formatting

	if p.this().Type != tokens.RParen {
		p.error(p.this(), "expected ')' after argument in @print", "parsePrint")
		return nil
	}

	p.advance("parsePrint )") // consume ')'

	return &ast.Builtin{
		Token:      t,
		Name:       "print",
		ReturnType: types.None,
		Arguments:  []ast.Expression{arg},
	}
}
