package parser

import (
	"context"
	"fmt"

	"github.com/samborkent/cog/internal/ast"
	"github.com/samborkent/cog/internal/tokens"
	"github.com/samborkent/cog/internal/types"
)

// TODO: base on heuristics.
const matchPreallocationSize = 4

func (p *Parser) parseMatch(ctx context.Context) ast.NodeIndex {
	var label *ast.Identifier

	prev := p.lex.Peek(-1)

	if prev.Type == tokens.Identifier &&
		p.lex.This().Type == tokens.Colon {
		label = &ast.Identifier{
			Token: prev,
		}

		p.lex.Step() // consume colon
	}

	matchToken := p.lex.This()

	p.lex.Step() // consume match

	var binding *ast.Identifier

	this := p.lex.This()

	if this.Type == tokens.Identifier &&
		p.lex.Peek(1).Type == tokens.Declaration {
		binding = &ast.Identifier{
			Token:     this,
			Exported:  false,
			Qualifier: ast.QualifierImmutable,
			Global:    p.symbols.Outer == nil,
		}
		p.lex.Step() // consume ident
		p.lex.Step() // consume :=
	}

	subject := p.expression(ctx, types.None)
	if subject == ast.ZeroExprIndex {
		p.error(p.lex.This(), "unable to parse match subject", "parseMatch")
		return ast.ZeroNodeIndex
	}

	subjectType := p.ast.Expr(subject).Type()

	isEither := subjectType.Kind() == types.EitherKind

	var isGeneric bool

	if !isEither {
		if tp, ok := subjectType.(*types.Alias); ok && tp.IsTypeParam() {
			if tp.Constraint != nil && (tp.Constraint.Kind() == types.UnionKind || tp.Constraint.Kind() == types.AnyKind) {
				isGeneric = true
			}
		}
	}

	if !isEither && !isGeneric {
		p.error(p.lex.This(), fmt.Sprintf("match subject must be an either type or a generic type parameter bounded by a union or any, got %s", subjectType.String()), "parseMatch")
		return ast.ZeroNodeIndex
	}

	if p.lex.This().Type != tokens.LBrace {
		p.error(p.lex.This(), "expected '{' after match subject", "parseMatch")
		return ast.ZeroNodeIndex
	}

	p.lex.Step() // consume {

	cases := make([]*ast.MatchCase, 0, matchPreallocationSize)
	var branchDeltas []branchConsumption

	for p.lex.This().Type == tokens.Case {
		caseNode := &ast.MatchCase{
			Token: p.lex.This(),
		}

		p.lex.Step() // consume case

		if p.lex.This().Type == tokens.Tilde {
			caseNode.Tilde = true

			p.lex.Step() // consume ~
		}

		caseType := p.parseType(ctx)
		if caseType == nil {
			p.error(p.lex.This(), "unable to parse case type", "parseMatch")
			return ast.ZeroNodeIndex
		}

		caseNode.MatchType = caseType

		if p.lex.This().Type != tokens.Colon {
			p.error(p.lex.This(), "expected ':' after case type", "parseMatch")
			return ast.ZeroNodeIndex
		}

		p.lex.Step() // consume :

		// Rule 16: Snapshot the enclosing scope's ownership before entering the arm.
		caseOwn, caseFieldOwn := p.symbols.snapshotOwnership()

		p.symbols = NewEnclosedSymbolTable(p.symbols)

		if binding != nil {
			binding.ValueType = caseType
			p.symbols.Define(binding)
			p.symbols.MarkUsed(binding.Token.Literal)
		}

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

		p.checkUnused()
		p.symbols = p.symbols.Outer

		// Rule 16: Capture delta and restore for next case.
		branchDeltas = append(branchDeltas, p.symbols.diffOwnership(caseOwn, caseFieldOwn))
		p.symbols.restoreOwnership(caseOwn, caseFieldOwn)

		cases = append(cases, caseNode)
	}

	var defaultNode *ast.Default
	var defDelta branchConsumption

	if p.lex.This().Type == tokens.Default {
		defaultNode = &ast.Default{
			Token: p.lex.This(),
		}

		p.lex.Step() // consume default

		if p.lex.This().Type != tokens.Colon {
			p.error(p.lex.This(), "expected ':' after default", "parseMatch")
			return ast.ZeroNodeIndex
		}

		p.lex.Step() // consume :

		// Rule 16: Snapshot the enclosing scope's ownership before default arm.
		caseOwn, caseFieldOwn := p.symbols.snapshotOwnership()

		p.symbols = NewEnclosedSymbolTable(p.symbols)

		if binding != nil {
			// In default case, binding variable takes the original subject type
			binding.ValueType = subjectType
			p.symbols.Define(binding)
			p.symbols.MarkUsed(binding.Token.Literal)
		}

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

		p.checkUnused()
		p.symbols = p.symbols.Outer

		defDelta = p.symbols.diffOwnership(caseOwn, caseFieldOwn)
		p.symbols.restoreOwnership(caseOwn, caseFieldOwn)
	}

	if p.lex.This().Type != tokens.RBrace {
		p.error(p.lex.This(), "expected '}' to close match statement", "parseMatch")
		return ast.ZeroNodeIndex
	}

	p.lex.Step() // consume }

	// Rule 16: Union consumption across match arms.
	if len(branchDeltas) > 0 || (defaultNode != nil) {
		if defaultNode != nil {
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

	return p.ast.NewMatch(matchToken, label, binding, subject, cases, defaultNode)
}
