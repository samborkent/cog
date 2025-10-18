package parser

import (
	"context"

	"github.com/samborkent/cog/internal/ast"
	"github.com/samborkent/cog/internal/tokens"
	"github.com/samborkent/cog/internal/types"
)

func (p *Parser) parseProcedure(ctx context.Context, ident *ast.Identifier, exported bool) *ast.Procedure {
	if p.this().Type != tokens.Procedure && p.this().Type != tokens.Function {
		p.error(p.this(), "expected procedure or function keyword", "parseProcedure")
		return nil
	}

	node := &ast.Procedure{
		Token:      p.this(),
		Identifier: ident,
		Exported:   exported,
	}

	p.advance("parseProcedure proc / func") // consume proc or func token

	if p.next().Type == tokens.Identifier {
		// Enter parameter scope.
		p.symbols = NewEnclosedSymbolTable(p.symbols)

		inputParams := p.parseParameters(ctx)
		if inputParams == nil {
			p.error(p.this(), "unable to parse input parameters for procedure", "parseProcedure")
			return nil
		}

		node.InputParameters = inputParams
	} else {
		p.advance("parseProcedure (") // consume (
		p.advance("parseProcedure )") // consume )
	}

	// TODO: implement parsing multiple return parameters
	returnType, ok := types.Lookup[p.this().Type]
	if ok {
		// Parse return parameter
		node.ReturnParameters = []*ast.Parameter{
			{
				Identifier: &ast.Identifier{
					Token:     p.this(),
					ValueType: returnType,
				},
			},
		}

		p.advance("parseProcedure return type") // consume return type
	}

	if p.this().Type != tokens.Assign {
		p.error(p.this(), "missing assignment token '=' in procedure declaration", "parseProcedure")
		return nil
	}

	p.advance("parseProcedure =") // consume '='

	if p.this().Type != tokens.LBrace {
		p.error(p.this(), "missing body open token '{'", "parseProcedure")
		return nil
	}

	block := p.parseBlock(ctx)
	if block != nil {
		node.Body = block

		if block.End.Type != tokens.RBrace {
			p.error(block.Start, "missing body close token '}' in procedure declaration", "parseProcedure")
		}
	}

	if len(node.InputParameters) > 0 {
		// Restore parameter scope.
		p.symbols = p.symbols.Outer
	}

	return node
}
