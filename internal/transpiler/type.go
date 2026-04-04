package transpiler

import (
	"errors"
	"fmt"
	goast "go/ast"
	gotoken "go/token"
	gotypes "go/types"
	"strconv"

	"github.com/samborkent/cog/internal/ast"
	"github.com/samborkent/cog/internal/transpiler/component"
	"github.com/samborkent/cog/internal/types"
)

func (t *Transpiler) convertType(typ types.Type) (goast.Expr, error) {
	// Try to retrieve type expression from cache.
	expr, ok := t.typeCache[typ]
	if ok {
		return expr, nil
	}

	if alias, ok := typ.(*types.Alias); ok {
		name := component.ConvertExport(alias.Name, alias.Exported, alias.Global)

		switch alias.Kind() {
		case types.EnumKind:
			name += "Enum"
		case types.ErrorKind:
			name += "Error"
		}

		expr = &goast.Ident{Name: name}
		t.typeCache[typ] = expr

		return expr, nil
	}

	switch typ.Kind() {
	case types.ArrayKind:
		sliceType, ok := typ.(*types.Array)
		if !ok {
			return nil, errors.New("unable to assert array type")
		}

		lenExpr, err := t.convertExpr(sliceType.Length.(ast.Expression))
		if err != nil {
			return nil, fmt.Errorf("converting array length expression: %w", err)
		}

		elemType, err := t.convertType(sliceType.Element)
		if err != nil {
			return nil, fmt.Errorf("converting array element type: %w", err)
		}

		expr = &goast.ArrayType{
			Len: lenExpr,
			Elt: elemType,
		}
	case types.ASCII:
		t.addCogImport()

		expr = &goast.SelectorExpr{
			X:   &goast.Ident{Name: "cog"},
			Sel: &goast.Ident{Name: "ASCII"},
		}
	case types.Bool:
		expr = &goast.Ident{Name: gotypes.TypeString(gotypes.Typ[gotypes.Bool], nil)}
	case types.Complex32:
		t.addCogImport()

		expr = &goast.SelectorExpr{
			X:   &goast.Ident{Name: "cog"},
			Sel: &goast.Ident{Name: "Complex32"},
		}
	case types.Complex64:
		expr = &goast.Ident{Name: gotypes.TypeString(gotypes.Typ[gotypes.Complex64], nil)}
	case types.Complex128:
		expr = &goast.Ident{Name: gotypes.TypeString(gotypes.Typ[gotypes.Complex128], nil)}
	case types.Float16:
		t.addCogImport()

		expr = &goast.SelectorExpr{
			X:   &goast.Ident{Name: "cog"},
			Sel: &goast.Ident{Name: "Float16"},
		}
	case types.Float32:
		expr = &goast.Ident{Name: gotypes.TypeString(gotypes.Typ[gotypes.Float32], nil)}
	case types.Float64:
		expr = &goast.Ident{Name: gotypes.TypeString(gotypes.Typ[gotypes.Float64], nil)}
	case types.Int8:
		expr = &goast.Ident{Name: gotypes.TypeString(gotypes.Typ[gotypes.Int8], nil)}
	case types.Int16:
		expr = &goast.Ident{Name: gotypes.TypeString(gotypes.Typ[gotypes.Int16], nil)}
	case types.Int32:
		expr = &goast.Ident{Name: gotypes.TypeString(gotypes.Typ[gotypes.Int32], nil)}
	case types.Int64:
		expr = &goast.Ident{Name: gotypes.TypeString(gotypes.Typ[gotypes.Int64], nil)}
	case types.Int128:
		t.addCogImport()

		expr = &goast.SelectorExpr{
			X:   &goast.Ident{Name: "cog"},
			Sel: &goast.Ident{Name: "Int128"},
		}
	case types.MapKind:
		mapType, ok := typ.(*types.Map)
		if !ok {
			return nil, errors.New("unable to assert map type")
		}

		var keyExpr goast.Expr

		if mapType.Key.Kind() == types.ASCII {
			// ASCII is an alias of []byte, which is not a valid map key type in Go. Use string instead.
			aliasType, ok := mapType.Key.(*types.Alias)
			if ok {
				keyExpr = &goast.Ident{Name: component.ConvertExport(aliasType.Name, aliasType.Exported, aliasType.Global) + "Hash"}
			} else {
				keyExpr = &goast.SelectorExpr{
					X:   &goast.Ident{Name: "cog"},
					Sel: &goast.Ident{Name: "ASCIIHash"},
				}
			}
		} else {
			keyType, err := t.convertType(mapType.Key)
			if err != nil {
				return nil, fmt.Errorf("converting map key type: %w", err)
			}

			keyExpr = keyType
		}

		valType, err := t.convertType(mapType.Value)
		if err != nil {
			return nil, fmt.Errorf("converting map value type: %w", err)
		}

		expr = &goast.MapType{
			Key:   keyExpr,
			Value: valType,
		}
	case types.OptionKind:
		optionType, ok := typ.(*types.Option)
		if !ok {
			return nil, errors.New("unable to assert option type")
		}

		valueType, err := t.convertType(optionType.Value)
		if err != nil {
			return nil, fmt.Errorf("converting option value type: %w", err)
		}

		t.addCogImport()

		expr = &goast.IndexExpr{
			X: &goast.SelectorExpr{
				X:   &goast.Ident{Name: "cog"},
				Sel: &goast.Ident{Name: "Option"},
			},
			Index: valueType,
		}
	case types.ProcedureKind:
		procType, ok := typ.(*types.Procedure)
		if !ok {
			return nil, errors.New("unable to assert procedure type")
		}

		inputParams := make([]*goast.Field, 0, len(procType.Parameters))

		if !procType.Function && t.needsContext {
			// Procedures take context when context propagation is needed.
			inputParams = append(inputParams, component.ContextArg)
		}

		for i, param := range procType.Parameters {
			paramType, err := t.convertType(param.Type)
			if err != nil {
				return nil, fmt.Errorf("converting parameter %d type: %w", i, err)
			}

			inputParams = append(inputParams, &goast.Field{
				Names: []*goast.Ident{{Name: param.Name}},
				Type:  paramType,
			})
		}

		funcType := &goast.FuncType{
			Params: &goast.FieldList{List: inputParams},
		}

		if len(procType.TypeParams) > 0 {
			funcType.TypeParams = t.convertTypeParams(procType.TypeParams)
		}

		if procType.ReturnType != nil {
			returnType, err := t.convertType(procType.ReturnType)
			if err != nil {
				return nil, fmt.Errorf("converting return type: %w", err)
			}

			funcType.Results = &goast.FieldList{List: []*goast.Field{
				{Type: returnType},
			}}
		}

		expr = funcType
	case types.ReferenceKind:
		refType, ok := typ.(*types.Reference)
		if !ok {
			return nil, errors.New("unable to assert reference type")
		}

		valueType, err := t.convertType(refType.Value)
		if err != nil {
			return nil, fmt.Errorf("converting reference value type: %w", err)
		}

		expr = &goast.StarExpr{X: valueType}
	case types.Uint8:
		expr = &goast.Ident{Name: gotypes.TypeString(gotypes.Typ[gotypes.Uint8], nil)}
	case types.Uint16:
		expr = &goast.Ident{Name: gotypes.TypeString(gotypes.Typ[gotypes.Uint16], nil)}
	case types.Uint32:
		expr = &goast.Ident{Name: gotypes.TypeString(gotypes.Typ[gotypes.Uint32], nil)}
	case types.Uint64:
		expr = &goast.Ident{Name: gotypes.TypeString(gotypes.Typ[gotypes.Uint64], nil)}
	case types.Uint128:
		t.addCogImport()

		expr = &goast.SelectorExpr{
			X:   &goast.Ident{Name: "cog"},
			Sel: &goast.Ident{Name: "Uint128"},
		}
	case types.SetKind:
		setType, ok := typ.(*types.Set)
		if !ok {
			return nil, errors.New("unable to assert set type")
		}

		var indexExpr goast.Expr

		if setType.Element.Kind() == types.ASCII {
			// ASCII is an alias of []byte, which is not a valid map key type in Go. Use hash of ASCII instead.
			aliasType, ok := setType.Element.(*types.Alias)
			if ok {
				indexExpr = &goast.Ident{Name: component.ConvertExport(aliasType.Name, aliasType.Exported, aliasType.Global) + "Hash"}
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

		expr = &goast.IndexExpr{
			X: &goast.SelectorExpr{
				X:   &goast.Ident{Name: "cog"},
				Sel: &goast.Ident{Name: "Set"},
			},
			Index: indexExpr,
		}
	case types.SliceKind:
		sliceType, ok := typ.(*types.Slice)
		if !ok {
			return nil, errors.New("unable to assert slice type")
		}

		elemType, err := t.convertType(sliceType.Element)
		if err != nil {
			return nil, fmt.Errorf("converting slice element type: %w", err)
		}

		expr = &goast.ArrayType{
			Elt: elemType,
		}
	case types.StructKind:
		structType, ok := typ.(*types.Struct)
		if !ok {
			return nil, errors.New("unable to assert struct type")
		}

		fields := make([]*goast.Field, len(structType.Fields))

		for i := range structType.Fields {
			field, err := t.convertField(structType.Fields[i])
			if err != nil {
				return nil, err
			}

			fields[i] = field
		}

		expr = &goast.StructType{
			Fields: &goast.FieldList{
				List: fields,
			},
		}
	case types.TupleKind:
		tupleType, ok := typ.(*types.Tuple)
		if !ok {
			return nil, errors.New("unable to assert tuple type")
		}

		fields := make([]*goast.Field, 0, len(tupleType.Types))

		for i := range tupleType.Types {
			elemType, err := t.convertType(tupleType.Types[i])
			if err != nil {
				return nil, fmt.Errorf("converting tuple element %d type: %w", i, err)
			}

			fields = append(fields, &goast.Field{
				Names: []*goast.Ident{{Name: component.ConvertExport("t"+strconv.Itoa(i), tupleType.Exported, tupleType.Global)}},
				Type:  elemType,
			})
		}

		expr = &goast.StructType{
			Fields: &goast.FieldList{
				List: fields,
			},
		}
	case types.UnionKind:
		unionType, ok := typ.(*types.Union)
		if !ok {
			return nil, errors.New("unable to assert union type")
		}

		if len(unionType.Variants) != 2 {
			return nil, errors.New("transpiling union types as variables requires exactly 2 variants")
		}

		leftType, err := t.convertType(unionType.Variants[0])
		if err != nil {
			return nil, fmt.Errorf("converting union left type: %w", err)
		}

		rightType, err := t.convertType(unionType.Variants[1])
		if err != nil {
			return nil, fmt.Errorf("converting union right type: %w", err)
		}

		t.addCogImport()

		expr = &goast.IndexListExpr{
			X:       component.Selector(component.IdentName("cog"), "Either"),
			Indices: []goast.Expr{leftType, rightType},
		}
	case types.ResultKind:
		resultType, ok := typ.(*types.Result)
		if !ok {
			return nil, errors.New("unable to assert result type")
		}

		valueType, err := t.convertType(resultType.Value)
		if err != nil {
			return nil, fmt.Errorf("converting result value type: %w", err)
		}

		errorType, err := t.convertType(resultType.Error)
		if err != nil {
			return nil, fmt.Errorf("converting result error type: %w", err)
		}

		t.addCogImport()

		expr = &goast.IndexListExpr{
			X: &goast.SelectorExpr{
				X:   &goast.Ident{Name: "cog"},
				Sel: &goast.Ident{Name: "Result"},
			},
			Indices: []goast.Expr{valueType, errorType},
		}
	case types.UTF8:
		expr = &goast.Ident{Name: gotypes.TypeString(gotypes.Typ[gotypes.String], nil)}
	case types.AnyKind:
		expr = &goast.Ident{Name: "any"}
	case types.GenericKind:
		tp, ok := typ.(*types.TypeParam)
		if ok {
			expr = &goast.Ident{Name: tp.Name}
		} else {
			expr = t.convertConstraint(typ)
		}
	default:
		return nil, fmt.Errorf("unknown type %q", typ)
	}

	t.typeCache[typ] = expr

	return expr, nil
}

// convertConstraint maps a cog constraint type to its Go equivalent.
// For compound constraints (int, uint, etc.) it emits an inline tilde-union
// with the Go-native subset of the constraint's types.
func (t *Transpiler) convertConstraint(typ types.Type) goast.Expr {
	if typ.Kind() == types.AnyKind {
		return component.Any
	}

	g, ok := typ.(*types.Generic)
	if !ok {
		return component.Any
	}

	switch g.String() {
	case "comparable":
		return component.Comparable
	case "ordered":
		t.addStdLibImport("cmp")
		return component.CmpOrdered
	case "int":
		return component.TildeUnion(component.GoInt...)
	case "uint":
		return component.TildeUnion(component.GoUint...)
	case "float":
		return component.TildeUnion(component.GoFloat...)
	case "complex":
		return component.TildeUnion(component.GoComplex...)
	case "string":
		// cog string = ascii | utf8. ascii is Go []byte (no tilde constraint);
		// utf8 is Go string. Only ~string is representable.
		return component.TildeUnion(component.GoString...)
	case "signed":
		return component.TildeUnion(append(append(component.GoInt, component.GoFloat...), component.GoComplex...)...)
	case "number":
		return component.TildeUnion(append(append(append(component.GoInt, component.GoUint...), component.GoFloat...), component.GoComplex...)...)
	case "summable":
		return component.TildeUnion(append(append(append(append(component.GoInt, component.GoUint...), component.GoFloat...), component.GoComplex...), component.GoString...)...)
	default:
		return component.Any
	}
}

// convertTypeParamConstraints converts a TypeParam's constraint list to a
// single Go constraint expression. A single constraint maps directly; multiple
// constraints are joined with | into a union interface.
func (t *Transpiler) convertTypeParamConstraints(tp *types.TypeParam) goast.Expr {
	if tp.Constraint == nil || tp.Constraint.Kind() == types.AnyKind {
		return component.Any
	}

	if union, ok := tp.Constraint.(*types.Union); ok {
		for _, c := range union.Variants {
			if c.Kind() == types.AnyKind {
				return component.Any
			}
		}

		if len(union.Variants) == 1 {
			return t.convertConstraint(union.Variants[0])
		}

		var leaves []goast.Expr
		for _, c := range union.Variants {
			flattenBinaryOr(t.convertConstraint(c), &leaves)
		}

		expr := leaves[0]
		for _, leaf := range leaves[1:] {
			expr = &goast.BinaryExpr{
				X:  expr,
				Op: gotoken.OR,
				Y:  leaf,
			}
		}

		return &goast.InterfaceType{
			Methods: &goast.FieldList{
				List: []*goast.Field{{Type: expr}},
			},
		}
	}

	return t.convertConstraint(tp.Constraint)
}

// flattenBinaryOr recursively collects the leaf operands of a BinaryExpr tree
// joined by OR (|). Non-binary expressions are appended directly.
func flattenBinaryOr(e goast.Expr, out *[]goast.Expr) {
	if bin, ok := e.(*goast.BinaryExpr); ok && bin.Op == gotoken.OR {
		flattenBinaryOr(bin.X, out)
		flattenBinaryOr(bin.Y, out)

		return
	}

	*out = append(*out, e)
}

// convertTypeParams converts cog type parameters to a Go ast.FieldList
// suitable for TypeSpec.TypeParams or FuncType.TypeParams.
func (t *Transpiler) convertTypeParams(params []*types.TypeParam) *goast.FieldList {
	if len(params) == 0 {
		return nil
	}

	fields := make([]*goast.Field, 0, len(params))
	for _, tp := range params {
		fields = append(fields, &goast.Field{
			Names: []*goast.Ident{{Name: tp.Name}},
			Type:  t.convertTypeParamConstraints(tp),
		})
	}

	return &goast.FieldList{List: fields}
}

func (t *Transpiler) convertField(field *types.Field) (*goast.Field, error) {
	fieldType, err := t.convertType(field.Type)
	if err != nil {
		return nil, fmt.Errorf("converting field %q type: %w", field.Name, err)
	}

	return &goast.Field{
		Names: []*goast.Ident{{Name: component.ConvertExport(field.Name, field.Exported, false)}},
		Type:  fieldType,
	}, nil
}
