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

	if p.lex.Peek(-1).Type == tokens.Identifier && p.lex.This().Type == tokens.Colon {
		label = &ast.Identifier{
			Token: p.lex.Peek(-1),
		}

		p.lex.Step() // consume colon
	}

	p.lex.Step() // consume switch

	switch p.lex.This().Type {
	case tokens.Identifier:
		return p.parseIdentSwitch(ctx, label)
	case tokens.LBrace:
		return p.parseBoolSwitch(ctx, label)
	default:
		p.error(p.lex.This(), "unexpected token after switch", "parseSwitch")
		return ast.ZeroNodeIndex
	}
}

func (p *Parser) parseBoolSwitch(ctx context.Context, label *ast.Identifier) ast.NodeIndex {
	switchToken := p.lex.Peek(-1)

	p.lex.Step() // consume {

	cases := make([]*ast.Case, 0, switchCasePreallocationSize)

	var branchDeltas []branchConsumption

	for p.lex.This().Type != tokens.EOF && ctx.Err() == nil {
		t := p.lex.This()
		if t.Type != tokens.Case {
			break
		}

		caseNode := &ast.Case{
			Token: t,
		}

		p.lex.Step() // consume case

		expr := p.expression(ctx, types.None)
		if expr == ast.ZeroExprIndex {
			p.error(p.lex.This(), "unable to parse case expression", "parseBoolSwitch")
			return ast.ZeroNodeIndex
		}

		caseNode.Condition = expr

		if p.lex.This().Type != tokens.Colon {
			p.error(p.lex.This(), "expected ':' after case condition", "parseBoolSwitch")
			return ast.ZeroNodeIndex
		}

		p.lex.Step() // consume :

		// Rule 16: Snapshot ownership before each case body.
		caseOwn, caseFieldOwn := p.symbols.snapshotOwnership()

		for p.lex.This().Type != tokens.EOF && ctx.Err() == nil {
			if p.match(p.lex.This(), tokens.Case, tokens.Default, tokens.RBrace) {
				break
			}

			stmt := p.parseStatement(ctx)
			if stmt != ast.ZeroNodeIndex {
				caseNode.Body = append(caseNode.Body, stmt)
			} else {
				p.synchronize(ctx)
			}
		}

		// Rule 16: Capture delta and restore for next case.
		branchDeltas = append(branchDeltas, p.symbols.diffOwnership(caseOwn, caseFieldOwn))
		p.symbols.restoreOwnership(caseOwn, caseFieldOwn)

		cases = append(cases, caseNode)
	}

	var def *ast.Default

	if p.lex.This().Type == tokens.Default {
		defaultNode := &ast.Default{
			Token: p.lex.This(),
		}

		p.lex.Step() // consume default

		if p.lex.This().Type != tokens.Colon {
			p.error(p.lex.This(), "expected ':' after default", "parseBoolSwitch")
			return ast.ZeroNodeIndex
		}

		p.lex.Step() // consume :

		for p.lex.This().Type != tokens.EOF && ctx.Err() == nil {
			if p.lex.This().Type == tokens.RBrace {
				break
			}

			stmt := p.parseStatement(ctx)
			if stmt != ast.ZeroNodeIndex {
				defaultNode.Body = append(defaultNode.Body, stmt)
			} else {
				p.synchronize(ctx)
			}
		}

		def = defaultNode
	}

	p.lex.Step() // consume }

	// Rule 16: Union consumption across switch cases.
	if len(branchDeltas) > 0 {
		unionVars, unionFields := unionConsumption(branchDeltas)
		for name := range unionVars {
			p.symbols.MarkConsumed(name)
		}
		for sv, fs := range unionFields {
			for f := range fs {
				p.symbols.MarkFieldConsumed(sv, f)
			}
		}
	}

	return p.ast.NewSwitch(switchToken, label, nil, cases, def)
}

