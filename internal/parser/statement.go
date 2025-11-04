package parser

import (
	"context"

	"github.com/samborkent/cog/internal/ast"
	"github.com/samborkent/cog/internal/tokens"
	"github.com/samborkent/cog/internal/types"
)

func (p *Parser) parseStatement(ctx context.Context) ast.Statement {
	switch p.this().Type {
	case tokens.Break:
		node := &ast.Break{
			Token: p.this(),
		}

		p.advance("parseStatement break") // consume break

		if p.this().Type == tokens.Identifier {
			node.Label = &ast.Identifier{
				Token:     p.this(),
				Name:      p.this().Literal,
				ValueType: types.None,
			}

			p.advance("parseStatement break label") // consume label
		}

		return node
	case tokens.Builtin:
		t := p.this()

		p.advance("parseStatement builtin") // consume @

		builtinParser, ok := p.builtins[t.Literal]
		if !ok {
			p.error(t, "unknown builtin function", "parseStatement")
			return nil
		}

		return &ast.ExpressionStatement{
			Token:      t,
			Expression: builtinParser(ctx, t, types.None),
		}
	case tokens.Dynamic:
		// Skip, get it with prev in identifier case.
		p.advance("parseStatement dyn") // consume dyn

		if p.symbols.Outer != nil {
			p.error(p.this(), "dynamic scope variable declarations are only allowed in package scope", "parseStatement")
			return nil
		}

		return p.parseStatement(ctx)
	case tokens.Export:
		if p.symbols.Outer != nil {
			p.error(p.this(), "export statements are only allowed in the global scope", "parseStatement")
			return nil
		}

		p.advance("parseStatement export") // consume export

		switch p.this().Type {
		case tokens.Identifier:
			ident := &ast.Identifier{
				Token:     p.this(),
				Name:      p.this().Literal,
				Exported:  true,
				Qualifier: ast.QualifierImmutable,
			}

			p.advance("parseStatement export ident") // consume identifier

			switch p.this().Type {
			case tokens.Colon:
				p.advance("parseStatement export ident :") // consume :

				decl := p.parseTypedDeclaration(ctx, ident)
				if decl != nil {
					return decl
				}

				return nil
			case tokens.Declaration:
				decl := p.parseDeclaration(ctx, ident)
				if decl != nil {
					return decl
				}

				return nil
			case tokens.Tilde:
				typeDecl := p.parseTypeAlias(ctx, ident)
				if typeDecl != nil {
					return typeDecl
				}

				return nil
			default:
				p.error(p.this(), "unexpected token following exported identifier", "parseStatement")
				p.advance("parseStatement export error") // consume unknown token
				return nil
			}
		default:
			p.error(p.this(), "unexpected token found after export", "parseStatement")
			return nil
		}
	case tokens.Identifier:
		qualifier := ast.QualifierImmutable

		switch p.prev().Type {
		case tokens.Variable:
			qualifier = ast.QualifierVariable
		case tokens.Dynamic:
			qualifier = ast.QualifierDynamic
		}

		ident := &ast.Identifier{
			Token:     p.this(),
			Name:      p.this().Literal,
			Exported:  false,
			Qualifier: qualifier,
		}

		// Do not skip identifier for function call, parse as expression.
		if p.next().Type != tokens.LParen {
			p.advance("parseStatement ident") // consume identifier
		}

		switch p.this().Type {
		case tokens.Assign: // Assignment
			if p.symbols.Outer == nil {
				p.error(p.this(), "no assignment allowed in package scope, use declaration instead", "parseStatement")
				return nil
			}

			if a := p.parseAssignment(ctx, ident); a != nil {
				return a
			}

			return nil
		case tokens.Colon: // Typed declaration or label
			p.advance("parseStatement ident :") // consume ':'

			if p.this().Type == tokens.If {
				// Labeled if statement
				ifStatement := p.parseIfStatement(ctx)
				if ifStatement == nil {
					return nil
				}

				ident.ValueType = types.None
				ifStatement.Label = &ast.Label{
					Token: ident.Token,
					Label: ident,
				}

				return ifStatement
			} else if p.this().Type == tokens.Switch {
				// Labeled switch statement
				switchStatement := p.parseSwitch(ctx)
				if switchStatement == nil {
					return nil
				}

				ident.ValueType = types.None
				switchStatement.Label = &ast.Label{
					Token: ident.Token,
					Label: ident,
				}

				return switchStatement
			}

			if d := p.parseTypedDeclaration(ctx, ident); d != nil {
				return d
			}

			return nil
		case tokens.Declaration: // Untyped declaration
			if d := p.parseDeclaration(ctx, ident); d != nil {
				return d
			}

			return nil
		case tokens.Identifier: // Procedure call
			identToken := p.this()

			callExpr := p.expression(ctx, types.None)
			if callExpr == nil {
				return nil
			}

			return &ast.ExpressionStatement{
				Token:      identToken,
				Expression: callExpr,
			}
		case tokens.Tilde: // Type declaration
			typeDecl := p.parseTypeAlias(ctx, ident)
			if typeDecl != nil {
				return typeDecl
			}

			return nil
		default:
			p.error(p.this(), "unexpected token found after identifier", "parseStatement")
			return nil
		}
	case tokens.If:
		if node := p.parseIfStatement(ctx); node != nil {
			return node
		}

		return nil
	case tokens.Return:
		node := &ast.Return{
			Token: p.this(),
		}

		p.advance("parseStatement return") // consume return

		node.Values = make([]ast.Expression, 0)

		for {
			expr := p.expression(ctx, types.None)
			if expr != nil {
				node.Values = append(node.Values, expr)
			}

			if p.this().Type != tokens.Comma {
				break
			}

			p.advance("parseStatement return ,") // consume comma
		}

		return node
	case tokens.Switch:
		if node := p.parseSwitch(ctx); node != nil {
			return node
		}

		return nil
	case tokens.Variable:
		// Skip, get it with prev in identifier case.
		p.advance("parseStatement var") // consume var

		if p.symbols.Outer == nil {
			p.error(p.this(), "variable declarations are not allowed in package scope", "parseStatement")
			return nil
		}

		return p.parseStatement(ctx)
	case tokens.EOF:
		return nil
	default:
		p.error(p.this(), "unknown token", "parseStatement")
		p.advance("parseStatement unknown") // consume unknown token
		return nil
	}
}
