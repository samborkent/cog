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
	case tokens.Constant:
		// Constant declaration.
		if n := p.parseConstant(ctx, false); n != nil {
			return n
		}

		return nil
	case tokens.Export:
		if p.symbols.Outer != nil {
			p.error(p.this(), "export statements are only allowed in the global scope", "parseStatement")
			return nil
		}

		p.advance("parseStatement export") // consume export

		switch p.this().Type {
		case tokens.Constant:
			if n := p.parseConstant(ctx, true); n != nil {
				return n
			}

			return nil
		case tokens.Identifier:
			ident := &ast.Identifier{
				Token:    p.this(),
				Name:     p.this().Literal,
				Exported: true,
			}

			p.advance("parseStatement export ident") // consume identifier

			switch p.this().Type {
			case tokens.Colon:
				p.advance("parseStatement export ident :") // consume :

				// Do not allow exporting variables.
				_, ok := tokens.FuncTypes[p.this().Type]
				if !ok {
					p.error(p.this(), "expected procedure or function type after exported identifier", "parseStatement")
					return nil
				}

				if proc := p.parseProcedure(ctx, ident, true); proc != nil {
					return proc
				}

				return nil
			case tokens.Tilde:
				typeDecl := p.parseTypeAlias(ctx, ident)
				if typeDecl != nil {
					return typeDecl
				}

				return nil
			default:
				p.error(p.this(), "unexpected token following exported identifierr", "parseStatement")
				return nil
			}
		default:
			p.error(p.this(), "unexpected token found after export", "parseStatement")
			return nil
		}
	case tokens.Identifier:
		ident := &ast.Identifier{
			Token:    p.this(),
			Name:     p.this().Literal,
			Exported: false,
		}

		p.advance("parseStatement ident") // consume identifier

		switch p.this().Type {
		case tokens.Assign: // Assignment
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

			_, ok := tokens.FuncTypes[p.this().Type]
			if ok {
				if proc := p.parseProcedure(ctx, ident, false); proc != nil {
					return proc
				}

				return nil
			} else {
				if p.symbols.Outer == nil {
					p.error(ident.Token, "global variable declarations are not allowed", "parseStatement")
				}

				if d := p.parseTypedDeclaration(ctx, ident, false); d != nil {
					return d
				}

				return nil
			}
		case tokens.Declaration: // Untyped declaration
			if p.symbols.Outer == nil {
				p.error(p.this(), "global variable declarations are not allowed", "parseStatement")
			}

			if d := p.parseDeclaration(ctx, ident, false); d != nil {
				return d
			}

			return nil
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
	case tokens.EOF:
		return nil
	default:
		p.error(p.this(), "unknown token", "parseStatement")
		p.advance("parseStatement unknown") // consume unknown token
		return nil
	}
}
