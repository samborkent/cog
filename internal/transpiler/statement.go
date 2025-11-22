package transpiler

import (
	"fmt"
	goast "go/ast"
	gotoken "go/token"

	"github.com/samborkent/cog/internal/ast"
	"github.com/samborkent/cog/internal/transpiler/comp"
	"github.com/samborkent/cog/internal/types"
)

func (t *Transpiler) convertStmt(node ast.Statement) ([]goast.Stmt, error) {
	switch n := node.(type) {
	case *ast.Assignment:
		ident := &goast.Ident{Name: "_"}

		if n.Identifier.Name != "_" {
			name := convertExport(n.Identifier.Name, n.Identifier.Exported)

			id, ok := t.symbols.Resolve(name)
			if !ok {
				_, ok := t.symbols.ResolveDynamic(name)
				if !ok {
					return nil, fmt.Errorf("undefined dynamic variable '%s'", n.Identifier.Name)
				}

				// Dynamic variable assignment, set context value instead.
				val, err := t.convertExpr(n.Expression)
				if err != nil {
					return nil, err
				}

				return []goast.Stmt{
					comp.ContextWithValue(&goast.Ident{Name: joinStr(name, "Key")}, val),
				}, nil
			}

			ident = id
		}

		expr, err := t.convertExpr(n.Expression)
		if err != nil {
			return nil, err
		}

		if n.Identifier.ValueType.Kind() == types.OptionKind {
			// Wrap option type.
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
		stm := &goast.BranchStmt{
			Tok:   gotoken.BREAK,
			Label: n.Label.Go(),
		}

		return prependLineMarker(n, []goast.Stmt{stm}), nil
	case *ast.Declaration:
		// Define as unused variable.
		ident := t.symbols.Define(n.Assignment.Identifier.Name)
		typ := n.Assignment.Identifier.ValueType

		if n.Assignment.Expression == nil {
			return []goast.Stmt{
				&goast.DeclStmt{
					Decl: &goast.GenDecl{
						Tok: gotoken.VAR,
						Specs: []goast.Spec{
							&goast.ValueSpec{
								Names: []*goast.Ident{ident},
								Type:  t.convertType(typ),
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

		if typ != nil && typ != types.None {
			declType = t.convertType(typ)
		}

		if typ.Kind() == types.OptionKind {
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

		// Declarations are handled as Decls in the top-level transpilation
		// and already have their own comment/doc groups where needed. Do
		// not prepend inline line markers here.
		return []goast.Stmt{&goast.DeclStmt{
			Decl: &goast.GenDecl{
				Tok: gotoken.VAR,
				Specs: []goast.Spec{
					&goast.ValueSpec{
						Names:  []*goast.Ident{ident},
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

		stm := &goast.ExprStmt{X: expr}

		return prependLineMarker(n, []goast.Stmt{stm}), nil
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

		return prependLineMarker(n, stmts), nil
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

		stm := &goast.ReturnStmt{Results: exprs}

		return prependLineMarker(n, []goast.Stmt{stm}), nil
	case *ast.Switch:
		cases := make([]goast.Stmt, 0, len(n.Cases))

		for _, c := range n.Cases {
			expr, err := t.convertExpr(c.Condition)
			if err != nil {
				return nil, fmt.Errorf("converting case expression: %w", err)
			}

			stmts := make([]goast.Stmt, 0, len(c.Body))

			if len(c.Body) > 0 {
				// Enter case block scope.
				t.symbols = NewEnclosedSymbolTable(t.symbols)
			}

			for _, stmt := range c.Body {
				caseStmt, err := t.convertStmt(stmt)
				if err != nil {
					return nil, fmt.Errorf("converting case statement: %w", err)
				}

				stmts = append(stmts, caseStmt...)
			}

			if len(c.Body) > 0 {
				// Leave case block scope.
				t.symbols = t.symbols.Outer
			}

			cases = append(cases, &goast.CaseClause{
				List: []goast.Expr{expr},
				Body: stmts,
			})
		}

		if n.Default != nil {
			stmts := make([]goast.Stmt, 0, len(n.Default.Body))

			if len(n.Default.Body) > 0 {
				// Enter default block scope.
				t.symbols = NewEnclosedSymbolTable(t.symbols)
			}

			for _, stmt := range n.Default.Body {
				defaultStmt, err := t.convertStmt(stmt)
				if err != nil {
					return nil, fmt.Errorf("converting default statement: %w", err)
				}

				stmts = append(stmts, defaultStmt...)
			}

			if len(n.Default.Body) > 0 {
				// Leave default block scope.
				t.symbols = t.symbols.Outer
			}

			cases = append(cases, &goast.CaseClause{
				Body: stmts,
			})
		}

		switchStmt := &goast.SwitchStmt{
			Body: &goast.BlockStmt{List: cases},
		}

		if n.Identifier != nil {
			ident, ok := t.symbols.Resolve(n.Identifier.Name)
			if !ok {
				return nil, fmt.Errorf("unknown identifier %q", n.Identifier.Name)
			}

			t.symbols.MarkUsed(n.Identifier.Name)

			switchStmt.Tag = ident
		}

		if n.Label != nil {
			return prependLineMarker(n, []goast.Stmt{&goast.LabeledStmt{
				Label: n.Label.Label.Go(),
				Stmt:  switchStmt,
			}}), nil
		}

		return prependLineMarker(n, []goast.Stmt{switchStmt}), nil
	default:
		return nil, fmt.Errorf("unknown statement type '%T'", n)
	}
}

// prependLineMarker returns a new slice of statements with a marker assignment
// prepended. The marker is later replaced with a proper //line directive in
// the emitter (cmd/main.go). We use the node hash so we can correlate the
// marker back to the original AST node position.
func prependLineMarker(n ast.Node, stmts []goast.Stmt) []goast.Stmt {
	marker := &goast.AssignStmt{
		Lhs: []goast.Expr{&goast.Ident{Name: "_"}},
		Tok: gotoken.ASSIGN,
		Rhs: []goast.Expr{&goast.BasicLit{
			Kind:  gotoken.STRING,
			Value: fmt.Sprintf("\"__COG_LINE_%d__\"", n.Hash()),
		}},
	}

	out := make([]goast.Stmt, 0, 1+len(stmts))
	out = append(out, marker)
	out = append(out, stmts...)

	return out
}
