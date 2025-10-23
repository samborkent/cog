package transpiler

import (
	"fmt"
	goast "go/ast"
	gotoken "go/token"
	gotypes "go/types"
	"math"

	"github.com/samborkent/cog/internal/ast"
	"github.com/samborkent/cog/internal/types"
)

func (t *Transpiler) convertDecl(node ast.Node) ([]goast.Decl, error) {
	switch n := node.(type) {
	case *ast.Declaration:
		ident := t.symbols.Define(convertExport(n.Assignment.Identifier.Name, n.Assignment.Identifier.Exported))

		if n.Assignment.Expression == nil {
			return []goast.Decl{&goast.GenDecl{
				Tok: gotoken.VAR,
				Specs: []goast.Spec{
					&goast.ValueSpec{
						Names: []*goast.Ident{ident},
						Type:  t.convertType(n.Type),
					},
				},
			}}, nil
		}

		expr, err := t.convertExpr(n.Assignment.Expression)
		if err != nil {
			return nil, err
		}

		valueSpec := &goast.ValueSpec{
			Names:  []*goast.Ident{ident},
			Values: []goast.Expr{expr},
		}

		if n.Type != types.None {
			valueSpec.Type = t.convertType(n.Type)
		}

		tok := gotoken.VAR

		if n.Constant {
			tok = gotoken.CONST
		}

		return []goast.Decl{&goast.GenDecl{
			Tok:   tok,
			Specs: []goast.Spec{valueSpec},
		}}, nil
	case *ast.Procedure:
		funcName := n.Identifier.Go()

		var funcDecl goast.Decl

		if funcName.Name == "main" {
			gofunc := &goast.FuncDecl{
				Name: funcName,
				Type: &goast.FuncType{
					Params: &goast.FieldList{
						List: make([]*goast.Field, 0, len(n.InputParameters)),
					},
				},
			}

			mainWithContext := false

			if len(n.InputParameters) > 0 {
				// Enter parameter scope.
				t.symbols = NewEnclosedSymbolTable(t.symbols)
			}

			for i, param := range n.InputParameters {
				// Handle context argument for main func.
				if i == 0 && funcName.Name == "main" && param.Identifier.Name == "ctx" {
					mainWithContext = true
					_ = t.symbols.Define(param.Identifier.Name)
					continue
				}

				ident := t.symbols.Define(param.Identifier.Name)

				gofunc.Type.Params.List = append(gofunc.Type.Params.List, &goast.Field{
					Names: []*goast.Ident{ident},
					Type:  t.convertType(param.ValueType),
				})
			}

			if mainWithContext {
				ident, ok := t.symbols.Resolve("ctx")
				if !ok {
					panic("missing ctx identifier")
				}

				// Add signal context to top of main if it has a context parameter
				gofunc.Body = &goast.BlockStmt{
					List: []goast.Stmt{
						// ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, os.Kill)
						&goast.AssignStmt{
							Lhs: []goast.Expr{
								ident,
								&goast.Ident{
									Name: "_stop",
								},
							},
							Tok: gotoken.DEFINE,
							Rhs: []goast.Expr{
								&goast.CallExpr{
									Fun: &goast.SelectorExpr{
										X: &goast.Ident{
											Name: "signal",
										},
										Sel: &goast.Ident{
											Name: "NotifyContext",
										},
									},
									Args: []goast.Expr{
										&goast.CallExpr{
											Fun: &goast.SelectorExpr{
												X: &goast.Ident{
													Name: "context",
												},
												Sel: &goast.Ident{
													Name: "Background",
												},
											},
										},
										&goast.SelectorExpr{
											X: &goast.Ident{
												Name: "os",
											},
											Sel: &goast.Ident{
												Name: "Interrupt",
											},
										},
										&goast.SelectorExpr{
											X: &goast.Ident{
												Name: "os",
											},
											Sel: &goast.Ident{
												Name: "Kill",
											},
										},
									},
								},
							},
						},
						// defer stop()
						&goast.DeferStmt{
							Call: &goast.CallExpr{
								Fun: &goast.Ident{
									Name: "_stop",
								},
							},
						},
					},
				}

				// Define imports
				_, ok = t.imports["ctx"]
				if !ok {
					t.imports["ctx"] = &goast.ImportSpec{
						Path: &goast.BasicLit{
							Kind:  gotoken.STRING,
							Value: `"context"`,
						},
					}
				}

				_, ok = t.imports["os"]
				if !ok {
					t.imports["os"] = &goast.ImportSpec{
						Path: &goast.BasicLit{
							Kind:  gotoken.STRING,
							Value: `"os"`,
						},
					}
				}

				_, ok = t.imports["os/signal"]
				if !ok {
					t.imports["os/signal"] = &goast.ImportSpec{
						Path: &goast.BasicLit{
							Kind:  gotoken.STRING,
							Value: `"os/signal"`,
						},
					}
				}
			}

			if n.ReturnType != nil {
				gofunc.Type.Results = &goast.FieldList{List: []*goast.Field{{Type: t.convertType(n.ReturnType)}}}
			}

			if n.Body != nil {
				stmts := make([]goast.Stmt, 0, len(n.Body.Statements))

				if len(n.Body.Statements) > 0 {
					// Enter body scope.
					t.symbols = NewEnclosedSymbolTable(t.symbols)
				}

				for _, stmt := range n.Body.Statements {
					s, err := t.convertStmt(stmt)
					if err != nil {
						return nil, err
					}

					stmts = append(stmts, s...)
				}

				if len(n.Body.Statements) > 0 {
					// Leave body scope.
					t.symbols = t.symbols.Outer
				}

				if gofunc.Body != nil && gofunc.Body.List != nil {
					// Add to statement list if some statements already exist.
					gofunc.Body.List = append(gofunc.Body.List, stmts...)
				} else {
					gofunc.Body = &goast.BlockStmt{
						List: stmts,
					}
				}
			}

			if len(n.InputParameters) > 0 {
				// Leave parameter scope.
				t.symbols = t.symbols.Outer
			}

			funcDecl = gofunc
		}

		return []goast.Decl{funcDecl}, nil
	case *ast.Type:
		if n.Alias.Underlying().Kind() == types.EnumKind {
			return t.convertEnumDecl(n)
		}

		return []goast.Decl{&goast.GenDecl{
			Tok: gotoken.TYPE,
			Specs: []goast.Spec{
				&goast.TypeSpec{
					Name: &goast.Ident{Name: convertExport(n.Identifier.Name, n.Identifier.Exported)},
					Type: t.convertType(n.Alias),
				},
			},
		}}, nil
	default:
		return nil, fmt.Errorf("unknown declaration type '%T'", n)
	}
}

