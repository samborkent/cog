package transpiler

import (
	"fmt"
	goast "go/ast"
	gotoken "go/token"
	"slices"

	"github.com/samborkent/cog/internal/ast"
	"github.com/samborkent/cog/internal/tokens"
	"github.com/samborkent/cog/internal/transpiler/comp"
	"github.com/samborkent/cog/internal/types"
)

func (t *Transpiler) convertExpr(node ast.Expression) (goast.Expr, error) {
	switch n := node.(type) {
	case *ast.ASCIILiteral:
		return n.Go(), nil
	case *ast.BoolLiteral:
		return n.Go(), nil
	case *ast.Builtin:
		return t.convertBuiltin(n)
	case *ast.Call:
		procType, ok := n.Identifier.ValueType.(*types.Procedure)
		if !ok {
			panic("failed to assert procedure type")
		}

		args := make([]goast.Expr, 0, len(procType.Parameters))

		if !procType.Function {
			// Pass context variable to all procedures.
			args = append(args, comp.ContextVar)
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
					// Add zero value of parameter type.
					args = append(args, comp.ZeroValue(t.convertType(procType.Parameters[i].Type)))
					continue
				}

				defaultExpr, err := t.convertExpr(procType.Parameters[i].Default.(ast.Expression))
				if err != nil {
					return nil, fmt.Errorf("parsing default value of input parameter in call expression: %w", err)
				}

				args = append(args, defaultExpr)
			}
		}

		t.symbols.MarkUsed(n.Identifier.Name)

		return &goast.CallExpr{
			Fun:  n.Identifier.Go(),
			Args: args,
		}, nil
	case *ast.Float32Literal:
		return n.Go(), nil
	case *ast.Float64Literal:
		return n.Go(), nil
	case *ast.GoCallExpression:
		expr := &goast.CallExpr{
			Fun: &goast.SelectorExpr{
				X:   n.Import.Go(),
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
			return &goast.TypeAssertExpr{
				X: &goast.CallExpr{
					Fun: &goast.SelectorExpr{
						X:   comp.ContextVar,
						Sel: &goast.Ident{Name: "Value"},
					},
					Args: []goast.Expr{
						&goast.CompositeLit{
							Type: &goast.Ident{Name: joinStr(name, "Key")},
						},
					},
				},
				Type: t.convertType(n.ValueType),
			}, nil
		}

		ident, ok := t.symbols.Resolve(name)
		if !ok {
			// New identifier
			return n.Go(), nil
		}

		t.symbols.MarkUsed(name)

		return ident, nil
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
		switch n.Left.Type().Underlying().Kind() {
		case types.ASCII:
			_, ok := t.imports["bytes"]
			if !ok {
				t.imports["bytes"] = &goast.ImportSpec{
					Path: &goast.BasicLit{
						Kind:  gotoken.STRING,
						Value: `"bytes"`,
					},
				}
			}

			return &goast.CallExpr{
				Fun: &goast.SelectorExpr{
					X:   &goast.Ident{Name: "bytes"},
					Sel: &goast.Ident{Name: "Equal"},
				},
				Args: []goast.Expr{lhs, rhs},
			}, nil
		}

		return &goast.BinaryExpr{
			X:  lhs,
			Op: convertBinaryOperator(n.Operator.Type),
			Y:  rhs,
		}, nil
	case *ast.Int64Literal:
		return n.Go(), nil
	case *ast.Prefix:
		right, err := t.convertExpr(n.Right)
		if err != nil {
			return nil, err
		}

		return &goast.UnaryExpr{
			Op: convertUnaryOperator(n.Operator.Type),
			X:  right,
		}, nil
	case *ast.ProcedureLiteral:
		// procType, ok := n.ProcdureType.(*types.Procedure)
		// if !ok {
		// 	panic("unable to assert procedure type")
		// }

		stmts := make([]goast.Stmt, 0, len(n.Body.Statements))

		// if len(procType.Parameters) > 0 {
		// 	// Enter parameter scope.
		// 	t.symbols = NewEnclosedSymbolTable(t.symbols)

		// 	for _, param := range procType.Parameters {
		// 		_ = t.symbols.Define(param.Name)
		// 	}
		// }

		if len(n.Body.Statements) > 0 {
			// Enter body scope.
			t.symbols = NewEnclosedSymbolTable(t.symbols)
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

		// if len(procType.Parameters) > 0 {
		// 	// Leave parameter scope.
		// 	t.symbols = t.symbols.Outer
		// }

		return &goast.FuncLit{
			Type: t.convertType(n.ProcdureType).(*goast.FuncType),
			Body: &goast.BlockStmt{
				List: stmts,
			},
		}, nil
	case *ast.Selector:
		name := convertExport(n.Identifier.Name, n.Identifier.Exported)

		ident, ok := t.symbols.Resolve(name)
		if !ok {
			return nil, fmt.Errorf("%s: unknown selector identifier", n.Identifier.Token)
		}

		t.symbols.MarkUsed(name)

		var exported bool

		switch n.Identifier.ValueType.Underlying().Kind() {
		case types.EnumKind:
			_, ok := n.Identifier.ValueType.Underlying().(*types.Enum)
			if !ok {
				panic("unable to assert enum type")
			}

			enumName := ident
			enumName.Name = enumName.Name + titleCaser.String(n.Field.Name)

			return enumName, nil
		case types.StructKind:
			structType, ok := n.Identifier.ValueType.Underlying().(*types.Struct)
			if !ok {
				panic("unable to assert struct type")
			}

			field := structType.Field(n.Field.Name)
			if field != nil {
				exported = field.Exported
			}

			return &goast.SelectorExpr{
				X:   n.Identifier.Go(),
				Sel: &goast.Ident{Name: convertExport(n.Field.Name, exported)},
			}, nil
		default:
			return nil, fmt.Errorf("%q: unknown type found for selector expression %q", n, n.Identifier.ValueType)
		}
	case *ast.SetLiteral:
		// TODO: handle not directly comparable types
		exprs := make([]goast.Expr, len(n.Values))

		for i, v := range n.Values {
			goExpr, err := t.convertExpr(v)
			if err != nil {
				return nil, fmt.Errorf("converting set literal value %d: %w", i, err)
			}

			exprs[i] = &goast.KeyValueExpr{
				Key:   goExpr,
				Value: &goast.CompositeLit{},
			}
		}

		return &goast.CompositeLit{
			Type: &goast.IndexExpr{
				X: &goast.SelectorExpr{
					X:   &goast.Ident{Name: "cog"},
					Sel: &goast.Ident{Name: "Set"},
				},
				Index: t.convertType(n.ValueType),
			},
			Elts: exprs,
		}, nil
	case *ast.StructLiteral:
		exprs := make([]goast.Expr, 0, len(n.Values))

		structType, ok := n.Type().Underlying().(*types.Struct)
		if !ok {
			panic("encountered struct literal with non-struct type")
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
				panic("struct literal contains non-existing field")
			}

			exprs = append(exprs, &goast.KeyValueExpr{
				// TODO: take into account exported value
				Key:   &goast.Ident{Name: convertExport(val.Name, structType.Fields[fieldIndex].Exported)},
				Value: expr,
			})
		}

		return &goast.CompositeLit{
			Elts: exprs,
		}, nil
	case *ast.Suffix:
		if n.Operator.Type != tokens.Question {
			return nil, fmt.Errorf("unknown suffix operator '%s'", n.Operator.Type.String())
		}

		ident, ok := n.Left.(*ast.Identifier)
		if !ok {
			panic("suffix operator applied to non-identifier")
		}

		name := convertExport(ident.Name, ident.Exported)

		// Mark identifier as used.
		symbol, ok := t.symbols.Resolve(name)
		if !ok {
			return nil, fmt.Errorf("identifier %q not found", name)
		}

		t.symbols.MarkUsed(name)

		return &goast.SelectorExpr{
			X:   symbol,
			Sel: &goast.Ident{Name: "Set"},
		}, nil
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
	case *ast.Uint64Literal:
		return n.Go(), nil
	case *ast.UnionLiteral:
		unionType, ok := n.UnionType.Underlying().(*types.Union)
		if !ok {
			panic("unable to assert union type")
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
	case *ast.UTF8Literal:
		return n.Go(), nil
	default:
		return nil, fmt.Errorf("unknown expression type '%T'", n)
	}
}

func convertBinaryOperator(t tokens.Type) gotoken.Token {
	switch t {
	case tokens.Plus:
		return gotoken.ADD
	case tokens.Minus:
		return gotoken.SUB
	case tokens.Asterisk:
		return gotoken.MUL
	case tokens.Divide:
		return gotoken.QUO
	case tokens.NotEqual:
		return gotoken.NEQ
	case tokens.Assign:
		return gotoken.ASSIGN
	case tokens.Equal:
		return gotoken.EQL
	case tokens.GT:
		return gotoken.GTR
	case tokens.GTEqual:
		return gotoken.GEQ
	case tokens.LT:
		return gotoken.LSS
	case tokens.LTEqual:
		return gotoken.LEQ
	case tokens.Declaration:
		return gotoken.DEFINE
	case tokens.And:
		return gotoken.LAND
	case tokens.Or:
		return gotoken.LOR
	default:
		panic("unknown binary operator " + t.String())
	}
}

func convertUnaryOperator(t tokens.Type) gotoken.Token {
	switch t {
	case tokens.Not:
		return gotoken.NOT
	case tokens.Minus:
		return gotoken.SUB
	default:
		panic("unknown unary operator " + t.String())
	}
}
