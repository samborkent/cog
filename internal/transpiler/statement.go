package transpiler

import (
	"fmt"
	goast "go/ast"
	gotoken "go/token"

	"github.com/samborkent/cog/internal/ast"
	"github.com/samborkent/cog/internal/tokens"
	"github.com/samborkent/cog/internal/transpiler/component"
	"github.com/samborkent/cog/internal/types"
)

func (t *Transpiler) convertStmt(node ast.Statement) ([]goast.Stmt, error) {
	var returnStmts []goast.Stmt

	switch n := node.(type) {
	case *ast.Comment:
		return []goast.Stmt{&goast.DeclStmt{Decl: t.commentDecl(n.Text)[0]}}, nil
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

				if t.inFunc {
					return nil, fmt.Errorf("func cannot assign dynamically scoped variable %q", n.Identifier.Name)
				}

				// Dynamic variable assignment via struct field.
				t.usesDyn = true
				val, err := t.convertExpr(n.Expression)
				if err != nil {
					return nil, err
				}

				return []goast.Stmt{
					component.DynWrite(name, val),
				}, nil
			}

			ident = id
		}

		expr, err := t.convertExpr(n.Expression)
		if err != nil {
			return nil, err
		}

		if n.Identifier.ValueType.Kind() == types.OptionKind {
			optionType, err := t.convertType(n.Identifier.ValueType)
			if err != nil {
				return nil, fmt.Errorf("converting option value type: %w", err)
			}

			// Wrap option type.
			expr = &goast.CompositeLit{
				Type: optionType,
				Elts: []goast.Expr{
					&goast.KeyValueExpr{Key: &goast.Ident{Name: "Value"}, Value: expr},
					&goast.KeyValueExpr{Key: &goast.Ident{Name: "Set"}, Value: &goast.Ident{Name: "true"}},
				},
			}
		}

		returnStmts = []goast.Stmt{&goast.AssignStmt{
			Lhs: []goast.Expr{ident},
			Tok: gotoken.ASSIGN,
			Rhs: []goast.Expr{expr},
		}}
	case *ast.Branch:
		var goTok gotoken.Token

		switch n.Token.Type {
		case tokens.Break:
			goTok = gotoken.BREAK
		case tokens.Continue:
			goTok = gotoken.CONTINUE
		default:
			return nil, fmt.Errorf("unknown branch token '%s'", n.Token.Literal)
		}

		returnStmts = []goast.Stmt{&goast.BranchStmt{
			Tok:   goTok,
			Label: component.Ident(n.Label),
		}}
	case *ast.Declaration:
		// Define as unused variable.
		ident := t.symbols.Define(n.Assignment.Identifier.Name)
		typ := n.Assignment.Identifier.ValueType

		ln, _ := node.Pos()
		comment := &goast.Comment{Text: fmt.Sprintf("\n//line %s:%d", t.file.Name, ln)}

		if n.Assignment.Expression == nil {
			declType, err := t.convertType(typ)
			if err != nil {
				return nil, fmt.Errorf("converting type in declaration: %w", err)
			}

			returnStmts = []goast.Stmt{
				&goast.DeclStmt{
					Decl: &goast.GenDecl{
						Doc: &goast.CommentGroup{
							List: []*goast.Comment{comment},
						},
						Tok: gotoken.VAR,
						Specs: []goast.Spec{
							&goast.ValueSpec{
								Names: []*goast.Ident{ident},
								Type:  declType,
							},
						},
					},
				},
			}
			break
		}

		expr, err := t.convertExpr(n.Assignment.Expression)
		if err != nil {
			return nil, err
		}

		// Optional type declaration.
		var declType goast.Expr

		if typ != nil && typ != types.None {
			declType, err = t.convertType(typ)
			if err != nil {
				return nil, fmt.Errorf("converting type in declaration: %w", err)
			}
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

		returnStmts = []goast.Stmt{&goast.DeclStmt{
			Decl: &goast.GenDecl{
				Doc: &goast.CommentGroup{
					List: []*goast.Comment{comment},
				},
				Tok: gotoken.VAR,
				Specs: []goast.Spec{
					&goast.ValueSpec{
						Names:  []*goast.Ident{ident},
						Type:   declType,
						Values: []goast.Expr{expr},
					},
				},
			},
		}}
	case *ast.ExpressionStatement:
		expr, err := t.convertExpr(n.Expression)
		if err != nil {
			return nil, err
		}

		returnStmts = []goast.Stmt{&goast.ExprStmt{
			X: expr,
		}}
	case *ast.ForStatement:
		body, err := t.convertForBlock(n.Loop)
		if err != nil {
			return nil, err
		}

		var stmt goast.Stmt

		if n.Range == nil {
			// C-style for loop.
			stmt = &goast.ForStmt{
				Body: body,
			}
		} else {
			// Range based for loop.
			rangeExpr, err := t.convertExpr(n.Range)
			if err != nil {
				return nil, err
			}

			var key goast.Expr
			var val goast.Expr

			tok := gotoken.ILLEGAL

			if n.Index != nil || n.Value != nil {
				tok = gotoken.DEFINE
			}

			if n.Index != nil && n.Value != nil {
				key = &goast.Ident{Name: n.Index.Name}
				val = &goast.Ident{Name: n.Value.Name}
			} else if n.Index != nil && n.Value == nil {
				key = &goast.Ident{Name: n.Index.Name}
			} else if n.Index == nil && n.Value != nil {
				key = &goast.Ident{Name: "_"}
				val = &goast.Ident{Name: n.Value.Name}
			}

			stmt = &goast.RangeStmt{
				Key:   key,
				Value: val,
				Tok:   tok,
				X:     rangeExpr,
				Body:  body,
			}
		}

		if n.Label != nil {
			returnStmts = []goast.Stmt{&goast.LabeledStmt{
				Label: component.Ident(n.Label.Label),
				Stmt:  stmt,
			}}
		} else {
			returnStmts = []goast.Stmt{stmt}
		}
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

		returnStmts = stmts
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

		returnStmts = []goast.Stmt{&goast.ReturnStmt{
			Results: exprs,
		}}
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
			Body: &goast.BlockStmt{
				List: cases,
			},
		}

		if n.Identifier != nil {
			ident, ok := t.symbols.Resolve(n.Identifier.Name)
			if !ok {
				return nil, fmt.Errorf("unknown identifier %q", n.Identifier.Name)
			}

			if err := t.symbols.MarkUsed(n.Identifier.Name); err != nil {
				return nil, fmt.Errorf("marking switch identifier used: %w", err)
			}

			switchStmt.Tag = ident
		}

		if n.Label != nil {
			returnStmts = []goast.Stmt{&goast.LabeledStmt{
				Label: component.Ident(n.Label.Label),
				Stmt:  switchStmt,
			}}
		} else {
			returnStmts = []goast.Stmt{switchStmt}
		}
	case *ast.Type:
		typ, err := t.convertType(n.Alias)
		if err != nil {
			return nil, fmt.Errorf("converting type alias: %w", err)
		}

		returnStmts = []goast.Stmt{&goast.DeclStmt{
			Decl: &goast.GenDecl{
				Tok: gotoken.TYPE,
				Specs: []goast.Spec{
					&goast.TypeSpec{
						Name: component.Ident(n.Identifier),
						Type: typ,
					},
				},
			},
		}}
	default:
		return nil, fmt.Errorf("unknown statement type '%T'", n)
	}

	// Attach //line directives to statements that don't already carry one.
	// Comments return early above; declarations embed their own //line inline.
	switch node.(type) {
	case *ast.Comment, *ast.Declaration:
	default:
		ln, _ := node.Pos()
		lineComment := fmt.Sprintf("\n//line %s:%d", t.file.Name, ln)
		lineDecl := &goast.DeclStmt{Decl: t.commentDecl(lineComment)[0]}
		returnStmts = append([]goast.Stmt{lineDecl}, returnStmts...)
	}

	return returnStmts, nil
}
