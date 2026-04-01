package transpiler

import (
	"fmt"
	goast "go/ast"
	gotoken "go/token"
	"slices"

	"github.com/samborkent/cog/internal/ast"
	"github.com/samborkent/cog/internal/tokens"
	"github.com/samborkent/cog/internal/transpiler/component"
	"github.com/samborkent/cog/internal/types"
)

func (t *Transpiler) convertExpr(node ast.Expression) (goast.Expr, error) {
	switch n := node.(type) {
	case *ast.ArrayLiteral:
		exprs := make([]goast.Expr, 0, len(n.Values))

		for _, val := range n.Values {
			expr, err := t.convertExpr(val)
			if err != nil {
				return nil, fmt.Errorf("converting expression in slice literal: %w", err)
			}

			exprs = append(exprs, expr)
		}

		arrayType, err := t.convertType(n.ArrayType)
		if err != nil {
			return nil, fmt.Errorf("converting array type: %w", err)
		}

		return &goast.CompositeLit{
			Type: arrayType,
			Elts: exprs,
		}, nil
	case *ast.ASCIILiteral:
		return component.ASCIILit(n.Value), nil
	case *ast.BoolLiteral:
		return component.BoolLit(n.Value), nil
	case *ast.Builtin:
		return t.convertBuiltin(n)
	case *ast.Call:
		procType, ok := n.Identifier.ValueType.(*types.Procedure)
		if !ok {
			return nil, fmt.Errorf("failed to assert procedure type for %q", n.Identifier.Name)
		}

		args := make([]goast.Expr, 0, len(procType.Parameters))

		if !procType.Function && t.needsContext {
			// Pass context variable to all procedures.
			args = append(args, component.ContextVar)
		}

		for _, arg := range n.Arguments {
			expr, err := t.convertExpr(arg)
			if err != nil {
				return nil, fmt.Errorf("transpiling argument in call expression: %w", err)
			}

			args = append(args, expr)
		}

		if len(procType.Parameters) > len(n.Arguments) {
			// The number of input parameters is greater than the number of arguments, so there are optional parameters.
			for i := len(args); i < len(procType.Parameters); i++ {
				if procType.Parameters[i].Default == nil {
					argType, err := t.convertType(procType.Parameters[i].Type)
					if err != nil {
						return nil, fmt.Errorf("converting call argument %d type: %w", i, err)
					}

					// Add zero value of parameter type.
					args = append(args, component.ZeroValue(argType))
					continue
				}

				defaultExpr, err := t.convertExpr(procType.Parameters[i].Default.(ast.Expression))
				if err != nil {
					return nil, fmt.Errorf("parsing default value of input parameter in call expression: %w", err)
				}

				args = append(args, defaultExpr)
			}
		}

		if n.Package == "" {
			if err := t.symbols.MarkUsed(n.Identifier.Name); err != nil {
				return nil, fmt.Errorf("marking call identifier used: %w", err)
			}
		}

		var fun goast.Expr
		if n.Package != "" {
			fun = &goast.SelectorExpr{
				X:   &goast.Ident{Name: n.Package},
				Sel: &goast.Ident{Name: convertExport(n.Identifier.Name, n.Identifier.Exported)},
			}
		} else {
			fun = component.Ident(n.Identifier)
		}

		return &goast.CallExpr{
			Fun:  fun,
			Args: args,
		}, nil
	case *ast.Complex32Literal:
		t.addCogImport()

		fromF32 := func(v float32) *goast.CallExpr {
			return &goast.CallExpr{
				Fun: &goast.SelectorExpr{
					X:   &goast.Ident{Name: "cog"},
					Sel: &goast.Ident{Name: "Float16Fromfloat32"},
				},
				Args: []goast.Expr{component.Float32Lit(v)},
			}
		}

		return &goast.CompositeLit{
			Type: &goast.SelectorExpr{
				X:   &goast.Ident{Name: "cog"},
				Sel: &goast.Ident{Name: "Complex32"},
			},
			Elts: []goast.Expr{
				&goast.KeyValueExpr{Key: &goast.Ident{Name: "Real"}, Value: fromF32(n.Value[0].Float32())},
				&goast.KeyValueExpr{Key: &goast.Ident{Name: "Imag"}, Value: fromF32(n.Value[1].Float32())},
			},
		}, nil
	case *ast.Float16Literal:
		t.addCogImport()

		return &goast.CallExpr{
			Fun: &goast.SelectorExpr{
				X:   &goast.Ident{Name: "cog"},
				Sel: &goast.Ident{Name: "Float16Fromfloat32"},
			},
			Args: []goast.Expr{component.Float32Lit(n.Value.Float32())},
		}, nil
	case *ast.Float32Literal:
		return component.Float32Lit(n.Value), nil
	case *ast.Float64Literal:
		return component.Float64Lit(n.Value), nil
	case *ast.GoCallExpression:
		expr := &goast.CallExpr{
			Fun: &goast.SelectorExpr{
				X:   &goast.Ident{Name: goStdLibAlias(n.Import.Name)},
				Sel: &goast.Ident{Name: n.CallIdentifier.Name},
			},
			Args: make([]goast.Expr, 0, len(n.Arguments)),
		}

		for _, arg := range n.Arguments {
			goarg, err := t.convertExpr(arg)
			if err != nil {
				return nil, fmt.Errorf("converting call argument: %w", err)
			}

			expr.Args = append(expr.Args, goarg)
		}

		return expr, nil
	case *ast.Identifier:
		name := convertExport(n.Name, n.Exported)

		if n.Qualifier == ast.QualifierDynamic {
			if t.inFunc {
				return nil, fmt.Errorf("func cannot reference dynamically scoped variable %q", n.Name)
			}

			t.usesDyn = true
			return component.DynRead(name), nil
		}

		ident, ok := t.symbols.Resolve(name)
		if !ok {
			// New identifier
			return component.Ident(n), nil
		}

		if err := t.symbols.MarkUsed(name); err != nil {
			return nil, fmt.Errorf("marking identifier used: %w", err)
		}

		// Auto-unwrap checked option/result identifiers to .Value
		if n.ValueType != nil {
			switch n.ValueType.Kind() {
			case types.OptionKind, types.ResultKind:
				return &goast.SelectorExpr{
					X:   ident,
					Sel: &goast.Ident{Name: "Value"},
				}, nil
			}
		}

		return ident, nil
	case *ast.Index:
		ident, err := t.convertExpr(n.Identifier)
		if err != nil {
			return nil, fmt.Errorf("converting identifier: %w", err)
		}

		index, err := t.convertExpr(n.Index)
		if err != nil {
			return nil, fmt.Errorf("converting index: %w", err)
		}

		return &goast.IndexExpr{
			X:     ident,
			Index: index,
		}, nil
	case *ast.Infix:
		lhs, err := t.convertExpr(n.Left)
		if err != nil {
			return nil, fmt.Errorf("converting lhs: %w", err)
		}

		rhs, err := t.convertExpr(n.Right)
		if err != nil {
			return nil, fmt.Errorf("converting rhs: %w", err)
		}

		// Use bytes.Equal for ascii type.
		switch n.Left.Type().Kind() {
		case types.ASCII:
			return &goast.CallExpr{
				Fun: &goast.SelectorExpr{
					X:   lhs,
					Sel: &goast.Ident{Name: "Equal"},
				},
				Args: []goast.Expr{rhs},
			}, nil
		case types.Complex32:
			t.addCogImport()

			binOp, err := convertBinaryOperator(n.Operator.Type)
			if err != nil {
				return nil, err
			}

			// Promote both operands to complex64 for the operation.
			lhsC64 := &goast.CallExpr{
				Fun: &goast.SelectorExpr{X: lhs, Sel: &goast.Ident{Name: "Complex64"}},
			}
			rhsC64 := &goast.CallExpr{
				Fun: &goast.SelectorExpr{X: rhs, Sel: &goast.Ident{Name: "Complex64"}},
			}

			binaryExpr := &goast.BinaryExpr{X: lhsC64, Op: binOp, Y: rhsC64}

			// Equality comparisons return bool directly; arithmetic wraps result back to Complex32.
			switch n.Operator.Type {
			case tokens.Equal, tokens.NotEqual:
				return binaryExpr, nil
			default:
				return &goast.CallExpr{
					Fun: &goast.SelectorExpr{
						X:   &goast.Ident{Name: "cog"},
						Sel: &goast.Ident{Name: "Complex32FromComplex64"},
					},
					Args: []goast.Expr{binaryExpr},
				}, nil
			}
		case types.Float16:
			t.addCogImport()

			binOp, err := convertBinaryOperator(n.Operator.Type)
			if err != nil {
				return nil, err
			}

			// Promote both operands to float32 for the operation.
			lhsF32 := &goast.CallExpr{
				Fun: &goast.SelectorExpr{X: lhs, Sel: &goast.Ident{Name: "Float32"}},
			}
			rhsF32 := &goast.CallExpr{
				Fun: &goast.SelectorExpr{X: rhs, Sel: &goast.Ident{Name: "Float32"}},
			}

			binaryExpr := &goast.BinaryExpr{X: lhsF32, Op: binOp, Y: rhsF32}

			// Comparisons return bool directly; arithmetic wraps result back to Float16.
			switch n.Operator.Type {
			case tokens.Equal, tokens.NotEqual, tokens.GT, tokens.GTEqual, tokens.LT, tokens.LTEqual:
				return binaryExpr, nil
			default:
				return &goast.CallExpr{
					Fun: &goast.SelectorExpr{
						X:   &goast.Ident{Name: "cog"},
						Sel: &goast.Ident{Name: "Float16Fromfloat32"},
					},
					Args: []goast.Expr{binaryExpr},
				}, nil
			}
		case types.Uint128:
			switch n.Operator.Type {
			case tokens.Plus:
				return &goast.CallExpr{
					Fun:  &goast.SelectorExpr{X: lhs, Sel: &goast.Ident{Name: "Add"}},
					Args: []goast.Expr{rhs},
				}, nil
			case tokens.Minus:
				return &goast.CallExpr{
					Fun:  &goast.SelectorExpr{X: lhs, Sel: &goast.Ident{Name: "Sub"}},
					Args: []goast.Expr{rhs},
				}, nil
			case tokens.Asterisk:
				return &goast.CallExpr{
					Fun:  &goast.SelectorExpr{X: lhs, Sel: &goast.Ident{Name: "Mul"}},
					Args: []goast.Expr{rhs},
				}, nil
			case tokens.Divide:
				return &goast.CallExpr{
					Fun:  &goast.SelectorExpr{X: lhs, Sel: &goast.Ident{Name: "Div"}},
					Args: []goast.Expr{rhs},
				}, nil
			case tokens.Equal:
				return &goast.CallExpr{
					Fun:  &goast.SelectorExpr{X: lhs, Sel: &goast.Ident{Name: "Equals"}},
					Args: []goast.Expr{rhs},
				}, nil
			case tokens.NotEqual:
				return &goast.UnaryExpr{
					Op: gotoken.NOT,
					X: &goast.CallExpr{
						Fun:  &goast.SelectorExpr{X: lhs, Sel: &goast.Ident{Name: "Equals"}},
						Args: []goast.Expr{rhs},
					},
				}, nil
			case tokens.LT:
				return &goast.BinaryExpr{
					X: &goast.CallExpr{
						Fun:  &goast.SelectorExpr{X: lhs, Sel: &goast.Ident{Name: "Cmp"}},
						Args: []goast.Expr{rhs},
					},
					Op: gotoken.LSS,
					Y:  component.Int64Lit(0),
				}, nil
			case tokens.LTEqual:
				return &goast.BinaryExpr{
					X: &goast.CallExpr{
						Fun:  &goast.SelectorExpr{X: lhs, Sel: &goast.Ident{Name: "Cmp"}},
						Args: []goast.Expr{rhs},
					},
					Op: gotoken.LEQ,
					Y:  component.Int64Lit(0),
				}, nil
			case tokens.GT:
				return &goast.BinaryExpr{
					X: &goast.CallExpr{
						Fun:  &goast.SelectorExpr{X: lhs, Sel: &goast.Ident{Name: "Cmp"}},
						Args: []goast.Expr{rhs},
					},
					Op: gotoken.GTR,
					Y:  component.Int64Lit(0),
				}, nil
			case tokens.GTEqual:
				return &goast.BinaryExpr{
					X: &goast.CallExpr{
						Fun:  &goast.SelectorExpr{X: lhs, Sel: &goast.Ident{Name: "Cmp"}},
						Args: []goast.Expr{rhs},
					},
					Op: gotoken.GEQ,
					Y:  component.Int64Lit(0),
				}, nil
			default:
				return nil, fmt.Errorf("unsupported operator %q for uint128", n.Operator.Type)
			}
		case types.Int128:
			switch n.Operator.Type {
			case tokens.Plus:
				return &goast.CallExpr{
					Fun:  &goast.SelectorExpr{X: lhs, Sel: &goast.Ident{Name: "Add"}},
					Args: []goast.Expr{rhs},
				}, nil
			case tokens.Minus:
				return &goast.CallExpr{
					Fun:  &goast.SelectorExpr{X: lhs, Sel: &goast.Ident{Name: "Sub"}},
					Args: []goast.Expr{rhs},
				}, nil
			case tokens.Asterisk:
				return &goast.CallExpr{
					Fun:  &goast.SelectorExpr{X: lhs, Sel: &goast.Ident{Name: "Mul"}},
					Args: []goast.Expr{rhs},
				}, nil
			case tokens.Divide:
				return &goast.CallExpr{
					Fun:  &goast.SelectorExpr{X: lhs, Sel: &goast.Ident{Name: "Div"}},
					Args: []goast.Expr{rhs},
				}, nil
			case tokens.Equal:
				return &goast.CallExpr{
					Fun:  &goast.SelectorExpr{X: lhs, Sel: &goast.Ident{Name: "Eq"}},
					Args: []goast.Expr{rhs},
				}, nil
			case tokens.NotEqual:
				return &goast.UnaryExpr{
					Op: gotoken.NOT,
					X: &goast.CallExpr{
						Fun:  &goast.SelectorExpr{X: lhs, Sel: &goast.Ident{Name: "Eq"}},
						Args: []goast.Expr{rhs},
					},
				}, nil
			case tokens.LT:
				return &goast.BinaryExpr{
					X: &goast.CallExpr{
						Fun:  &goast.SelectorExpr{X: lhs, Sel: &goast.Ident{Name: "Cmp"}},
						Args: []goast.Expr{rhs},
					},
					Op: gotoken.LSS,
					Y:  component.Int64Lit(0),
				}, nil
			case tokens.LTEqual:
				return &goast.BinaryExpr{
					X: &goast.CallExpr{
						Fun:  &goast.SelectorExpr{X: lhs, Sel: &goast.Ident{Name: "Cmp"}},
						Args: []goast.Expr{rhs},
					},
					Op: gotoken.LEQ,
					Y:  component.Int64Lit(0),
				}, nil
			case tokens.GT:
				return &goast.BinaryExpr{
					X: &goast.CallExpr{
						Fun:  &goast.SelectorExpr{X: lhs, Sel: &goast.Ident{Name: "Cmp"}},
						Args: []goast.Expr{rhs},
					},
					Op: gotoken.GTR,
					Y:  component.Int64Lit(0),
				}, nil
			case tokens.GTEqual:
				return &goast.BinaryExpr{
					X: &goast.CallExpr{
						Fun:  &goast.SelectorExpr{X: lhs, Sel: &goast.Ident{Name: "Cmp"}},
						Args: []goast.Expr{rhs},
					},
					Op: gotoken.GEQ,
					Y:  component.Int64Lit(0),
				}, nil
			default:
				return nil, fmt.Errorf("unsupported operator %q for int128", n.Operator.Type)
			}
		}

		binOp, err := convertBinaryOperator(n.Operator.Type)
		if err != nil {
			return nil, err
		}

		return &goast.BinaryExpr{
			X:  lhs,
			Op: binOp,
			Y:  rhs,
		}, nil
	case *ast.Int8Literal:
		return component.Int8Lit(n.Value), nil
	case *ast.Int16Literal:
		return component.Int16Lit(n.Value), nil
	case *ast.Int32Literal:
		return component.Int32Lit(n.Value), nil
	case *ast.Int64Literal:
		return component.Int64Lit(n.Value), nil
	case *ast.Int128Literal:
		t.addCogImport()

		return &goast.CallExpr{
			Fun: &goast.SelectorExpr{
				X:   &goast.Ident{Name: "cog"},
				Sel: &goast.Ident{Name: "Int128FromString"},
			},
			Args: []goast.Expr{component.UTF8Lit(n.Token.Literal)},
		}, nil
	case *ast.MapLiteral:
		// TODO: handle not directly comparable types
		exprs := make([]goast.Expr, len(n.Pairs))

		mType := n.MapType.(*types.Map)
		hasASCIIKey := mType.Key.Kind() == types.ASCII

		for i, pair := range n.Pairs {
			keyExpr, err := t.convertExpr(pair.Key)
			if err != nil {
				return nil, fmt.Errorf("converting map literal key %d: %w", i, err)
			}

			if hasASCIIKey {
				// Convert ascii literal keys to utf8 literals, since Go map keys must be comparable and byte slices are not.
				var indexExpr goast.Expr

				keyAlias, ok := pair.Key.Type().(*types.Alias)
				if ok {
					indexExpr = &goast.Ident{Name: convertExport(keyAlias.Name, keyAlias.Exported) + "Hash"}
				} else {
					indexExpr = &goast.SelectorExpr{
						X:   &goast.Ident{Name: "cog"},
						Sel: &goast.Ident{Name: "ASCIIHash"},
					}
				}

				keyExpr = &goast.CallExpr{
					Fun: &goast.IndexExpr{
						X: &goast.SelectorExpr{
							X:   &goast.Ident{Name: "cog"},
							Sel: &goast.Ident{Name: "HashASCII"},
						},
						Index: indexExpr,
					},
					Args: []goast.Expr{keyExpr},
				}
			} else {
				kExpr, err := t.convertExpr(pair.Key)
				if err != nil {
					return nil, fmt.Errorf("converting map literal key %d: %w", i, err)
				}

				keyExpr = kExpr
			}

			valExpr, err := t.convertExpr(pair.Value)
			if err != nil {
				return nil, fmt.Errorf("converting map literal value %d: %w", i, err)
			}

			exprs[i] = &goast.KeyValueExpr{
				Key:   keyExpr,
				Value: valExpr,
			}
		}

		mapType, err := t.convertType(n.MapType)
		if err != nil {
			return nil, fmt.Errorf("converting map key type: %w", err)
		}

		return &goast.CompositeLit{
			Type: mapType,
			Elts: exprs,
		}, nil
	case *ast.Prefix:
		right, err := t.convertExpr(n.Right)
		if err != nil {
			return nil, err
		}

		unaryOp, err := convertUnaryOperator(n.Operator.Type)
		if err != nil {
			return nil, err
		}

		// Float16 has no native operators; promote to float32, apply, demote.
		if n.Right.Type().Kind() == types.Float16 {
			t.addCogImport()

			return &goast.CallExpr{
				Fun: &goast.SelectorExpr{
					X:   &goast.Ident{Name: "cog"},
					Sel: &goast.Ident{Name: "Float16Fromfloat32"},
				},
				Args: []goast.Expr{
					&goast.UnaryExpr{
						Op: unaryOp,
						X: &goast.CallExpr{
							Fun: &goast.SelectorExpr{X: right, Sel: &goast.Ident{Name: "Float32"}},
						},
					},
				},
			}, nil
		}

		// Complex32 has no native operators; promote to complex64, apply, demote.
		if n.Right.Type().Kind() == types.Complex32 {
			t.addCogImport()

			return &goast.CallExpr{
				Fun: &goast.SelectorExpr{
					X:   &goast.Ident{Name: "cog"},
					Sel: &goast.Ident{Name: "Complex32FromComplex64"},
				},
				Args: []goast.Expr{
					&goast.UnaryExpr{
						Op: unaryOp,
						X: &goast.CallExpr{
							Fun: &goast.SelectorExpr{X: right, Sel: &goast.Ident{Name: "Complex64"}},
						},
					},
				},
			}, nil
		}

		// Int128 uses .Neg() method for unary minus.
		if n.Right.Type().Kind() == types.Int128 {
			return &goast.CallExpr{
				Fun: &goast.SelectorExpr{X: right, Sel: &goast.Ident{Name: "Neg"}},
			}, nil
		}

		// Collapse double negation: !(!x) → x
		if unaryOp == gotoken.NOT {
			if inner, ok := right.(*goast.UnaryExpr); ok && inner.Op == gotoken.NOT {
				return inner.X, nil
			}
		}

		return &goast.UnaryExpr{
			Op: unaryOp,
			X:  right,
		}, nil
	case *ast.ProcedureLiteral:
		stmts := make([]goast.Stmt, 0, len(n.Body.Statements))

		if len(n.Body.Statements) > 0 {
			// Enter body scope.
			t.symbols = NewEnclosedSymbolTable(t.symbols)
		}

		// Register function parameters in the transpiler symbol table so that
		// selector expressions (e.g. param.field) can resolve them.
		if procType, ok := n.ProcedureType.(*types.Procedure); ok {
			for _, param := range procType.Parameters {
				t.symbols.Define(param.Name)
				_ = t.symbols.MarkUsed(param.Name)
			}
		}

		// Track whether we're inside a func and reset usesDyn for this body.
		prevInFunc := t.inFunc
		prevUsesDyn := t.usesDyn
		t.usesDyn = false
		if procType, ok := n.ProcedureType.(*types.Procedure); ok {
			t.inFunc = procType.Function
		} else {
			t.inFunc = false
		}

		for _, s := range n.Body.Statements {
			stmt, err := t.convertStmt(s)
			if err != nil {
				return nil, err
			}

			stmts = append(stmts, stmt...)
		}

		if len(n.Body.Statements) > 0 {
			// Leave body scope.
			t.symbols = t.symbols.Outer
		}

		// Capture whether this body used dyn vars, then restore outer state.
		bodyUsesDyn := t.usesDyn
		t.usesDyn = prevUsesDyn || bodyUsesDyn
		t.inFunc = prevInFunc

		procType, err := t.convertType(n.ProcedureType)
		if err != nil {
			return nil, fmt.Errorf("converting procedure type: %w", err)
		}

		return &goast.FuncLit{
			Type: procType.(*goast.FuncType),
			Body: &goast.BlockStmt{
				List: stmts,
			},
		}, nil
	case *ast.Selector:
		// Check if this is a package import selector (pkg.Symbol).
		if types.IsNone(n.Identifier.ValueType) {
			return &goast.SelectorExpr{
				X:   &goast.Ident{Name: n.Identifier.Name},
				Sel: &goast.Ident{Name: convertExport(n.Field.Name, n.Field.Exported)},
			}, nil
		}

		name := convertExport(n.Identifier.Name, n.Identifier.Exported)

		ident, ok := t.symbols.Resolve(name)
		if !ok {
			return nil, fmt.Errorf("%s: unknown selector identifier", n.Identifier.Token)
		}

		if err := t.symbols.MarkUsed(name); err != nil {
			return nil, fmt.Errorf("marking selector identifier used: %w", err)
		}

		var exported bool

		switch n.Identifier.ValueType.Kind() {
		case types.EnumKind, types.ErrorKind:
			enumName := ident
			enumName.Name = enumName.Name + titleCaser.String(n.Field.Name)

			return enumName, nil
		case types.StructKind:
			structType, ok := n.Identifier.ValueType.Underlying().(*types.Struct)
			if !ok {
				return nil, fmt.Errorf("unable to assert struct type for %q", n.Identifier.Name)
			}

			field := structType.Field(n.Field.Name)
			if field != nil {
				exported = field.Exported
			}

			return &goast.SelectorExpr{
				X:   component.Ident(n.Identifier),
				Sel: &goast.Ident{Name: convertExport(n.Field.Name, exported)},
			}, nil
		default:
			return nil, fmt.Errorf("%q: unknown type found for selector expression %q", n, n.Identifier.ValueType)
		}
	case *ast.SetLiteral:
		// TODO: handle not directly comparable types
		exprs := make([]goast.Expr, len(n.Values))

		setType := n.SetType.(*types.Set)
		isASCII := setType.Element.Kind() == types.ASCII

		for i, v := range n.Values {
			goExpr, err := t.convertExpr(v)
			if err != nil {
				return nil, fmt.Errorf("converting set literal value %d: %w", i, err)
			}

			if isASCII {
				// Convert ascii literal keys to hash, since Go map keys must be comparable and byte slices are not.
				var indexExpr goast.Expr

				elemAlias, ok := setType.Element.(*types.Alias)
				if ok {
					indexExpr = &goast.Ident{Name: convertExport(elemAlias.Name, elemAlias.Exported) + "Hash"}
				} else {
					indexExpr = &goast.SelectorExpr{
						X:   &goast.Ident{Name: "cog"},
						Sel: &goast.Ident{Name: "ASCIIHash"},
					}
				}

				goExpr = &goast.CallExpr{
					Fun: &goast.IndexExpr{
						X: &goast.SelectorExpr{
							X:   &goast.Ident{Name: "cog"},
							Sel: &goast.Ident{Name: "HashASCII"},
						},
						Index: indexExpr,
					},
					Args: []goast.Expr{goExpr},
				}
			}

			exprs[i] = &goast.KeyValueExpr{
				Key:   goExpr,
				Value: &goast.CompositeLit{},
			}
		}

		var indexExpr goast.Expr

		if isASCII {
			aliasType, ok := setType.Element.(*types.Alias)
			if ok {
				indexExpr = &goast.Ident{Name: convertExport(aliasType.Name, aliasType.Exported) + "Hash"}
			} else {
				indexExpr = &goast.SelectorExpr{
					X:   &goast.Ident{Name: "cog"},
					Sel: &goast.Ident{Name: "ASCIIHash"},
				}
			}
		} else {
			elemType, err := t.convertType(setType.Element)
			if err != nil {
				return nil, fmt.Errorf("converting set element type: %w", err)
			}

			indexExpr = elemType
		}

		t.addCogImport()

		return &goast.CompositeLit{
			Type: &goast.IndexExpr{
				X: &goast.SelectorExpr{
					X:   &goast.Ident{Name: "cog"},
					Sel: &goast.Ident{Name: "Set"},
				},
				Index: indexExpr,
			},
			Elts: exprs,
		}, nil
	case *ast.SliceLiteral:
		exprs := make([]goast.Expr, 0, len(n.Values))

		for _, val := range n.Values {
			expr, err := t.convertExpr(val)
			if err != nil {
				return nil, fmt.Errorf("converting expression in slice literal: %w", err)
			}

			exprs = append(exprs, expr)
		}

		elemType, err := t.convertType(n.ElementType)
		if err != nil {
			return nil, fmt.Errorf("converting array type: %w", err)
		}

		return &goast.CompositeLit{
			Type: &goast.ArrayType{
				Elt: elemType,
			},
			Elts: exprs,
		}, nil
	case *ast.StructLiteral:
		exprs := make([]goast.Expr, 0, len(n.Values))

		structType, ok := n.Type().Underlying().(*types.Struct)
		if !ok {
			return nil, fmt.Errorf("encountered struct literal with non-struct type %q", n.Type())
		}

		for _, val := range n.Values {
			expr, err := t.convertExpr(val.Value)
			if err != nil {
				return nil, fmt.Errorf("converting expression in struct literal: %w", err)
			}

			fieldIndex := slices.IndexFunc(structType.Fields, func(f *types.Field) bool {
				return f.Name == val.Name
			})
			if fieldIndex == -1 {
				return nil, fmt.Errorf("struct literal contains non-existing field %q", val.Name)
			}

			exprs = append(exprs, &goast.KeyValueExpr{
				Key:   &goast.Ident{Name: convertExport(val.Name, structType.Fields[fieldIndex].Exported)},
				Value: expr,
			})
		}

		return &goast.CompositeLit{
			Elts: exprs,
		}, nil
	case *ast.Suffix:
		if n.Operator.Type != tokens.Question && n.Operator.Type != tokens.Not {
			return nil, fmt.Errorf("unknown suffix operator '%s'", n.Operator.Type.String())
		}

		ident, ok := n.Left.(*ast.Identifier)
		if !ok {
			return nil, fmt.Errorf("suffix operator applied to non-identifier")
		}

		name := convertExport(ident.Name, ident.Exported)

		// Mark identifier as used.
		symbol, ok := t.symbols.Resolve(name)
		if !ok {
			return nil, fmt.Errorf("identifier %q not found", name)
		}

		if err := t.symbols.MarkUsed(name); err != nil {
			return nil, fmt.Errorf("marking suffix identifier used: %w", err)
		}

		leftType := ident.ValueType
		if leftType == nil {
			return nil, fmt.Errorf("suffix operator applied to untyped identifier")
		}

		switch n.Operator.Type {
		case tokens.Question:
			switch leftType.Kind() {
			case types.OptionKind:
				return &goast.SelectorExpr{
					X:   symbol,
					Sel: &goast.Ident{Name: "Set"},
				}, nil
			case types.ResultKind:
				return &goast.UnaryExpr{
					Op: gotoken.NOT,
					X: &goast.SelectorExpr{
						X:   symbol,
						Sel: &goast.Ident{Name: "IsError"},
					},
				}, nil
			default:
				return nil, fmt.Errorf("? operator requires option or result type")
			}
		case tokens.Not:
			if leftType.Kind() != types.ResultKind {
				return nil, fmt.Errorf("! operator requires result type")
			}

			return &goast.SelectorExpr{
				X:   symbol,
				Sel: &goast.Ident{Name: "Error"},
			}, nil
		}

		return nil, fmt.Errorf("unknown suffix operator '%s'", n.Operator.Type.String())
	case *ast.TupleLiteral:
		values := make([]goast.Expr, 0, len(n.Values))

		for _, v := range n.Values {
			val, err := t.convertExpr(v)
			if err != nil {
				return nil, fmt.Errorf("converting expression in tuple literal: %w", err)
			}

			values = append(values, val)
		}

		return &goast.CompositeLit{
			Elts: values,
		}, nil
	case *ast.Uint8Literal:
		return component.Uint8Lit(n.Value), nil
	case *ast.Uint16Literal:
		return component.Uint16Lit(n.Value), nil
	case *ast.Uint32Literal:
		return component.Uint32Lit(n.Value), nil
	case *ast.Uint64Literal:
		return component.Uint64Lit(n.Value), nil
	case *ast.Uint128Literal:
		t.addCogImport()

		return &goast.CallExpr{
			Fun: &goast.SelectorExpr{
				X:   &goast.Ident{Name: "cog"},
				Sel: &goast.Ident{Name: "Uint128FromString"},
			},
			Args: []goast.Expr{component.UTF8Lit(n.Value.String())},
		}, nil
	case *ast.UnionLiteral:
		unionType, ok := n.UnionType.Underlying().(*types.Union)
		if !ok {
			return nil, fmt.Errorf("unable to assert union type")
		}

		expr, err := t.convertExpr(n.Value)
		if err != nil {
			return nil, fmt.Errorf("converting union literal value: %w", err)
		}

		if n.Tag {
			return &goast.CompositeLit{
				Type: &goast.Ident{Name: convertExport(unionType.String(), unionType.Exported)},
				Elts: []goast.Expr{
					&goast.KeyValueExpr{
						Key:   &goast.Ident{Name: "Or"},
						Value: expr,
					},
					&goast.KeyValueExpr{
						Key:   &goast.Ident{Name: "Tag"},
						Value: &goast.Ident{Name: "true"},
					},
				},
			}, nil
		}

		return &goast.CompositeLit{
			Elts: []goast.Expr{&goast.KeyValueExpr{
				Key:   &goast.Ident{Name: "Either"},
				Value: expr,
			}},
		}, nil
	case *ast.ResultLiteral:
		_, ok := n.ResultType.Underlying().(*types.Result)
		if !ok {
			return nil, fmt.Errorf("unable to assert result type")
		}

		expr, err := t.convertExpr(n.Value)
		if err != nil {
			return nil, fmt.Errorf("converting result literal value: %w", err)
		}

		goType, err := t.convertType(n.ResultType)
		if err != nil {
			return nil, fmt.Errorf("converting result type: %w", err)
		}

		if n.IsError {
			return &goast.CompositeLit{
				Type: goType,
				Elts: []goast.Expr{
					&goast.KeyValueExpr{
						Key:   &goast.Ident{Name: "Error"},
						Value: expr,
					},
					&goast.KeyValueExpr{
						Key:   &goast.Ident{Name: "IsError"},
						Value: &goast.Ident{Name: "true"},
					},
				},
			}, nil
		}

		return &goast.CompositeLit{
			Type: goType,
			Elts: []goast.Expr{
				&goast.KeyValueExpr{
					Key:   &goast.Ident{Name: "Value"},
					Value: expr,
				},
			},
		}, nil
	case *ast.UTF8Literal:
		return component.UTF8Lit(n.Value), nil
	default:
		return nil, fmt.Errorf("unknown expression type '%T'", n)
	}
}

