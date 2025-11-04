package transpiler

import (
	"fmt"
	goast "go/ast"
	gotoken "go/token"
	gotypes "go/types"
	"math"
	"strings"

	"github.com/samborkent/cog/internal/ast"
	"github.com/samborkent/cog/internal/transpiler/comp"
	"github.com/samborkent/cog/internal/types"
)

const delim = "_"

func joinStr(strs ...string) string {
	return strings.Join(strs, delim)
}

func (t *Transpiler) convertDecl(node ast.Node) ([]goast.Decl, error) {
	switch n := node.(type) {
	case *ast.Declaration:
		if n.Assignment.Identifier.Qualifier == ast.QualifierDynamic {
			keyIdent := t.symbols.Define(joinStr(convertExport(n.Assignment.Identifier.Name, n.Assignment.Identifier.Exported), "Key"))
			valIdent := t.symbols.Define(joinStr(convertExport(n.Assignment.Identifier.Name, n.Assignment.Identifier.Exported), "Default"))

			tok := gotoken.CONST
			if mustBeVariable(n.Assignment.Identifier.ValueType.Kind()) {
				tok = gotoken.VAR
			}

			var values []goast.Expr

			if n.Assignment.Expression != nil {
				val, err := t.convertExpr(n.Assignment.Expression)
				if err != nil {
					return nil, fmt.Errorf("converting dynamically variable expression: %w", err)
				}

				values = []goast.Expr{val}
			} else if tok == gotoken.CONST {
				// Variable has zero value.
				tok = gotoken.VAR
			}

			return []goast.Decl{
				&goast.GenDecl{
					Tok: gotoken.TYPE,
					Specs: []goast.Spec{
						&goast.TypeSpec{
							Name: keyIdent,
							Type: &goast.StructType{Fields: &goast.FieldList{}},
						},
					},
				},
				&goast.GenDecl{
					Tok: tok,
					Specs: []goast.Spec{
						&goast.ValueSpec{
							Names:  []*goast.Ident{valIdent},
							Type:   t.convertType(n.Assignment.Identifier.ValueType),
							Values: values,
						},
					},
				},
			}, nil
		}

		ident := t.symbols.Define(convertExport(n.Assignment.Identifier.Name, n.Assignment.Identifier.Exported))

		tok := gotoken.CONST

		if n.Assignment.Identifier.Qualifier == ast.QualifierVariable || mustBeVariable(n.Assignment.Identifier.ValueType.Kind()) {
			tok = gotoken.VAR
		}

		if n.Assignment.Expression == nil {
			return []goast.Decl{&goast.GenDecl{
				Tok: tok,
				Specs: []goast.Spec{
					&goast.ValueSpec{
						Names: []*goast.Ident{ident},
						Type:  t.convertType(n.Assignment.Identifier.ValueType),
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
				panic("unable to assert function literal")
			}

			if n.Assignment.Identifier.Name == "main" {
				// Main func
				ctxIdent := t.symbols.Define("ctx")
				if len(t.symbols.dynamics) > 0 {
					t.symbols.MarkUsed("ctx")
				}

				body := make([]goast.Stmt, 0, 1+len(t.symbols.dynamics))
				body = append(body, comp.ContextMain(ctxIdent))

				// Define dynamically scoped variables.
				for _, dyn := range t.symbols.dynamics {
					key := joinStr(convertExport(dyn.Name, dyn.Exported), "Key")
					val := joinStr(convertExport(dyn.Name, dyn.Exported), "Default")

					keyIdent, ok := t.symbols.Resolve(key)
					if !ok {
						return nil, fmt.Errorf("missing dynamic variable context key %q", key)
					}

					valIdent, ok := t.symbols.Resolve(val)
					if !ok {
						return nil, fmt.Errorf("missing dynamic variable default value %q", val)
					}

					t.symbols.MarkUsed(key)
					t.symbols.MarkUsed(val)

					body = append(body, &goast.AssignStmt{
						Tok: gotoken.ASSIGN,
						Lhs: []goast.Expr{ctxIdent},
						Rhs: []goast.Expr{
							&goast.CallExpr{
								Fun: &goast.SelectorExpr{
									X:   comp.ContextPackage,
									Sel: &goast.Ident{Name: "WithValue"},
								},
								Args: []goast.Expr{
									ctxIdent,
									&goast.CompositeLit{Type: keyIdent},
									valIdent,
								},
							},
						},
					})
				}

				funcLiteral.Body.List = append(body, funcLiteral.Body.List...)

				_, ok = t.imports["ctx"]
				if !ok {
					t.imports["ctx"] = &goast.ImportSpec{
						Path: &goast.BasicLit{
							Kind:  gotoken.STRING,
							Value: `"context"`,
						},
					}
				}

				// Remove context argument for main func.
				funcLiteral.Type.Params.List = funcLiteral.Type.Params.List[1:]

				return []goast.Decl{&goast.FuncDecl{
					Name: &goast.Ident{Name: "main"},
					Type: funcLiteral.Type,
					Body: funcLiteral.Body,
				}}, nil
			}
		}

		valueSpec := &goast.ValueSpec{
			Names:  []*goast.Ident{ident},
			Values: []goast.Expr{expr},
		}

		if n.Assignment.Identifier.ValueType != types.None {
			valueSpec.Type = t.convertType(n.Assignment.Identifier.ValueType)
		}

		return []goast.Decl{&goast.GenDecl{
			Tok:   tok,
			Specs: []goast.Spec{valueSpec},
		}}, nil
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
	enumType, ok := n.Alias.(*types.Enum)
	if !ok {
		panic(fmt.Sprintf("cannot convert type %q to enum", n.Alias))
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
				panic("cannot cast struct literal as composite literal")
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
					Type: t.convertType(enumType.ValueType),
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
	case types.ProcedureKind:
		return true
	default:
		return false
	}
}
