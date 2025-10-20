package transpiler

import (
	"fmt"
	goast "go/ast"
	gotoken "go/token"

	"github.com/samborkent/cog/internal/ast"
	"github.com/samborkent/cog/internal/types"
)

func (t *Transpiler) convertStmt(node ast.Statement) ([]goast.Stmt, error) {
	switch n := node.(type) {
	case *ast.Assignment:
		ident := &goast.Ident{Name: "_"}

		if n.Identifier.Name != "_" {
			id, ok := identifiers[n.Identifier.Name]
			if !ok {
				return nil, fmt.Errorf("undefined variable '%s'", n.Identifier.Name)
			}

			ident = id
		}

		expr, err := t.convertExpr(n.Expression)
		if err != nil {
			return nil, err
		}

		if n.Identifier.ValueType.Kind() == types.OptionKind {
			// Warp option type.
			expr = &goast.CompositeLit{
				Type: t.convertType(n.Identifier.ValueType),
				Elts: []goast.Expr{
					&goast.KeyValueExpr{Key: &goast.Ident{Name: "Value"}, Value: expr},
					&goast.KeyValueExpr{Key: &goast.Ident{Name: "Set"}, Value: &goast.Ident{Name: "true"}},
				},
			}
		}

		return []goast.Stmt{&goast.AssignStmt{
			Lhs: []goast.Expr{ident},
			Tok: gotoken.ASSIGN,
			Rhs: []goast.Expr{expr},
		}}, nil
	case *ast.Break:
		return []goast.Stmt{&goast.BranchStmt{
			Tok:   gotoken.BREAK,
			Label: n.Label.Go(),
		}}, nil
	case *ast.Declaration:
		// Define as unused variable.
		identifiers[n.Assignment.Identifier.Name] = &goast.Ident{Name: "_"}

		if n.Assignment.Expression == nil {
			return []goast.Stmt{
				&goast.DeclStmt{
					Decl: &goast.GenDecl{
						Tok: gotoken.VAR,
						Specs: []goast.Spec{
							&goast.ValueSpec{
								Names: []*goast.Ident{identifiers[n.Assignment.Identifier.Name]},
								Type:  t.convertType(n.Type),
							},
						},
					},
				},
			}, nil
		}

		expr, err := t.convertExpr(n.Assignment.Expression)
		if err != nil {
			return nil, err
		}

		// Optional type declaration.
		var declType goast.Expr

		if n.Type != nil && n.Type != types.None {
			declType = t.convertType(n.Type)
		}

		if n.Type.Kind() == types.OptionKind {
			// Warp option type.
			expr = &goast.CompositeLit{
				Type: declType,
				Elts: []goast.Expr{
					&goast.KeyValueExpr{Key: &goast.Ident{Name: "Value"}, Value: expr},
					&goast.KeyValueExpr{Key: &goast.Ident{Name: "Set"}, Value: &goast.Ident{Name: "true"}},
				},
			}
		}

		// Replace type string with type name if missing (for structs, tuples, unions).
		compositeLiteral, ok := expr.(*goast.CompositeLit)
		if ok && compositeLiteral.Type == nil {
			compositeLiteral.Type = &goast.Ident{Name: convertExport(n.Assignment.Identifier.Type().String(), n.Assignment.Identifier.Exported)}
		}

		return []goast.Stmt{&goast.DeclStmt{
			Decl: &goast.GenDecl{
				Tok: gotoken.VAR,
				Specs: []goast.Spec{
					&goast.ValueSpec{
						Names:  []*goast.Ident{identifiers[n.Assignment.Identifier.Name]},
						Type:   declType,
						Values: []goast.Expr{expr},
					},
				},
			},
		}}, nil
	case *ast.ExpressionStatement:
		expr, err := t.convertExpr(n.Expression)
		if err != nil {
			return nil, err
		}

		return []goast.Stmt{&goast.ExprStmt{
			X: expr,
		}}, nil
	case *ast.IfStatement:
		cond, err := t.convertExpr(n.Condition)
		if err != nil {
			return nil, err
		}

		consequence, ifLabel, err := t.convertIfBlock(n.Consequence)
		if err != nil {
			return nil, err
		}

		var (
			alternative goast.Stmt
			elseLabel   *goast.LabeledStmt
		)

		if n.Alternative != nil {
			alternative, elseLabel, err = t.convertIfBlock(n.Alternative)
			if err != nil {
				return nil, err
			}
		}

		stmts := []goast.Stmt{&goast.IfStmt{
			Cond: cond,
			Body: consequence,
			Else: alternative,
		}}

		if ifLabel != nil {
			stmts = append(stmts, ifLabel)
		} else if elseLabel != nil {
			stmts = append(stmts, elseLabel)
		}

		if n.Label != nil {
			stmts = append(stmts, &goast.LabeledStmt{
				Label: &goast.Ident{Name: n.Label.Label.Name},
				Stmt:  noOp,
			})
		}

		return stmts, nil
	case *ast.Return:
		if len(n.Values) == 0 {
			return []goast.Stmt{&goast.ReturnStmt{}}, nil
		}

		exprs := make([]goast.Expr, 0, len(n.Values))

		for _, val := range n.Values {
			expr, err := t.convertExpr(val)
			if err != nil {
				return nil, fmt.Errorf("converting return expression: %w", err)
			}

			exprs = append(exprs, expr)
		}

		return []goast.Stmt{&goast.ReturnStmt{
			Results: exprs,
		}}, nil
	case *ast.Switch:
		cases := make([]goast.Stmt, 0, len(n.Cases))

		for _, c := range n.Cases {
			expr, err := t.convertExpr(c.Condition)
			if err != nil {
				return nil, fmt.Errorf("converting case expression: %w", err)
			}

			stmts := make([]goast.Stmt, 0, len(c.Body))

			for _, stmt := range c.Body {
				caseStmt, err := t.convertStmt(stmt)
				if err != nil {
					return nil, fmt.Errorf("converting case statement: %w", err)
				}

				stmts = append(stmts, caseStmt...)
			}

			cases = append(cases, &goast.CaseClause{
				List: []goast.Expr{expr},
				Body: stmts,
			})
		}

		if n.Default != nil {
			stmts := make([]goast.Stmt, 0, len(n.Default.Body))

			for _, stmt := range n.Default.Body {
				defaultStmt, err := t.convertStmt(stmt)
				if err != nil {
					return nil, fmt.Errorf("converting default statement: %w", err)
				}

				stmts = append(stmts, defaultStmt...)
			}

			cases = append(cases, &goast.CaseClause{
				Body: stmts,
			})
		}

		switchStmt := &goast.SwitchStmt{
			Body: &goast.BlockStmt{
				List: cases,
			},
		}

		if n.Identifier != nil {
			if _, ok := identifiers[n.Identifier.Name]; ok {
				// Mark identifier as used.
				identifiers[n.Identifier.Name].Name = n.Identifier.Name
			}

			switchStmt.Tag = n.Identifier.Go()
		}

		if n.Label != nil {
			return []goast.Stmt{&goast.LabeledStmt{
				Label: n.Label.Label.Go(),
				Stmt:  switchStmt,
			}}, nil
		}

		return []goast.Stmt{switchStmt}, nil
	default:
		return nil, fmt.Errorf("unknown statement type '%T'", n)
	}
}