func convertBinaryOperator(t tokens.Type) (gotoken.Token, error) {
	switch t {
	case tokens.Plus:
		return gotoken.ADD, nil
	case tokens.Minus:
		return gotoken.SUB, nil
	case tokens.Asterisk:
		return gotoken.MUL, nil
	case tokens.Divide:
		return gotoken.QUO, nil
	case tokens.NotEqual:
		return gotoken.NEQ, nil
	case tokens.Assign:
		return gotoken.ASSIGN, nil
	case tokens.Equal:
		return gotoken.EQL, nil
	case tokens.GT:
		return gotoken.GTR, nil
	case tokens.GTEqual:
		return gotoken.GEQ, nil
	case tokens.LT:
		return gotoken.LSS, nil
	case tokens.LTEqual:
		return gotoken.LEQ, nil
	case tokens.Declaration:
		return gotoken.DEFINE, nil
	case tokens.And:
		return gotoken.LAND, nil
	case tokens.Or:
		return gotoken.LOR, nil
	default:
		return gotoken.ILLEGAL, fmt.Errorf("unknown binary operator %s", t.String())
	}
}

func convertUnaryOperator(t tokens.Type) (gotoken.Token, error) {
	switch t {
	case tokens.Not:
		return gotoken.NOT, nil
	case tokens.Minus:
		return gotoken.SUB, nil
	default:
		return gotoken.ILLEGAL, fmt.Errorf("unknown unary operator %s", t.String())
	}
}