func (p *Parser) parseIdentSwitch(ctx context.Context, label *ast.Identifier) ast.NodeIndex {
	switchToken := p.lex.Peek(-1)

	symbol, ok := p.symbols.Resolve(p.lex.This().Literal)
	if !ok {
		p.error(p.lex.This(), "unknown identifier in switch expression", "parseIdentSwitch")
		return ast.ZeroNodeIndex
	}

	p.symbols.MarkUsed(p.lex.This().Literal)
	p.lex.Step() // consume identifier

	if p.lex.This().Type != tokens.LBrace {
		p.error(p.lex.This(), "expected '{' after switch expression", "parseIdentSwitch")
		return ast.ZeroNodeIndex
	}

	p.lex.Step() // consume {

	cases := make([]*ast.Case, 0, switchCasePreallocationSize)

	var branchDeltas []branchConsumption

	for p.lex.This().Type != tokens.EOF && ctx.Err() == nil {
		t := p.lex.This()
		if t.Type != tokens.Case {
			break
		}

		caseNode := &ast.Case{
			Token: t,
		}

		p.lex.Step() // consume case

		cond := p.expression(ctx, symbol.Type())
		if cond == ast.ZeroExprIndex {
			p.error(p.lex.This(), "unable to parse case expression", "parseIdentSwitch")
			return ast.ZeroNodeIndex
		}

		if p.ast.Expr(cond).Type() != symbol.Type() {
			p.error(p.lex.This(), "case condition type does not match switch expression type", "parseIdentSwitch")
			return ast.ZeroNodeIndex
		}

		caseNode.Condition = cond

		if p.lex.This().Type != tokens.Colon {
			p.error(p.lex.This(), "expected ':' after case condition", "parseIdentSwitch")
			return ast.ZeroNodeIndex
		}

		p.lex.Step() // consume :

		// Rule 16: Snapshot ownership before each case body.
		caseOwn, caseFieldOwn := p.symbols.snapshotOwnership()

		for p.lex.This().Type != tokens.EOF && ctx.Err() == nil {
			if p.match(p.lex.This(), tokens.Case, tokens.Default, tokens.RBrace) {
				break
			}

			stmt := p.parseStatement(ctx)
			if stmt != ast.ZeroNodeIndex {
				caseNode.Body = append(caseNode.Body, stmt)
			} else {
				p.synchronize(ctx)
			}
		}

		// Rule 16: Capture delta and restore for next case.
		branchDeltas = append(branchDeltas, p.symbols.diffOwnership(caseOwn, caseFieldOwn))
		p.symbols.restoreOwnership(caseOwn, caseFieldOwn)

		cases = append(cases, caseNode)
	}

	var def *ast.Default
	var defDelta branchConsumption

	if p.lex.This().Type == tokens.Default {
		defaultNode := &ast.Default{
			Token: p.lex.This(),
		}

		p.lex.Step() // consume default

		if p.lex.This().Type != tokens.Colon {
			p.error(p.lex.This(), "expected ':' after default", "parseIdentSwitch")
			return ast.ZeroNodeIndex
		}

		p.lex.Step() // consume :

		// Rule 16: Snapshot for default case.
		caseOwn, caseFieldOwn := p.symbols.snapshotOwnership()

		for p.lex.This().Type != tokens.EOF && ctx.Err() == nil {
			if p.lex.This().Type == tokens.RBrace {
				break
			}

			stmt := p.parseStatement(ctx)
			if stmt != ast.ZeroNodeIndex {
				defaultNode.Body = append(defaultNode.Body, stmt)
			} else {
				p.synchronize(ctx)
			}
		}

		defDelta = p.symbols.diffOwnership(caseOwn, caseFieldOwn)
		p.symbols.restoreOwnership(caseOwn, caseFieldOwn)

		def = defaultNode
	}

	p.lex.Step() // consume }

	// Rule 16: Union consumption across switch cases.
	if len(branchDeltas) > 0 || (def != nil) {
		if def != nil {
			branchDeltas = append(branchDeltas, defDelta)
		}
		unionVars, unionFields := unionConsumption(branchDeltas)
		for name := range unionVars {
			p.symbols.MarkConsumed(name)
		}
		for sv, fs := range unionFields {
			for f := range fs {
				p.symbols.MarkFieldConsumed(sv, f)
			}
		}
	}

	return p.ast.NewSwitch(switchToken, label, symbol.Identifier, cases, def)
}
