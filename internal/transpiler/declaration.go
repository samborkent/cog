package transpiler

import (
	"fmt"
	goast "go/ast"
	gotoken "go/token"
	gotypes "go/types"
	"math"

	"github.com/samborkent/cog/internal/ast"
	"github.com/samborkent/cog/internal/transpiler/component"
	"github.com/samborkent/cog/internal/types"
)

func (t *Transpiler) convertDecl(node ast.Node) ([]goast.Decl, error) {
	switch n := node.(type) {
	case *ast.Comment:
		return t.commentDecl(n.Text), nil
	case *ast.Declaration:
		if n.Assignment.Identifier.Qualifier == ast.QualifierDynamic {
			// Dynamic variable declarations are handled collectively via the
			// cogDyn struct generated in Transpile(). Individual dyn declarations
			// emit no Go declarations.
			return nil, nil
		}

		ident := t.symbols.Define(convertExport(n.Assignment.Identifier.Name, n.Assignment.Identifier.Exported))

		tok := gotoken.CONST

		if n.Assignment.Identifier.Qualifier == ast.QualifierVariable || mustBeVariable(n.Assignment.Identifier.ValueType.Kind()) {
			tok = gotoken.VAR
		}

		if n.Assignment.Expression == nil {
			declType, err := t.convertType(n.Assignment.Identifier.ValueType)
			if err != nil {
				return nil, fmt.Errorf("converting type in declaration: %w", err)
			}

			return []goast.Decl{&goast.GenDecl{
				Tok: tok,
				Specs: []goast.Spec{
					&goast.ValueSpec{
						Names: []*goast.Ident{ident},
						Type:  declType,
					},
				},
			}}, nil
		}

		expr, err := t.convertExpr(n.Assignment.Expression)
		if err != nil {
			return nil, err
		}

		if n.Assignment.Expression.Type().Kind() == types.ProcedureKind {
			// Procedure declaration
			funcLiteral, ok := expr.(*goast.FuncLit)
			if !ok {
				return nil, fmt.Errorf("unable to assert function literal for %q", n.Assignment.Identifier.Name)
			}

			if n.Assignment.Identifier.Name == "main" {
				hasDynVars := len(t.symbols.dynamics) > 0

				if hasDynVars || t.needsContext {
					if hasDynVars {
						// Main with dynamic variables: init dyn struct.
						dynIdent := t.symbols.Define("dyn")
						if err := t.symbols.MarkUsed("dyn"); err != nil {
							return nil, fmt.Errorf("marking dyn used: %w", err)
						}

						structElts := make([]goast.Expr, 0, len(t.symbols.dynamics))
						for name := range t.symbols.dynamics {
							defaultExpr, hasDefault := t.dynDefaults[name]
							if !hasDefault {
								continue
							}

							val, err := t.convertExpr(defaultExpr)
							if err != nil {
								return nil, fmt.Errorf("converting dynamic variable %q default: %w", name, err)
							}

							structElts = append(structElts, &goast.KeyValueExpr{
								Key:   &goast.Ident{Name: name},
								Value: val,
							})
						}

						structLit := &goast.CompositeLit{
							Type: component.DynStructType,
							Elts: structElts,
						}

						if t.needsContext {
							// Also seed context for proc propagation.
							ctxIdent := t.symbols.Define("ctx")
							if err := t.symbols.MarkUsed("ctx"); err != nil {
								return nil, fmt.Errorf("marking ctx used: %w", err)
							}

							body := component.DynMainInit(dynIdent, ctxIdent, structLit)
							funcLiteral.Body.List = append(body, funcLiteral.Body.List...)
						} else {
							// No procs: just create dyn struct, no context needed.
							funcLiteral.Body.List = append([]goast.Stmt{
								&goast.AssignStmt{
									Tok: gotoken.DEFINE,
									Lhs: []goast.Expr{dynIdent},
									Rhs: []goast.Expr{structLit},
								},
							}, funcLiteral.Body.List...)
						}
					} else {
						// Main with procs but no dynamic variables: just init context.
						ctxIdent := t.symbols.Define("ctx")
						if err := t.symbols.MarkUsed("ctx"); err != nil {
							return nil, fmt.Errorf("marking ctx used: %w", err)
						}

						funcLiteral.Body.List = append(
							[]goast.Stmt{component.ContextMain(ctxIdent)},
							funcLiteral.Body.List...,
						)
					}

					if t.needsContext {
						t.imports["ctx"] = &goast.ImportSpec{
							Path: &goast.BasicLit{
								Kind:  gotoken.STRING,
								Value: `"context"`,
							},
						}

						// Remove context argument for main func.
						funcLiteral.Type.Params.List = funcLiteral.Type.Params.List[1:]
					}
				}

				return []goast.Decl{&goast.FuncDecl{
					Name: &goast.Ident{Name: "main"},
					Type: funcLiteral.Type,
					Body: funcLiteral.Body,
				}}, nil
			}

			// Non-main proc with dynamic variables: inject copy-on-entry preamble.
			if len(t.symbols.dynamics) > 0 {
				procType, ok := n.Assignment.Expression.Type().(*types.Procedure)
				if ok && !procType.Function {
					funcLiteral.Body.List = append(component.DynProcEntry(), funcLiteral.Body.List...)
				}
			}
		}

		// Replace type string with type name if missing (for structs, tuples, unions).
		compositeLiteral, ok := expr.(*goast.CompositeLit)
		if ok && compositeLiteral.Type == nil {
			compositeLiteral.Type = &goast.Ident{Name: convertExport(n.Assignment.Identifier.Type().String(), n.Assignment.Identifier.Exported)}
		}

		valueSpec := &goast.ValueSpec{
			Names:  []*goast.Ident{ident},
			Values: []goast.Expr{expr},
		}

		if n.Assignment.Identifier.ValueType != types.None {
			valType, err := t.convertType(n.Assignment.Identifier.ValueType)
			if err != nil {
				return nil, fmt.Errorf("converting type in declaration: %w", err)
			}

			valueSpec.Type = valType
		}

		return []goast.Decl{&goast.GenDecl{
			Tok:   tok,
			Specs: []goast.Spec{valueSpec},
		}}, nil
	case *ast.Type:
		if n.Alias.Underlying().Kind() == types.EnumKind {
			return t.convertEnumDecl(n)
		}

		aliasType, err := t.convertType(n.Alias)
		if err != nil {
			return nil, fmt.Errorf("converting alias type: %w", err)
		}

		decls := make([]goast.Decl, 0, 2)
		decls = append(decls, &goast.GenDecl{
			Tok: gotoken.TYPE,
			Specs: []goast.Spec{
				&goast.TypeSpec{
					Name: &goast.Ident{Name: convertExport(n.Identifier.Name, n.Identifier.Exported)},
					Type: aliasType,
				},
			},
		})

		if n.Alias.Kind() == types.ASCII {
			// Generate hash type for ASCII alias, to allow usage of ASCII as map keys.
			decls = append(decls, &goast.GenDecl{
				Tok: gotoken.TYPE,
				Specs: []goast.Spec{
					&goast.TypeSpec{
						Name: &goast.Ident{Name: convertExport(n.Identifier.Name, n.Identifier.Exported) + "Hash"},
						Type: &goast.Ident{Name: gotypes.Typ[gotypes.Uint64].String()},
					},
				},
			})
		}

		return decls, nil
	default:
		return nil, fmt.Errorf("unknown declaration type '%T'", n)
	}
}

