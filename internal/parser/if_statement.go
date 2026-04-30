package parser

import (
	"context"

	"github.com/samborkent/cog/internal/ast"
	"github.com/samborkent/cog/internal/tokens"
	"github.com/samborkent/cog/internal/types"
)

func (p *Parser) parseIfStatement(ctx context.Context) ast.NodeIndex {
	var label *ast.Identifier

	if p.prev().Type == tokens.Identifier && p.this().Type == tokens.Colon {
		label = &ast.Identifier{
			Token: p.prev(),
			Name:  p.prev().Literal,
		}

		p.advance("parseIfStatement :") // consume colon
	}

	ifToken := p.this()

	p.advance("parseIfStatement if") // consume if

	condition := p.expression(ctx, types.None)
	if condition == ast.ZeroExprIndex {
		p.error(p.this(), "unable to parse bool expression in if condition", "parseIfStatement")
		return ast.ZeroNodeIndex
	}

	expr := p.ast.Expr(condition)

	if expr.Type().Kind() != types.Bool &&
		expr.Type().Kind() != types.ErrorKind &&
		expr.Type().Kind() != types.OptionKind {
		p.error(p.this(), "expected bool, result, or option expression in if condition", "parseIfStatement")
		return ast.ZeroNodeIndex
	}

	// Detect option/result ? check patterns for must-check analysis.
	checkedVar, negated := p.extractCheckVar(expr)

	// ? means "is OK?" for both option and result:
	//   if val?   → consequence: value safe, persists after
	//   if !val?  → consequence: error safe (scoped); else: value safe (scoped)
	//
	// Early-exit promotion: if the consequence block exits scope (return/break/continue),
	// the opposite check is promoted to persist after the if-statement.
	//   if !val? { return }  → value safe after (error case handled)
	//   if val?  { return }  → error safe after (value case handled)
	persistsAfterIf := checkedVar != "" && !negated

	// Save check state before modifying, so scoped checks can be restored.
	prevState, hadPrevState := p.symbols.checked[checkedVar]

	if checkedVar != "" && !negated {
		// Direct check: value safe in consequence.
		p.symbols.MarkChecked(checkedVar, checkValue)
	} else if checkedVar != "" && negated {
		// Negated check: error safe in consequence.
		p.symbols.MarkChecked(checkedVar, checkError)
	}

	consequence := p.parseBlockStatement(ctx)

	if checkedVar != "" && !persistsAfterIf {
		if hadPrevState {
			p.symbols.checked[checkedVar] = prevState
		} else {
			delete(p.symbols.checked, checkedVar)
		}
	}

	if consequence == nil {
		p.error(p.this(), "unable to parse if block", "parseIfStatement")
		return ast.ZeroNodeIndex
	}

	// Early-exit promotion: if the consequence block exits scope,
	// the opposite check persists after the if-statement.
	if checkedVar != "" && !persistsAfterIf && p.blockExitsScope(consequence) {
		if negated {
			// if !val? { return } → value safe after.
			persistsAfterIf = true
		} else {
			// if val? { return } → error safe after.
			p.symbols.MarkChecked(checkedVar, checkError)
		}
	}

	var alternative *ast.Block

	if p.this().Type == tokens.Else {
		if ctx.Err() != nil {
			return ast.ZeroNodeIndex
		}

		p.advance("parseIfStatement else") // consume 'else'

		if checkedVar != "" && negated {
			// Negated check else: value safe.
			p.symbols.MarkChecked(checkedVar, checkValue)
		}

		alternative = p.parseBlockStatement(ctx)

		if checkedVar != "" && negated {
			if hadPrevState {
				p.symbols.checked[checkedVar] = prevState
			} else {
				delete(p.symbols.checked, checkedVar)
			}
		}

		if alternative == nil {
			p.error(p.this(), "unable to parse else block", "parseIfStatement")
			return ast.ZeroNodeIndex
		}
	}

	// Direct checks persist: value access safe for rest of scope.
	if persistsAfterIf {
		p.symbols.MarkChecked(checkedVar, checkValue)
	}

	return p.ast.NewIfStatement(ifToken, label, condition, consequence, alternative)
}

// blockExitsScope reports whether a block's last statement unconditionally
// exits the enclosing scope (return, break, or continue).
func (p *Parser) blockExitsScope(block *ast.Block) bool {
	if len(block.Statements) == 0 {
		return false
	}

	last := block.Statements[len(block.Statements)-1]

	switch p.ast.Node(last).(type) {
	case *ast.Return, *ast.Branch:
		return true
	}

	return false
}

// extractCheckVar extracts the variable name from a ? check expression.
// Only ? is a check operator. Returns the variable name and whether negated.
// Patterns: val? → (name, false), !val? → (name, true)
func (p *Parser) extractCheckVar(expr ast.Expr) (string, bool) {
	switch e := expr.(type) {
	case *ast.Suffix:
		if e.Operator.Type == tokens.Question {
			if ident, ok := p.ast.Expr(e.Left).(*ast.Identifier); ok {
				return ident.Name, false
			}
		}
	case *ast.Prefix:
		if e.Operator.Type == tokens.Not {
			name, neg := p.extractCheckVar(p.ast.Expr(e.Right))
			return name, !neg
		}
	}

	return "", false
}
