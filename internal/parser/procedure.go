package parser

import (
	"context"

	"github.com/samborkent/cog/internal/ast"
	"github.com/samborkent/cog/internal/tokens"
)

func (p *Parser) parseProcedure(ctx context.Context, ident *ast.Identifier, skipBody bool) *ast.Procedure {
	if p.this().Type != tokens.Procedure && p.this().Type != tokens.Function {
		p.error(p.this(), "expected procedure or function keyword", "parseProcedure")
		return nil
	}

	node := &ast.Procedure{
		Token:      p.this(),
		Identifier: ident,
	}

	p.advance("parseProcedure proc / func") // consume proc or func token

	if p.this().Type != tokens.LParen {
		p.error(p.this(), "expected '(' after procedure identifier", "parseParameters")
		return nil
	}

	p.advance("parseParameters (") // consume '('

	if p.this().Type == tokens.Identifier {
		// Enter parameter scope.
		p.symbols = NewEnclosedSymbolTable(p.symbols)

		inputParams := p.parseParameters(ctx, node.Token.Type == tokens.Procedure)
		if inputParams == nil {
			p.error(p.this(), "unable to parse input parameters for procedure", "parseProcedure")
			return nil
		}

		node.InputParameters = inputParams
	}

	if p.this().Type != tokens.RParen {
		p.error(p.this(), "missing parameter close token ')' in procedure declaration", "parseProcedure")
		return nil
	}

	p.advance("parseProcedure )") // consume )

	if p.this().Type != tokens.Assign {
		// There is a return parameter
		returnType := p.parseCombinedType(ctx, ident.Exported)
		if returnType == nil {
			p.error(p.this(), "unable to parse return type", "parseProcedure")
			return nil
		}

		node.ReturnType = returnType
	} else if node.Token.Type == tokens.Function {
		p.error(p.this(), "functions must have a return parameter", "parseProcedure")
		return nil
	}

	if p.this().Type != tokens.Assign {
		// Empty function declaration.
		return node
	}

	p.advance("parseProcedure =") // consume '='

	if p.this().Type != tokens.LBrace {
		p.error(p.this(), "missing body open token '{'", "parseProcedure")
		return nil
	}

	if skipBody {
		// Only used for globals pass.
		p.skipScope(ctx)
	} else {
		block := p.parseBlock(ctx)
		if block != nil {
			node.Body = block

			if block.End.Type != tokens.RBrace {
				p.error(block.Start, "missing body close token '}' in procedure declaration", "parseProcedure")
			}
		}
	}

	if len(node.InputParameters) > 0 {
		// Restore parameter scope.
		p.symbols = p.symbols.Outer
	}

	return node
}