func (t *Transpiler) convertEnumDecl(n *ast.Type) ([]goast.Decl, error) {
	enumType, ok := n.Alias.(*types.Enum)
	if !ok {
		return nil, fmt.Errorf("cannot convert type %q to enum", n.Alias)
	}

	identifier := convertExport(n.Identifier.Name, n.Identifier.Exported)

	enumName := identifier + "Enum"

	enumTypeIdent := gotypes.Typ[gotypes.Uint8].String()

	if len(enumType.Values) > math.MaxUint8 {
		enumTypeIdent = gotypes.Typ[gotypes.Uint16].String()
	}

	specs := make([]goast.Spec, 0, len(enumType.Values))
	exprs := make([]goast.Expr, 0, len(enumType.Values))

	for i, enumVal := range enumType.Values {
		val := enumVal.Value.(ast.Expression)

		expr, err := t.convertExpr(val)
		if err != nil {
			return nil, fmt.Errorf("converting expression %d in enum literal: %w", i, err)
		}

		if val.Type().Underlying().Kind() == types.StructKind {
			compositeLit, ok := expr.(*goast.CompositeLit)
			if !ok {
				return nil, fmt.Errorf("cannot cast struct literal as composite literal in enum")
			}

			// Remove type for struct literals, to avoid naming issues with type aliases.
			compositeLit.Type = nil
		}

		spec := &goast.ValueSpec{
			Names: []*goast.Ident{{Name: identifier + titleCaser.String(enumVal.Name)}},
		}

		if i == 0 {
			spec.Type = &goast.Ident{Name: enumName}
			spec.Values = []goast.Expr{&goast.Ident{Name: "iota"}}
		}

		specs = append(specs, spec)
		exprs = append(exprs, expr)
	}

	typeName := &goast.Ident{Name: identifier + "Type"}

	enumValType, err := t.convertType(enumType.ValueType)
	if err != nil {
		return nil, fmt.Errorf("converting enum value type: %w", err)
	}

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
					Type: enumValType,
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

func mustBeVariable(t types.Kind) bool {
	switch t {
	case types.ArrayKind,
		types.MapKind,
		types.ProcedureKind,
		types.SetKind,
		types.SliceKind,
		types.StructKind,
		types.TupleKind,
		types.UnionKind:
		return true
	default:
		return false
	}
}