func (t *Transpiler) convertEnumDecl(n *ast.Type) ([]goast.Decl, error) {
	_, ok := n.Alias.(*types.Enum)
	if !ok {
		panic(fmt.Sprintf("cannot convert type %q to enum", n.Alias))
	}

	identifier := convertExport(n.Identifier.Name, n.Identifier.Exported)

	enumName := identifier + "Enum"

	enumTypeIdent := gotypes.Typ[gotypes.Uint8].String()

	if len(n.Literal.Values) > math.MaxUint8 {
		enumTypeIdent = gotypes.Typ[gotypes.Uint16].String()
	}

	specs := make([]goast.Spec, 0, len(n.Literal.Values))
	exprs := make([]goast.Expr, 0, len(n.Literal.Values))

	for i, val := range n.Literal.Values {
		expr, err := t.convertExpr(val.Value)
		if err != nil {
			return nil, fmt.Errorf("converting expression %d in enum literal: %w", i, err)
		}

		if val.Value.Type().Underlying().Kind() == types.StructKind {
			compositeLit, ok := expr.(*goast.CompositeLit)
			if !ok {
				panic("cannot cast struct literal as composite literal")
			}

			// Remove type for struct literals, to avoid naming issues with type aliases.
			compositeLit.Type = nil
		}

		spec := &goast.ValueSpec{
			Names: []*goast.Ident{{Name: identifier + titleCaser.String(val.Identifier.Name)}},
		}

		if i == 0 {
			spec.Type = &goast.Ident{Name: enumName}
			spec.Values = []goast.Expr{&goast.Ident{Name: "iota"}}
		}

		specs = append(specs, spec)
		exprs = append(exprs, expr)
	}

	typeName := &goast.Ident{Name: identifier + "Type"}

	return []goast.Decl{
		// Enum type declaration
		&goast.GenDecl{
			Tok: gotoken.TYPE,
			Specs: []goast.Spec{
				&goast.TypeSpec{
					Name: &goast.Ident{Name: enumName},
					Type: &goast.Ident{Name: enumTypeIdent},
				},
			},
		},
		// Enum index declaration
		&goast.GenDecl{
			Tok:   gotoken.CONST,
			Specs: specs,
		},
		// Enum underlyng type declaration
		&goast.GenDecl{
			Tok: gotoken.TYPE,
			Specs: []goast.Spec{
				&goast.TypeSpec{
					Name: typeName,
					Type: t.convertType(n.Literal.ValueType),
				},
			},
		},
		// Enum value declaration
		&goast.GenDecl{
			Tok: gotoken.VAR,
			Specs: []goast.Spec{
				&goast.ValueSpec{
					Names: []*goast.Ident{{Name: identifier}},
					Values: []goast.Expr{
						&goast.CompositeLit{
							Type: &goast.ArrayType{
								Elt: typeName,
							},
							Elts: exprs,
						},
					},
				},
			},
		},
	}, nil
}
