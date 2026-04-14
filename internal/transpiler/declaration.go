package transpiler

import (
	"errors"
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
		text := n.Text

		commentLn, _ := n.Pos()
		if commentLn != t.lastSourceLine {
			text = "\n" + text
		}

		return t.commentDecl(text), nil
	case *ast.Declaration:
		if n.Assignment.Identifier.Qualifier == ast.QualifierDynamic {
			// Dynamic variable declarations are handled collectively via the
			// cogDyn struct generated in Transpile(). Individual dyn declarations
			// emit no Go declarations.
			return nil, nil
		}

		ident := t.symbols.Define(component.ConvertExport(n.Assignment.Identifier.Name, n.Assignment.Identifier.Exported, n.Assignment.Identifier.Global))

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

		prevUsesDyn := t.usesDyn
		t.usesDyn = false

		// Pre-define ctx and dyn in the symbol table for main so that
		// the body conversion can reference them (e.g. passing ctx to procs).
		if n.Assignment.Identifier.Name == "main" {
			if len(t.symbols.dynamics) > 0 {
				t.symbols.Define("dyn")
			}
			if t.currentFileNeedsContext() {
				t.symbols.Define("ctx")
			}
		}

		expr, err := t.convertExpr(n.Assignment.Expression)
		if err != nil {
			return nil, err
		}

		bodyUsesDyn := t.usesDyn
		t.usesDyn = prevUsesDyn

		if n.Assignment.Expression.Type().Kind() == types.ProcedureKind {
			// Procedure declaration - convert to function declaration
			funcLiteral, ok := expr.(*goast.FuncLit)
			if !ok {
				return nil, fmt.Errorf("unable to assert function literal for %q", n.Assignment.Identifier.Name)
			}

			// Create a function declaration instead of a variable declaration
			funcName := component.ConvertExport(n.Assignment.Identifier.Name, n.Assignment.Identifier.Exported, n.Assignment.Identifier.Global)
			funcDecl := &goast.FuncDecl{
				Name: &goast.Ident{Name: funcName},
				Type: funcLiteral.Type,
				Body: funcLiteral.Body,
			}

			// For procedure declarations, return function declaration instead of variable declaration
			if n.Assignment.Identifier.Name == "main" {
				hasDynVars := len(t.symbols.dynamics) > 0

				if hasDynVars || t.currentFileNeedsContext() {
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

						if t.currentFileNeedsContext() {
							// Also seed context for proc propagation.
							ctxIdent := t.symbols.Define("ctx")
							if err := t.symbols.MarkUsed("ctx"); err != nil {
								return nil, fmt.Errorf("marking ctx used: %w", err)
							}

							body := component.DynMainInit(dynIdent, ctxIdent, structLit)
							funcDecl.Body.List = append(body, funcDecl.Body.List...)
						} else {
							// No procs: just create dyn struct, no context needed.
							funcDecl.Body.List = append([]goast.Stmt{
								&goast.AssignStmt{
									Tok: gotoken.DEFINE,
									Lhs: []goast.Expr{dynIdent},
									Rhs: []goast.Expr{structLit},
								},
							}, funcDecl.Body.List...)
						}
					} else {
						// Main with procs but no dynamic variables: just init context.
						ctxIdent := t.symbols.Define("ctx")

						funcDecl.Body.List = append(
							[]goast.Stmt{component.ContextMain(ctxIdent)},
							funcDecl.Body.List...,
						)
					}

					if t.currentFileNeedsContext() {
						t.addStdLibImport("context")

						// Remove context argument for main func.
						funcDecl.Type.Params.List = funcDecl.Type.Params.List[1:]
					}
				}

				t.injectArena(funcDecl.Body)

				return []goast.Decl{funcDecl}, nil
			}

			// Non-main proc: inject dyn preamble only when body uses dyn.
			if bodyUsesDyn && len(t.symbols.dynamics) > 0 {
				procType, ok := n.Assignment.Expression.Type().(*types.Procedure)
				if ok && !procType.Function {
					funcDecl.Body.List = append(component.DynProcEntry(), funcDecl.Body.List...)
				}
			}

			if t.currentFileNeedsContext() {
				t.addStdLibImport("context")
			}

			t.injectArena(funcDecl.Body)

			// Return function declaration for procedures
			return []goast.Decl{funcDecl}, nil
		}

		// Replace type string with type name if missing (for structs, tuples, unions).
		compositeLiteral, ok := expr.(*goast.CompositeLit)
		if ok && compositeLiteral.Type == nil {
			litType := n.Assignment.Identifier.Type()
			litName := litType.String()

			// Handle exported type aliases.
			litAlias, ok := litType.(*types.Alias)
			if ok {
				litName = component.ConvertExport(litAlias.Name, litAlias.Exported, litAlias.Global)
			}

			compositeLiteral.Type = &goast.Ident{Name: litName}
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
	case *ast.Method:
		prevInMethod := t.inMethod
		t.inMethod = true

		decls, err := t.convertDecl(n.Declaration)

		t.inMethod = prevInMethod

		if err != nil {
			return nil, err
		}

		if len(decls) != 1 {
			return nil, fmt.Errorf("transpilation of method declaration resulted in an unexpected number of declarations: %d", len(decls))
		}

		funcDecl, ok := decls[0].(*goast.FuncDecl)
		if !ok {
			return nil, errors.New("unable to assert function declartion during method transpilation")
		}

		funcDecl.Recv = component.Receiver(n.Receiver, n.Reference)

		return decls, nil
	case *ast.Type:
		if n.Alias.Kind() == types.EnumKind || n.Alias.Kind() == types.ErrorKind {
			return t.convertEnumDecl(n)
		}

		aliasType, err := t.convertType(n.Alias)
		if err != nil {
			return nil, fmt.Errorf("converting alias type: %w", err)
		}

		decls := make([]goast.Decl, 0, 2)

		typeSpec := &goast.TypeSpec{
			Name: component.Ident(n.Identifier),
			Type: aliasType,
		}

		if len(n.TypeParameters) > 0 {
			typeParams, err := t.convertTypeParams(n.TypeParameters)
			if err != nil {
				return nil, fmt.Errorf("converting type parameters: %w", err)
			}

			typeSpec.TypeParams = typeParams
		}

		decls = append(decls, &goast.GenDecl{
			Tok:   gotoken.TYPE,
			Specs: []goast.Spec{typeSpec},
		})

		if n.Alias.Kind() == types.ASCII {
			// Generate hash type for ASCII alias, to allow usage of ASCII as map keys.
			decls = append(decls, &goast.GenDecl{
				Tok: gotoken.TYPE,
				Specs: []goast.Spec{
					&goast.TypeSpec{
						Name: &goast.Ident{Name: component.ConvertExport(n.Identifier.Name, n.Identifier.Exported, n.Identifier.Global) + "Hash"},
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
	var (
		valueType types.Type
		values    []*types.EnumValue
	)

	switch a := n.Alias.(type) {
	case *types.Enum:
		valueType = a.ValueType
		values = a.Values
	case *types.Error:
		if a.ValueType != nil {
			valueType = a.ValueType
		} else {
			valueType = types.Basics[types.UTF8]
		}

		values = a.Values
	default:
		return nil, fmt.Errorf("cannot convert type %q to enum", n.Alias)
	}

	identifier := component.ConvertExport(n.Identifier.Name, n.Identifier.Exported, n.Identifier.Global)

	var enumName string
	if n.Alias.Kind() == types.ErrorKind {
		enumName = identifier + "Error"
	} else {
		enumName = identifier + "Enum"
	}

	enumTypeIdent := gotypes.Typ[gotypes.Uint8].String()

	if len(values) > math.MaxUint8 {
		enumTypeIdent = gotypes.Typ[gotypes.Uint16].String()
	}

	specs := make([]goast.Spec, 0, len(values))
	exprs := make([]goast.Expr, 0, len(values))

	for i, enumVal := range values {
		val := enumVal.Value.(ast.Expression)

		expr, err := t.convertExpr(val)
		if err != nil {
			return nil, fmt.Errorf("converting expression %d in enum literal: %w", i, err)
		}

		if val.Type().Kind() == types.StructKind {
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

	enumValType, err := t.convertType(valueType)
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
