package parser

import (
	"context"
	"strings"

	"github.com/samborkent/cog/internal/ast"
	"github.com/samborkent/cog/internal/tokens"
	"github.com/samborkent/cog/internal/types"
)

// TODO: take label ident and fill in inside this method
func (p *Parser) parseForStatement(ctx context.Context, labelIdent *ast.Identifier) ast.NodeValue {
	node := &ast.ForStatement{
		Token: p.this(),
	}

	p.advance("parseForStatement for") // consume for

	var (
		valueVar *ast.Identifier
		indexVar *ast.Identifier
	)

	// TODO: add support for value and index variables

	switch p.this().Type {
	case tokens.LBrace:
		// Infinite loop, no range.
	default:
		switch p.next().Type {
		case tokens.In:
			if p.this().Type != tokens.Identifier {
				p.error(p.this(), "expected identifier for loop variable", "parseForStatement")
				return ast.ZeroNode
			}

			valueVar = &ast.Identifier{
				Token:     p.this(),
				Name:      p.this().Literal,
				Qualifier: ast.QualifierImmutable,
			}

			p.advance("parseForStatement value") // consume value variable
			p.advance("parseForStatement in")    // consume in keyword
		case tokens.Comma:
			if p.this().Type != tokens.Identifier {
				p.error(p.this(), "expected identifier for loop value variable", "parseForStatement")
				return ast.ZeroNode
			}

			// Skip _ value variable.
			if p.this().Literal != "_" {
				valueVar = &ast.Identifier{
					Token:     p.this(),
					Name:      p.this().Literal,
					Qualifier: ast.QualifierImmutable,
				}
			}

			p.advance("parseForStatement value") // consume value variable
			p.advance("parseForStatement ,")     // consume ,

			if p.this().Type != tokens.Identifier {
				p.error(p.this(), "expected identifier for loop index variable", "parseForStatement")
				return ast.ZeroNode
			}

			indexVar = &ast.Identifier{
				Token:     p.this(),
				Name:      p.this().Literal,
				ValueType: types.Basics[types.Uint64],
				Qualifier: ast.QualifierImmutable,
			}

			p.advance("parseForStatement index") // consume index

			if p.this().Type != tokens.In {
				p.error(p.this(), "expected in keyword after loop index variable", "parseForStatement")
				return ast.ZeroNode
			}

			p.advance("parseForStatement in") // consume in keyword
		}

		expr := p.expression(ctx, types.None)
		if expr == ast.ZeroExpr {
			p.error(p.this(), "expected range expression or loop body", "parseForStatement")
			return ast.ZeroNode
		}

		if !types.IsIterator(expr.Type()) {
			p.error(p.this(), "cannot iterate over type "+expr.Type().String(), "parseForStatement")
			return ast.ZeroNode
		}

		if valueVar != nil {
			valueVar.ValueType = expr.Type()
		}

		node.Range = expr
	}

	if valueVar != nil || indexVar != nil {
		// Add value variable to scope.
		p.symbols = NewEnclosedSymbolTable(p.symbols)

		if valueVar != nil {
			p.symbols.Define(valueVar)
		}

		if indexVar != nil {
			p.symbols.Define(indexVar)
		}
	}

	prevErrorCount := len(p.Errs)

	loop := p.parseBlockStatement(ctx)
	if loop == nil {
		p.error(p.this(), "unable to parse for block", "parseForStatement")
		return ast.ZeroNode
	}

	if valueVar != nil || indexVar != nil {
		// Restore scope.
		p.symbols = p.symbols.Outer

		// Add value to AST node.
		if valueVar != nil {
			node.Value = valueVar
		}

		if indexVar != nil {
			node.Index = indexVar
		}
	}

	// Logic for specific error when a untyped container literal is passed in loop range expression.
	if len(p.Errs) > prevErrorCount {
		for _, err := range p.Errs[prevErrorCount:] {
			if strings.Contains(err.Error(), "unknown token") {
				p.error(p.this(), "untyped container literal not allowed in loop range expression", "parseIfStatement")
				return ast.ZeroNode
			}
		}
	}

	node.Loop = loop

	if labelIdent != nil {
		// Set label if present.
		labelIdent.ValueType = types.None
		node.Label = &ast.Label{
			Token: labelIdent.Token,
			Label: labelIdent,
		}
	}

	return ast.NewNode(ast.KindForStatement, node)
}
