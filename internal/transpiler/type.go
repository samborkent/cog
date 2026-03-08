package transpiler

import (
	"errors"
	"fmt"
	goast "go/ast"
	gotypes "go/types"
	"strconv"

	"github.com/samborkent/cog/internal/ast"
	"github.com/samborkent/cog/internal/transpiler/component"
	"github.com/samborkent/cog/internal/types"
)

func (t *Transpiler) convertType(typ types.Type) (goast.Expr, error) {
	// Try to retrieve type expression from cache.
	expr, ok := t.typeCache[typ.String()]
	if ok {
		return expr, nil
	}

	if alias, ok := typ.(*types.Alias); ok {
		if alias.Underlying().Kind() == types.EnumKind {
			expr = &goast.Ident{Name: convertExport(alias.Name, alias.Exported) + "Enum"}
		} else {
			expr = &goast.Ident{Name: convertExport(alias.Name, alias.Exported)}
		}

		t.typeCache[typ.String()] = expr
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
	case types.Complex64:
		expr = &goast.Ident{Name: gotypes.TypeString(gotypes.Typ[gotypes.Complex64], nil)}
	case types.Complex128:
		expr = &goast.Ident{Name: gotypes.TypeString(gotypes.Typ[gotypes.Complex128], nil)}
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
	case types.MapKind:
		mapType, ok := typ.(*types.Map)
		if !ok {
			return nil, errors.New("unable to assert map type")
		}

		keyType, err := t.convertType(mapType.Key)
		if err != nil {
			return nil, fmt.Errorf("converting map key type: %w", err)
		}

		valType, err := t.convertType(mapType.Value)
		if err != nil {
			return nil, fmt.Errorf("converting map value type: %w", err)
		}

		expr = &goast.MapType{
			Key:   keyType,
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

		if !procType.Function {
			// All procedures take context.
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
	case types.Uint8:
		expr = &goast.Ident{Name: gotypes.TypeString(gotypes.Typ[gotypes.Uint8], nil)}
	case types.Uint16:
		expr = &goast.Ident{Name: gotypes.TypeString(gotypes.Typ[gotypes.Uint16], nil)}
	case types.Uint32:
		expr = &goast.Ident{Name: gotypes.TypeString(gotypes.Typ[gotypes.Uint32], nil)}
	case types.Uint64:
		expr = &goast.Ident{Name: gotypes.TypeString(gotypes.Typ[gotypes.Uint64], nil)}
	case types.SetKind:
		setType, ok := typ.(*types.Set)
		if !ok {
			return nil, errors.New("unable to assert set type")
		}

		elemType, err := t.convertType(setType.Element)
		if err != nil {
			return nil, fmt.Errorf("converting set element type: %w", err)
		}

		t.addCogImport()

		expr = &goast.IndexExpr{
			X: &goast.SelectorExpr{
				X:   &goast.Ident{Name: "cog"},
				Sel: &goast.Ident{Name: "Set"},
			},
			Index: elemType,
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
				Names: []*goast.Ident{{Name: convertExport("t"+strconv.Itoa(i), tupleType.Exported)}},
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

		eitherType, err := t.convertType(unionType.Either)
		if err != nil {
			return nil, fmt.Errorf("converting union either type: %w", err)
		}

		orType, err := t.convertType(unionType.Or)
		if err != nil {
			return nil, fmt.Errorf("converting union or type: %w", err)
		}

		expr = &goast.StructType{
			Fields: &goast.FieldList{
				List: []*goast.Field{
					{
						Names: []*goast.Ident{{Name: "Either"}},
						Type:  eitherType,
					},
					{
						Names: []*goast.Ident{{Name: "Or"}},
						Type:  orType,
					},
					{
						Names: []*goast.Ident{{Name: "Tag"}},
						Type:  &goast.Ident{Name: "bool"},
					},
				},
			},
		}
	case types.UTF8:
		expr = &goast.Ident{Name: gotypes.TypeString(gotypes.Typ[gotypes.String], nil)}
	default:
		return nil, fmt.Errorf("unknown type %q", typ)
	}

	t.typeCache[typ.String()] = expr
	return expr, nil
}

func (t *Transpiler) convertField(field *types.Field) (*goast.Field, error) {
	fieldType, err := t.convertType(field.Type)
	if err != nil {
		return nil, fmt.Errorf("converting field %q type: %w", field.Name, err)
	}

	return &goast.Field{
		Names: []*goast.Ident{{Name: convertExport(field.Name, field.Exported)}},
		Type:  fieldType,
	}, nil
}
