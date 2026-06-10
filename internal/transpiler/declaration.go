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

		name := component.ConvertExport(n.Assignment.Identifier.Token.Literal, n.Assignment.Identifier.Exported, n.Assignment.Identifier.Global)
		ident := &goast.Ident{Name: name}

		tok := gotoken.CONST

		if n.Assignment.Identifier.Qualifier == ast.QualifierVariable || mustBeVariable(n.Assignment.Identifier.ValueType.Kind()) {
			tok = gotoken.VAR
		}

		if n.Assignment.Expr == ast.ZeroExprIndex {
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

		assignmentExpr := t.Expr(n.Assignment.Expr)

		expr, err := t.convertExpr(assignmentExpr)
		if err != nil {
			return nil, err
		}

		bodyUsesDyn := t.usesDyn
		t.usesDyn = prevUsesDyn

		// Rule 5: immutable → var deep copy for pointer-like types.
		if n.Assignment.Identifier.Qualifier == ast.QualifierVariable &&
			types.IsPointerLike(assignmentExpr.Type()) {
			if astIdent, ok := assignmentExpr.(*ast.Identifier); ok &&
				astIdent.Qualifier != ast.QualifierVariable &&
				astIdent.Qualifier != ast.QualifierDynamic {
				expr = &goast.CallExpr{
					Fun: &goast.SelectorExpr{
						X:   &goast.Ident{Name: "cog"},
						Sel: &goast.Ident{Name: "Copy"},
					},
					Args: []goast.Expr{expr},
				}
			}
		}

		if assignmentExpr.Type().Kind() == types.ProcedureKind {
			// Procedure declaration - convert to function declaration
			funcLiteral, ok := expr.(*goast.FuncLit)
			if !ok {
				return nil, fmt.Errorf("unable to assert function literal for %q", n.Assignment.Identifier.Token.Literal)
			}

			// Create a function declaration instead of a variable declaration
			funcName := component.ConvertExport(n.Assignment.Identifier.Token.Literal, n.Assignment.Identifier.Exported, n.Assignment.Identifier.Global)
			funcDecl := &goast.FuncDecl{
				Name: &goast.Ident{Name: funcName},
				Type: funcLiteral.Type,
				Body: funcLiteral.Body,
			}

			// For procedure declarations, return function declaration instead of variable declaration
			if n.Assignment.Identifier.Token.Literal == "main" {
				t.addStdLibImport("context")
				t.addStdLibImport("os/signal")
				t.addStdLibImport("syscall")

				hasDynVars := len(t.dynamics) > 0
				needsContext := t.currentFileNeedsContext()
				// Only pass existing ctx to Signal when dyn init creates one for proc propagation.
				passCtx := hasDynVars && needsContext

				ctxIdent := &goast.Ident{Name: "ctx"}

				// Add signal notify context and adaptive GC.
				body := component.Signal(ctxIdent, passCtx)
				body = append(body, component.SetMaxProcs(), component.SetMemLimit(), component.AdaptiveGC(ctxIdent))
				funcDecl.Body.List = append(body, funcDecl.Body.List...)

				if hasDynVars || needsContext {
					if hasDynVars {
						// Main with dynamic variables: init dyn struct.
						dynIdent := &goast.Ident{Name: "dyn"}

						structElts := make([]goast.Expr, 0, len(t.dynamics))
						for name := range t.dynamics {
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

						if needsContext {
							// Also seed context for proc propagation.
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
					}

					if needsContext {
						// Remove context argument for main func.
						funcDecl.Type.Params.List = funcDecl.Type.Params.List[1:]
					}
				}

				t.injectDeferred(funcDecl.Body)
				t.injectArena(funcDecl.Body)

				return []goast.Decl{funcDecl}, nil
			}

			// Non-main proc: inject dyn preamble only when body uses dyn.
			if bodyUsesDyn && len(t.dynamics) > 0 {
				procType, ok := assignmentExpr.Type().(*types.Procedure)
				if ok && !procType.Function {
					funcDecl.Body.List = append(component.DynProcEntry(), funcDecl.Body.List...)
				}
			}

			if t.currentFileNeedsContext() {
				t.addStdLibImport("context")
			}

			t.injectDeferred(funcDecl.Body)
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
		recType, err := t.convertType(n.ReceiverType)
		if err != nil {
			return nil, err
		}

		methodType, err := t.convertType(n.Type)
		if err != nil {
			return nil, err
		}

		goType, ok := methodType.(*goast.FuncType)
		if !ok {
			return nil, errors.New("unable to cast method type as go FuncType")
		}

		var recIdent *goast.Ident

		if n.ReceiverIdent != nil {
			recIdent = component.Ident(n.ReceiverIdent)
		}

		methodBody, err := t.convertExpr(t.Expr(n.Body))
		if err != nil {
			return nil, err
		}

		funcLit, ok := methodBody.(*goast.FuncLit)
		if !ok {
			return nil, errors.New("unable to cast method body as go FuncLit")
		}

		return []goast.Decl{
			&goast.FuncDecl{
				Recv: component.Receiver(recIdent, recType),
				Type: goType,
				Body: funcLit.Body,
			},
		}, nil
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
			Name: component.IdentName(n.Alias.Name),
			Type: aliasType,
		}

		if len(n.Alias.TypeParameters) > 0 {
			typeParams, err := t.convertTypeParams(n.Alias.TypeParameters)
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
						Name: &goast.Ident{Name: component.ConvertExport(n.Alias.Name, n.Alias.Exported, n.Alias.Global) + "Hash"},
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

	switch a := n.Alias.Underlying().(type) {
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

	identifier := component.ConvertExport(n.Alias.Name, n.Alias.Exported, n.Alias.Global)

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
		val := t.Expr(ast.ExprIndex(enumVal.Value.Index))

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
			Names: []*goast.Ident{{Name: identifier + t.titleCaser.String(enumVal.Name)}},
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
		types.EitherKind,
		types.MapKind,
		types.ProcedureKind,
		types.SetKind,
		types.SliceKind,
		types.StructKind,
		types.TupleKind:
		return true
	default:
		return false
	}
}
