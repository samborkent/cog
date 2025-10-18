package parser

import (
	"context"

	"github.com/samborkent/cog/internal/ast"
	"github.com/samborkent/cog/internal/tokens"
	"github.com/samborkent/cog/internal/types"
)

func (p *Parser) parseSwitch(ctx context.Context) *ast.Switch {
	p.advance("parseSwitch switch") // consume switch

	switch p.this().Type {
	case tokens.Identifier:
		return p.parseIdentSwitch(ctx)
	case tokens.LBrace:
		return p.parseBoolSwitch(ctx)
	default:
		p.error(p.this(), "unexpected token after switch")
		return nil
	}
}

func (p *Parser) parseBoolSwitch(ctx context.Context) *ast.Switch {
	node := &ast.Switch{
		Token: p.prev(),
	}

	p.advance("parseBoolSwitch {") // consume {

	for p.this().Type == tokens.Case {
		caseNode := &ast.Case{
			Token: p.this(),
		}

		p.advance("parseBoolSwitch case") // consume case

		expr := p.expression(ctx, types.None)
		if expr == nil {
			p.error(p.this(), "unable to parse case expression")
			return nil
		}

		caseNode.Condition = expr

		if p.this().Type != tokens.Colon {
			p.error(p.this(), "expected ':' after case condition")
			return nil
		}

		p.advance("parseBoolSwitch case :") // consume :

		for !p.match(tokens.Case, tokens.Default, tokens.RBrace) {
			if ctx.Err() != nil {
				return nil
			}

			stmt := p.parseStatement(ctx)
			if stmt != nil {
				caseNode.Body = append(caseNode.Body, stmt)
			}
		}

		node.Cases = append(node.Cases, caseNode)
	}

	if p.this().Type == tokens.Default {
		defaultNode := &ast.Default{
			Token: p.this(),
		}

		p.advance("parseBoolSwitch default") // consume default

		if p.this().Type != tokens.Colon {
			p.error(p.this(), "expected ':' after default")
			return nil
		}

		p.advance("parseBoolSwitch default :") // consume :

		for p.this().Type != tokens.RBrace {
			if ctx.Err() != nil {
				return nil
			}

			stmt := p.parseStatement(ctx)
			if stmt != nil {
				defaultNode.Body = append(defaultNode.Body, stmt)
			}
		}

		node.Default = defaultNode
	}

	p.advance("parseBoolSwitch }") // consume }

	return node
}

func (p *Parser) parseIdentSwitch(ctx context.Context) *ast.Switch {
	node := &ast.Switch{
		Token: p.prev(),
	}

	symbol, ok := p.symbols.Resolve(p.this().Literal)
	if !ok {
		p.error(p.this(), "unknown identifier in switch expression")
		return nil
	}

	node.Identifier = symbol.Identifier

	p.advance("parseIdentSwitch") // consume identifier

	if p.this().Type != tokens.LBrace {
		p.error(p.this(), "expected '{' after switch expression")
		return nil
	}

	p.advance("parseIdentSwitch {") // consume {

	for p.this().Type == tokens.Case {
		caseNode := &ast.Case{
			Token: p.this(),
		}

		p.advance("parseIdentSwitch case") // consume case

		cond := p.expression(ctx, symbol.Type())
		if cond == nil {
			p.error(p.this(), "unable to parse case expression")
			return nil
		}

		if cond.Type() != symbol.Type() {
			p.error(p.this(), "case condition type does not match switch expression type")
			return nil
		}

		caseNode.Condition = cond

		if p.this().Type != tokens.Colon {
			p.error(p.this(), "expected ':' after case condition")
			return nil
		}

		p.advance("parseIdentSwitch case :") // consume :

		for !p.match(tokens.Case, tokens.Default, tokens.RBrace) {
			if ctx.Err() != nil {
				return nil
			}

			stmt := p.parseStatement(ctx)
			if stmt != nil {
				caseNode.Body = append(caseNode.Body, stmt)
			}
		}

		node.Cases = append(node.Cases, caseNode)
	}

	if p.this().Type == tokens.Default {
		defaultNode := &ast.Default{
			Token: p.this(),
		}

		p.advance("parseIdentSwitch default") // consume default

		if p.this().Type != tokens.Colon {
			p.error(p.this(), "expected ':' after default")
			return nil
		}

		p.advance("parseIdentSwitch default :") // consume :

		for p.this().Type != tokens.RBrace {
			if ctx.Err() != nil {
				return nil
			}

			stmt := p.parseStatement(ctx)
			if stmt != nil {
				defaultNode.Body = append(defaultNode.Body, stmt)
			}
		}

		node.Default = defaultNode
	}

	p.advance("parseIdentSwitch }") // consume }

	return node
}
