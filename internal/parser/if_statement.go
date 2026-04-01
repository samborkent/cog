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

	// Detect option/result check patterns for must-check analysis.
	checkedVar, negated, isResultCheck := extractCheckVar(expr, p.symbols)

	// Determine where the value is safe to access based on check type and negation.
	//
	// Option ? (true = "is set"):
	//   if val?   → consequence: value safe, persists after
	//   if !val?  → else: value safe (scoped)
	//
	// Result ! (true = "has error"):
	//   if val!   → consequence: value NOT safe (error branch), persists after
	//   if !val!  → consequence: value safe (no error, scoped)
	valueSafeInConsequence := checkedVar != "" && (isResultCheck == negated)
	valueSafeInElse := checkedVar != "" && !valueSafeInConsequence
	persistsAfterIf := checkedVar != "" && !negated

	if valueSafeInConsequence {
		p.symbols.MarkChecked(checkedVar)
	}

	consequence := p.parseBlockStatement(ctx)

	if valueSafeInConsequence && !persistsAfterIf {
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

		if valueSafeInElse {
			p.symbols.MarkChecked(checkedVar)
		}

		alternative := p.parseBlockStatement(ctx)

		if valueSafeInElse {
			delete(p.symbols.checked, checkedVar)
		}

		if alternative == nil {
			p.error(p.this(), "unable to parse else block", "parseIfStatement")
			return nil
		}

		node.Alternative = alternative
	}

	// Direct checks persist: mark after the if block if not already marked.
	if persistsAfterIf && !valueSafeInConsequence {
		p.symbols.MarkChecked(checkedVar)
	}

	return node
}

// extractCheckVar extracts the variable name from an option/result check expression.
// Returns the variable name, whether the check is negated, and whether it's a result check (!).
// Patterns: val? / val! → (name, false, ...), !val? / !val! → (name, true, ...)
func extractCheckVar(expr ast.Expression, symbols *SymbolTable) (string, bool, bool) {
	switch e := expr.(type) {
	case *ast.Suffix:
		if e.Operator.Type == tokens.Question || e.Operator.Type == tokens.Not {
			if ident, ok := e.Left.(*ast.Identifier); ok {
				isResult := false
				if sym, ok := symbols.Resolve(ident.Name); ok {
					isResult = sym.Type().Kind() == types.ResultKind
				}
				return ident.Name, false, isResult
			}
		}
	case *ast.Prefix:
		if e.Operator.Type == tokens.Not {
			name, neg, isResult := extractCheckVar(e.Right, symbols)
			return name, !neg, isResult
		}
	}

	return "", false, false
}
