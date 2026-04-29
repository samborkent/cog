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

	if p.prev().Type == tokens.Identifier && p.this().Type == tokens.Colon {
		label = &ast.Identifier{
			Token: p.prev(),
			Name:  p.prev().Literal,
		}

		p.advance("parseSwitch :") // consume colon
	}

	matchToken := p.this()

	p.advance("parseMatch match") // consume match

	var binding *ast.Identifier

	if p.this().Type == tokens.Identifier && p.next().Type == tokens.Declaration {
		binding = &ast.Identifier{
			Token:     p.this(),
			Name:      p.this().Literal,
			Exported:  false,
			Qualifier: ast.QualifierImmutable,
			Global:    p.symbols.Outer == nil,
		}
		p.advance("parseMatch binding ident") // consume ident
		p.advance("parseMatch binding :=")    // consume :=
	}

	subject := p.expression(ctx, types.None)
	if subject == ast.ZeroExprIndex {
		p.error(p.this(), "unable to parse match subject", "parseMatch")
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
		p.error(p.this(), fmt.Sprintf("match subject must be an either type or a generic type parameter bounded by a union or any, got %s", subjectType.String()), "parseMatch")
		return ast.ZeroNodeIndex
	}

	if p.this().Type != tokens.LBrace {
		p.error(p.this(), "expected '{' after match subject", "parseMatch")
		return ast.ZeroNodeIndex
	}

	p.advance("parseMatch {") // consume {

	cases := make([]*ast.MatchCase, 0)

	for p.this().Type == tokens.Case {
		caseNode := &ast.MatchCase{
			Token: p.this(),
		}

		p.advance("parseMatch case") // consume case

		if p.this().Type == tokens.Tilde {
			caseNode.Tilde = true

			p.advance("parseMatch case ~") // consume ~
		}

		caseType := p.parseType(ctx)
		if caseType == nil {
			p.error(p.this(), "unable to parse case type", "parseMatch")
			return ast.ZeroNodeIndex
		}

		caseNode.MatchType = caseType

		if p.this().Type != tokens.Colon {
			p.error(p.this(), "expected ':' after case type", "parseMatch")
			return ast.ZeroNodeIndex
		}

		p.advance("parseMatch case :") // consume :

		p.symbols = NewEnclosedSymbolTable(p.symbols)

		if binding != nil {
			binding.ValueType = caseType
			p.symbols.Define(binding)
		}

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
				p.advance("parseMatch case recovery")
			}
		}

		p.symbols = p.symbols.Outer

		cases = append(cases, caseNode)
	}

	var defaultNode *ast.Default

	if p.this().Type == tokens.Default {
		defaultNode = &ast.Default{
			Token: p.this(),
		}

		p.advance("parseMatch default") // consume default

		if p.this().Type != tokens.Colon {
			p.error(p.this(), "expected ':' after default", "parseMatch")
			return ast.ZeroNodeIndex
		}

		p.advance("parseMatch default :") // consume :

		p.symbols = NewEnclosedSymbolTable(p.symbols)

		if binding != nil {
			// In default case, binding variable takes the original subject type
			binding.ValueType = subjectType
			p.symbols.Define(binding)
		}

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
				p.advance("parseMatch default recovery")
			}
		}

		p.symbols = p.symbols.Outer
	}

	if p.this().Type != tokens.RBrace {
		p.error(p.this(), "expected '}' to close match statement", "parseMatch")
		return ast.ZeroNodeIndex
	}

	p.advance("parseMatch }") // consume }

	return p.ast.NewMatch(matchToken, label, binding, subject, cases, defaultNode)
}
