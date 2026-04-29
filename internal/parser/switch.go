package parser

import (
	"context"

	"github.com/samborkent/cog/internal/ast"
	"github.com/samborkent/cog/internal/tokens"
	"github.com/samborkent/cog/internal/types"
)

const switchCasePreallocationSize = 4

func (p *Parser) parseSwitch(ctx context.Context) ast.NodeIndex {
	var label *ast.Identifier

	if p.prev().Type == tokens.Identifier && p.this().Type == tokens.Colon {
		label = &ast.Identifier{
			Token: p.prev(),
			Name:  p.prev().Literal,
		}

		p.advance("parseSwitch :") // consume colon
	}

	p.advance("parseSwitch switch") // consume switch

	switch p.this().Type {
	case tokens.Identifier:
		return p.parseIdentSwitch(ctx, label)
	case tokens.LBrace:
		return p.parseBoolSwitch(ctx, label)
	default:
		p.error(p.this(), "unexpected token after switch", "parseSwitch")
		return ast.ZeroNodeIndex
	}
}

func (p *Parser) parseBoolSwitch(ctx context.Context, label *ast.Identifier) ast.NodeIndex {
	switchToken := p.prev()

	p.advance("parseBoolSwitch {") // consume {

	cases := make([]*ast.Case, 0, switchCasePreallocationSize)

	for p.this().Type == tokens.Case {
		caseNode := &ast.Case{
			Token: p.this(),
		}

		p.advance("parseBoolSwitch case") // consume case

		expr := p.expression(ctx, types.None)
		if expr == ast.ZeroExprIndex {
			p.error(p.this(), "unable to parse case expression", "parseBoolSwitch")
			return ast.ZeroNodeIndex
		}

		caseNode.Condition = expr

		if p.this().Type != tokens.Colon {
			p.error(p.this(), "expected ':' after case condition", "parseBoolSwitch")
			return ast.ZeroNodeIndex
		}

		p.advance("parseBoolSwitch case :") // consume :

		for !p.match(tokens.Case, tokens.Default, tokens.RBrace, tokens.EOF) {
			if ctx.Err() != nil {
				return ast.ZeroNodeIndex
			}

			prev := p.i

			stmt := p.parseStatement(ctx)
			if stmt != ast.ZeroNodeIndex {
				caseNode.Body = append(caseNode.Body, stmt)
			} else {
				p.synchronize()
			}

			if p.i == prev {
				p.advance("parseBoolSwitch case recovery")
			}
		}

		cases = append(cases, caseNode)
	}

	var def *ast.Default

	if p.this().Type == tokens.Default {
		defaultNode := &ast.Default{
			Token: p.this(),
		}

		p.advance("parseBoolSwitch default") // consume default

		if p.this().Type != tokens.Colon {
			p.error(p.this(), "expected ':' after default", "parseBoolSwitch")
			return ast.ZeroNodeIndex
		}

		p.advance("parseBoolSwitch default :") // consume :

		for !p.match(tokens.RBrace, tokens.EOF) {
			if ctx.Err() != nil {
				return ast.ZeroNodeIndex
			}

			prev := p.i

			stmt := p.parseStatement(ctx)
			if stmt != ast.ZeroNodeIndex {
				defaultNode.Body = append(defaultNode.Body, stmt)
			} else {
				p.synchronize()
			}

			if p.i == prev {
				p.advance("parseBoolSwitch default recovery")
			}
		}

		def = defaultNode
	}

	p.advance("parseBoolSwitch }") // consume }

	return p.ast.NewSwitch(switchToken, label, nil, cases, def)
}

func (p *Parser) parseIdentSwitch(ctx context.Context, label *ast.Identifier) ast.NodeIndex {
	switchToken := p.prev()

	symbol, ok := p.symbols.Resolve(p.this().Literal)
	if !ok {
		p.error(p.this(), "unknown identifier in switch expression", "parseIdentSwitch")
		return ast.ZeroNodeIndex
	}

	p.advance("parseIdentSwitch") // consume identifier

	if p.this().Type != tokens.LBrace {
		p.error(p.this(), "expected '{' after switch expression", "parseIdentSwitch")
		return ast.ZeroNodeIndex
	}

	p.advance("parseIdentSwitch {") // consume {

	cases := make([]*ast.Case, 0, switchCasePreallocationSize)

	for p.this().Type == tokens.Case {
		caseNode := &ast.Case{
			Token: p.this(),
		}

		p.advance("parseIdentSwitch case") // consume case

		cond := p.expression(ctx, symbol.Type())
		if cond == ast.ZeroExprIndex {
			p.error(p.this(), "unable to parse case expression", "parseIdentSwitch")
			return ast.ZeroNodeIndex
		}

		if p.ast.Expr(cond).Type() != symbol.Type() {
			p.error(p.this(), "case condition type does not match switch expression type", "parseIdentSwitch")
			return ast.ZeroNodeIndex
		}

		caseNode.Condition = cond

		if p.this().Type != tokens.Colon {
			p.error(p.this(), "expected ':' after case condition", "parseIdentSwitch")
			return ast.ZeroNodeIndex
		}

		p.advance("parseIdentSwitch case :") // consume :

		for !p.match(tokens.Case, tokens.Default, tokens.RBrace, tokens.EOF) {
			if ctx.Err() != nil {
				return ast.ZeroNodeIndex
			}

			prev := p.i

			stmt := p.parseStatement(ctx)
			if stmt != ast.ZeroNodeIndex {
				caseNode.Body = append(caseNode.Body, stmt)
			} else {
				p.synchronize()
			}

			if p.i == prev {
				p.advance("parseIdentSwitch case recovery")
			}
		}

		cases = append(cases, caseNode)
	}

	var def *ast.Default

	if p.this().Type == tokens.Default {
		defaultNode := &ast.Default{
			Token: p.this(),
		}

		p.advance("parseIdentSwitch default") // consume default

		if p.this().Type != tokens.Colon {
			p.error(p.this(), "expected ':' after default", "parseIdentSwitch")
			return ast.ZeroNodeIndex
		}

		p.advance("parseIdentSwitch default :") // consume :

		for !p.match(tokens.RBrace, tokens.EOF) {
			if ctx.Err() != nil {
				return ast.ZeroNodeIndex
			}

			prev := p.i

			stmt := p.parseStatement(ctx)
			if stmt != ast.ZeroNodeIndex {
				defaultNode.Body = append(defaultNode.Body, stmt)
			} else {
				p.synchronize()
			}

			if p.i == prev {
				p.advance("parseIdentSwitch default recovery")
			}
		}

		def = defaultNode
	}

	p.advance("parseIdentSwitch }") // consume }

	return p.ast.NewSwitch(switchToken, label, symbol.Identifier, cases, def)
}
