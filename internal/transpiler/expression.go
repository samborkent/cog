package transpiler

import (
	"fmt"
	goast "go/ast"
	gotoken "go/token"
	"slices"

	"github.com/samborkent/cog/internal/ast"
	"github.com/samborkent/cog/internal/tokens"
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
	case *ast.Float32Literal:
		return n.Go(), nil
	case *ast.Float64Literal:
		return n.Go(), nil
	case *ast.GoCallExpression:
		expr := &goast.CallExpr{
			Fun: &goast.SelectorExpr{
				X:   n.Import.Go(),
				Sel: n.Call.Identifier.Go(),
			},
			Args: make([]goast.Expr, 0, len(n.Call.Arguments)),
		}

		for _, arg := range n.Call.Arguments {
			goarg, err := t.convertExpr(arg)
			if err != nil {
				return nil, fmt.Errorf("converting call argument: %w", err)
			}

			expr.Args = append(expr.Args, goarg)
		}

		return expr, nil
	case *ast.Identifier:
		if _, ok := identifiers[n.Name]; ok {
			// Mark identifier as used.
			identifiers[n.Name].Name = n.Name
			return identifiers[n.Name], nil
		} else {
			return n.Go(), nil
		}
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
	case *ast.Selector:
		if ident, ok := identifiers[n.Identifier.Name]; ok {
			// Mark identifier as used.
			ident.Name = convertExport(n.Identifier.Name, n.Identifier.Exported)
		}

		var exported bool

		switch n.Identifier.ValueType.Underlying().Kind() {
		case types.EnumKind:
			_, ok := n.Identifier.ValueType.Underlying().(*types.Enum)
			if !ok {
				panic("unable to assert enum type")
			}

			enumName := convertExport(n.Identifier.Name, n.Identifier.Exported)

			return &goast.Ident{Name: enumName + titleCaser.String(n.Field.Name)}, nil
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
				X:   &goast.Ident{Name: convertExport(n.Identifier.Name, n.Identifier.Exported)},
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
				Index: convertType(n.ValueType),
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
			Type: &goast.Ident{Name: n.StructType.String()},
			Elts: exprs,
		}, nil
	case *ast.TupleLiteral:
		tupleType, ok := n.TupleType.Underlying().(*types.Tuple)
		if !ok {
			panic("unable to assert tuple type")
		}

		values := make([]goast.Expr, 0, len(n.Values))

		for _, v := range n.Values {
			val, err := t.convertExpr(v)
			if err != nil {
				return nil, fmt.Errorf("converting expression in tuple literal: %w", err)
			}

			values = append(values, val)
		}

		return &goast.CompositeLit{
			Type: &goast.Ident{Name: convertExport(tupleType.String(), tupleType.Exported)},
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
						Key:   &goast.Ident{Name: convertExport("or", unionType.Exported)},
						Value: expr,
					},
					&goast.KeyValueExpr{
						Key:   &goast.Ident{Name: convertExport("tag", unionType.Exported)},
						Value: &goast.Ident{Name: "true"},
					},
				},
			}, nil
		}

		return &goast.CompositeLit{
			Type: &goast.Ident{Name: convertExport(unionType.String(), unionType.Exported)},
			Elts: []goast.Expr{&goast.KeyValueExpr{
				Key:   &goast.Ident{Name: convertExport("either", unionType.Exported)},
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
	case tokens.Xor:
		return gotoken.NEQ
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
