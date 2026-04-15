package parser

import (
	"context"
	"fmt"

	"github.com/samborkent/cog/internal/ast"
	"github.com/samborkent/cog/internal/tokens"
	"github.com/samborkent/cog/internal/types"
)

func (p *Parser) parseMatch(ctx context.Context) *ast.Match {
	node := &ast.Match{
		Token: p.this(),
	}

	p.advance("parseMatch match") // consume match

	if p.this().Type == tokens.Identifier && p.next().Type == tokens.Declaration {
		node.Binding = &ast.Identifier{
			Token:     p.this(),
			Name:      p.this().Literal,
			Exported:  false,
			Qualifier: ast.QualifierImmutable,
			Global:    p.symbols.Outer == nil,
		}
		p.advance("parseMatch binding ident") // consume ident
		p.advance("parseMatch binding :=")    // consume :=
	}

	node.Subject = p.expression(ctx, types.None)
	if node.Subject == nil {
		p.error(p.this(), "unable to parse match subject", "parseMatch")
		return nil
	}

	subjectType := node.Subject.Type()

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
		return nil
	}

	if p.this().Type != tokens.LBrace {
		p.error(p.this(), "expected '{' after match subject", "parseMatch")
		return nil
	}

	p.advance("parseMatch {") // consume {

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
			return nil
		}

		caseNode.MatchType = caseType

		if p.this().Type != tokens.Colon {
			p.error(p.this(), "expected ':' after case type", "parseMatch")
			return nil
		}

		p.advance("parseMatch case :") // consume :

		p.symbols = NewEnclosedSymbolTable(p.symbols)

		if node.Binding != nil {
			node.Binding.ValueType = caseType
			p.symbols.Define(node.Binding)
		}

		for !p.match(tokens.Case, tokens.Default, tokens.RBrace, tokens.EOF) {
			if ctx.Err() != nil {
				return nil
			}

			prev := p.i

			stmt := p.parseStatement(ctx)
			if stmt != nil {
				caseNode.Body = append(caseNode.Body, stmt)
			} else {
				p.synchronize()
			}

			if p.i == prev {
				p.advance("parseMatch case recovery")
			}
		}

		p.symbols = p.symbols.Outer

		node.Cases = append(node.Cases, caseNode)
	}

	if p.this().Type == tokens.Default {
		defaultNode := &ast.Default{
			Token: p.this(),
		}

		p.advance("parseMatch default") // consume default

		if p.this().Type != tokens.Colon {
			p.error(p.this(), "expected ':' after default", "parseMatch")
			return nil
		}

		p.advance("parseMatch default :") // consume :

		p.symbols = NewEnclosedSymbolTable(p.symbols)

		if node.Binding != nil {
			// In default case, binding variable takes the original subject type
			node.Binding.ValueType = subjectType
			p.symbols.Define(node.Binding)
		}

		for !p.match(tokens.RBrace, tokens.EOF) {
			if ctx.Err() != nil {
				return nil
			}

			prev := p.i

			stmt := p.parseStatement(ctx)
			if stmt != nil {
				defaultNode.Body = append(defaultNode.Body, stmt)
			} else {
				p.synchronize()
			}

			if p.i == prev {
				p.advance("parseMatch default recovery")
			}
		}

		p.symbols = p.symbols.Outer

		node.Default = defaultNode
	}

	if p.this().Type != tokens.RBrace {
		p.error(p.this(), "expected '}' to close match statement", "parseMatch")
		return nil
	}

	p.advance("parseMatch }") // consume }

	return node
}
