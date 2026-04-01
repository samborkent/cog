package parser

import (
	"context"

	"github.com/samborkent/cog/internal/ast"
	"github.com/samborkent/cog/internal/tokens"
	"github.com/samborkent/cog/internal/types"
)

func (p *Parser) parseIfStatement(ctx context.Context) *ast.IfStatement {
	node := &ast.IfStatement{
		Token: p.this(),
	}

	p.advance("parseIfStatement if") // consume if

	expr := p.expression(ctx, types.None)
	if expr == nil {
		p.error(p.this(), "unable to parse bool expression in if condition", "parseIfStatement")
		return nil
	}

	if expr.Type().Kind() != types.Bool {
		p.error(p.this(), "expected bool expression in if condition", "parseIfStatement")
		return nil
	}

	node.Condition = expr

	// Detect option/result ? check patterns for must-check analysis.
	checkedVar, negated := extractCheckVar(expr)

	// ? means "is OK?" for both option and result:
	//   if val?   → consequence: value safe, persists after
	//   if !val?  → consequence: error safe (scoped); else: value safe (scoped)
	persistsAfterIf := checkedVar != "" && !negated

	if checkedVar != "" && !negated {
		// Direct check: value safe in consequence.
		p.symbols.MarkChecked(checkedVar, checkValue)
	} else if checkedVar != "" && negated {
		// Negated check: error safe in consequence.
		p.symbols.MarkChecked(checkedVar, checkError)
	}

	consequence := p.parseBlockStatement(ctx)

	if checkedVar != "" && !persistsAfterIf {
		delete(p.symbols.checked, checkedVar)
	}

	if consequence == nil {
		p.error(p.this(), "unable to parse if block", "parseIfStatement")
		return nil
	}

	node.Consequence = consequence

	if p.this().Type == tokens.Else {
		if ctx.Err() != nil {
			return nil
		}

		p.advance("parseIfStatement else") // consume 'else'

		if checkedVar != "" && negated {
			// Negated check else: value safe.
			p.symbols.MarkChecked(checkedVar, checkValue)
		}

		alternative := p.parseBlockStatement(ctx)

		if checkedVar != "" && negated {
			delete(p.symbols.checked, checkedVar)
		}

		if alternative == nil {
			p.error(p.this(), "unable to parse else block", "parseIfStatement")
			return nil
		}

		node.Alternative = alternative
	}

	// Direct checks persist: value access safe for rest of scope.
	if persistsAfterIf {
		p.symbols.MarkChecked(checkedVar, checkValue)
	}

	return node
}

// extractCheckVar extracts the variable name from a ? check expression.
// Only ? is a check operator. Returns the variable name and whether negated.
// Patterns: val? → (name, false), !val? → (name, true)
func extractCheckVar(expr ast.Expression) (string, bool) {
	switch e := expr.(type) {
	case *ast.Suffix:
		if e.Operator.Type == tokens.Question {
			if ident, ok := e.Left.(*ast.Identifier); ok {
				return ident.Name, false
			}
		}
	case *ast.Prefix:
		if e.Operator.Type == tokens.Not {
			name, neg := extractCheckVar(e.Right)
			return name, !neg
		}
	}

	return "", false
}
