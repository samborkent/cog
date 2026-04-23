package parser

import (
	"context"

	"github.com/samborkent/cog/internal/ast"
	"github.com/samborkent/cog/internal/tokens"
	"github.com/samborkent/cog/internal/types"
)

func (p *Parser) parseSwitch(ctx context.Context, labelIdent *ast.Identifier) ast.NodeValue {
	p.advance("parseSwitch switch") // consume switch

	switch p.this().Type {
	case tokens.Identifier:
		return p.parseIdentSwitch(ctx, labelIdent)
	case tokens.LBrace:
		return p.parseBoolSwitch(ctx, labelIdent)
	default:
		p.error(p.this(), "unexpected token after switch", "parseSwitch")
		return ast.ZeroNode
	}
}

func (p *Parser) parseBoolSwitch(ctx context.Context, labelIdent *ast.Identifier) ast.NodeValue {
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
		if expr == ast.ZeroExpr {
			p.error(p.this(), "unable to parse case expression", "parseBoolSwitch")
			return ast.ZeroNode
		}

		caseNode.Condition = expr

		if p.this().Type != tokens.Colon {
			p.error(p.this(), "expected ':' after case condition", "parseBoolSwitch")
			return ast.ZeroNode
		}

		p.advance("parseBoolSwitch case :") // consume :

		for !p.match(tokens.Case, tokens.Default, tokens.RBrace, tokens.EOF) {
			if ctx.Err() != nil {
				return ast.ZeroNode
			}

			prev := p.i

			stmt := p.parseStatement(ctx)
			if stmt != ast.ZeroNode {
				caseNode.Body = append(caseNode.Body, stmt)
			} else {
				p.synchronize()
			}

			if p.i == prev {
				p.advance("parseBoolSwitch case recovery")
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
			p.error(p.this(), "expected ':' after default", "parseBoolSwitch")
			return ast.ZeroNode
		}

		p.advance("parseBoolSwitch default :") // consume :

		for !p.match(tokens.RBrace, tokens.EOF) {
			if ctx.Err() != nil {
				return ast.ZeroNode
			}

			prev := p.i

			stmt := p.parseStatement(ctx)
			if stmt != ast.ZeroNode {
				defaultNode.Body = append(defaultNode.Body, stmt)
			} else {
				p.synchronize()
			}

			if p.i == prev {
				p.advance("parseBoolSwitch default recovery")
			}
		}

		node.Default = defaultNode
	}

	p.advance("parseBoolSwitch }") // consume }

	if labelIdent != nil {
		// Set label if present.
		labelIdent.ValueType = types.None
		node.Label = &ast.Label{
			Token: labelIdent.Token,
			Label: labelIdent,
		}
	}

	return ast.NewNode(ast.KindSwitch, node)
}

func (p *Parser) parseIdentSwitch(ctx context.Context, labelIdent *ast.Identifier) ast.NodeValue {
	node := &ast.Switch{
		Token: p.prev(),
	}

	symbol, ok := p.symbols.Resolve(p.this().Literal)
	if !ok {
		p.error(p.this(), "unknown identifier in switch expression", "parseIdentSwitch")
		return ast.ZeroNode
	}

	node.Identifier = symbol.Identifier

	p.advance("parseIdentSwitch") // consume identifier

	if p.this().Type != tokens.LBrace {
		p.error(p.this(), "expected '{' after switch expression", "parseIdentSwitch")
		return ast.ZeroNode
	}

	p.advance("parseIdentSwitch {") // consume {

	for p.this().Type == tokens.Case {
		caseNode := &ast.Case{
			Token: p.this(),
		}

		p.advance("parseIdentSwitch case") // consume case

		cond := p.expression(ctx, symbol.Type())
		if cond == ast.ZeroExpr {
			p.error(p.this(), "unable to parse case expression", "parseIdentSwitch")
			return ast.ZeroNode
		}

		if cond.Type() != symbol.Type() {
			p.error(p.this(), "case condition type does not match switch expression type", "parseIdentSwitch")
			return ast.ZeroNode
		}

		caseNode.Condition = cond

		if p.this().Type != tokens.Colon {
			p.error(p.this(), "expected ':' after case condition", "parseIdentSwitch")
			return ast.ZeroNode
		}

		p.advance("parseIdentSwitch case :") // consume :

		for !p.match(tokens.Case, tokens.Default, tokens.RBrace, tokens.EOF) {
			if ctx.Err() != nil {
				return ast.ZeroNode
			}

			prev := p.i

			stmt := p.parseStatement(ctx)
			if stmt != ast.ZeroNode {
				caseNode.Body = append(caseNode.Body, stmt)
			} else {
				p.synchronize()
			}

			if p.i == prev {
				p.advance("parseIdentSwitch case recovery")
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
			p.error(p.this(), "expected ':' after default", "parseIdentSwitch")
			return ast.ZeroNode
		}

		p.advance("parseIdentSwitch default :") // consume :

		for !p.match(tokens.RBrace, tokens.EOF) {
			if ctx.Err() != nil {
				return ast.ZeroNode
			}

			prev := p.i

			stmt := p.parseStatement(ctx)
			if stmt != ast.ZeroNode {
				defaultNode.Body = append(defaultNode.Body, stmt)
			} else {
				p.synchronize()
			}

			if p.i == prev {
				p.advance("parseIdentSwitch default recovery")
			}
		}

		node.Default = defaultNode
	}

	p.advance("parseIdentSwitch }") // consume }

	if labelIdent != nil {
		// Set label if present.
		labelIdent.ValueType = types.None
		node.Label = &ast.Label{
			Token: labelIdent.Token,
			Label: labelIdent,
		}
	}

	return ast.NewNode(ast.KindSwitch, node)
}
