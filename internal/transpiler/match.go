package transpiler

import (
	"fmt"
	goast "go/ast"

	"github.com/samborkent/cog/internal/ast"
	"github.com/samborkent/cog/internal/transpiler/component"
	"github.com/samborkent/cog/internal/types"
)

func (t *Transpiler) convertMatch(n *ast.Match) ([]goast.Stmt, error) {
	subjectExpr := t.Expr(n.Subject)

	expr, err := t.convertExpr(subjectExpr)
	if err != nil {
		return nil, fmt.Errorf("converting match subject: %w", err)
	}

	subjectType := subjectExpr.Type()

	if subjectType.Kind() == types.EitherKind {
		eitherType := subjectType.(*types.Either)

		if len(n.Cases) == 0 {
			return nil, nil // Or return empty block.
		}

		var leftCase, rightCase *ast.MatchCase

		for _, c := range n.Cases {
			if types.Equal(c.MatchType, eitherType.Left) {
				leftCase = c
			} else if types.Equal(c.MatchType, eitherType.Right) {
				rightCase = c
			}
		}

		var leftBody, rightBody []goast.Stmt

		if leftCase != nil {
			if len(leftCase.Body) > 0 {
				t.symbols = NewEnclosedSymbolTable(t.symbols)
				if n.Binding != nil {
					ident := t.symbols.Define(n.Binding.Name)
					leftBody = append(leftBody, component.AssignDef(ident, component.Selector(expr, "Left")))
				}

				for _, stmt := range leftCase.Body {
					convStmt, err := t.convertStmt(t.Node(stmt))
					if err != nil {
						return nil, fmt.Errorf("converting match left case statement: %w", err)
					}

					leftBody = append(leftBody, convStmt...)
				}

				t.symbols = t.symbols.Outer
			}
		}

		if rightCase != nil {
			if len(rightCase.Body) > 0 {
				t.symbols = NewEnclosedSymbolTable(t.symbols)
				if n.Binding != nil {
					ident := t.symbols.Define(n.Binding.Name)
					rightBody = append(rightBody, component.AssignDef(ident, component.Selector(expr, "Right")))
				}

				for _, stmt := range rightCase.Body {
					convStmt, err := t.convertStmt(t.Node(stmt))
					if err != nil {
						return nil, fmt.Errorf("converting match right case statement: %w", err)
					}

					rightBody = append(rightBody, convStmt...)
				}

				t.symbols = t.symbols.Outer
			}
		}

		var elseStmt goast.Stmt
		if len(rightBody) > 0 {
			elseStmt = component.BlockStmt(rightBody...)
		}

		ifStmt := component.IfStmt(
			component.Not(component.Selector(expr, "IsRight")),
			leftBody,
			elseStmt,
		)

		return []goast.Stmt{ifStmt}, nil
	}

	var (
		assignStmt goast.Stmt
		typeSwitch goast.Stmt
	)

	tagExpr := component.Call(component.IdentName("any"), expr)

	if n.Binding != nil {
		ident := t.symbols.Define(n.Binding.Name)
		assignStmt = component.AssignDef(ident, component.TypeAssert(tagExpr, nil))
	} else {
		assignStmt = &goast.ExprStmt{X: component.TypeAssert(tagExpr, nil)}
	}

	switchStmt := &goast.TypeSwitchStmt{
		Assign: assignStmt,
		Body:   &goast.BlockStmt{},
	}

	cases := make([]goast.Stmt, 0, len(n.Cases))

	for _, c := range n.Cases {
		caseType, err := t.convertType(c.MatchType)
		if err != nil {
			return nil, fmt.Errorf("converting match case type: %w", err)
		}

		stmts := make([]goast.Stmt, 0, len(c.Body))

		t.symbols = NewEnclosedSymbolTable(t.symbols)

		for _, stmt := range c.Body {
			convStmt, err := t.convertStmt(t.Node(stmt))
			if err != nil {
				return nil, fmt.Errorf("converting match case statement: %w", err)
			}

			stmts = append(stmts, convStmt...)
		}

		t.symbols = t.symbols.Outer

		cases = append(cases, &goast.CaseClause{
			List: []goast.Expr{caseType},
			Body: stmts,
		})
	}

	if n.Default != nil {
		stmts := make([]goast.Stmt, 0, len(n.Default.Body))

		t.symbols = NewEnclosedSymbolTable(t.symbols)

		for _, stmt := range n.Default.Body {
			convStmt, err := t.convertStmt(t.Node(stmt))
			if err != nil {
				return nil, fmt.Errorf("converting match default statement: %w", err)
			}

			stmts = append(stmts, convStmt...)
		}

		t.symbols = t.symbols.Outer

		cases = append(cases, &goast.CaseClause{
			List: nil,
			Body: stmts,
		})
	}

	switchStmt.Body.List = cases
	typeSwitch = switchStmt

	return []goast.Stmt{typeSwitch}, nil
}
