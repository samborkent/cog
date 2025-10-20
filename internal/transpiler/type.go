package transpiler

import (
	"fmt"
	goast "go/ast"
	gotypes "go/types"
	"strconv"

	"github.com/samborkent/cog/internal/types"
)

func convertType(t types.Type) goast.Expr {
	if alias, ok := t.(*types.Alias); ok {
		if alias.Underlying().Kind() == types.EnumKind {
			return &goast.Ident{Name: convertExport(alias.Name, alias.Exported) + "Enum"}
		}

		return &goast.Ident{Name: convertExport(alias.Name, alias.Exported)}
	}

	switch t.Kind() {
	case types.ASCII:
		return &goast.SelectorExpr{
			X:   &goast.Ident{Name: "cog"},
			Sel: &goast.Ident{Name: "ASCII"},
		}
	case types.Bool:
		return &goast.Ident{Name: gotypes.TypeString(gotypes.Typ[gotypes.Bool], nil)}
	case types.Complex64:
		return &goast.Ident{Name: gotypes.TypeString(gotypes.Typ[gotypes.Complex64], nil)}
	case types.Complex128:
		return &goast.Ident{Name: gotypes.TypeString(gotypes.Typ[gotypes.Complex128], nil)}
	case types.Context:
		return &goast.SelectorExpr{
			X:   &goast.Ident{Name: "context"},
			Sel: &goast.Ident{Name: "Context"},
		}
	case types.Float32:
		return &goast.Ident{Name: gotypes.TypeString(gotypes.Typ[gotypes.Float32], nil)}
	case types.Float64:
		return &goast.Ident{Name: gotypes.TypeString(gotypes.Typ[gotypes.Float64], nil)}
	case types.Int8:
		return &goast.Ident{Name: gotypes.TypeString(gotypes.Typ[gotypes.Int8], nil)}
	case types.Int16:
		return &goast.Ident{Name: gotypes.TypeString(gotypes.Typ[gotypes.Int16], nil)}
	case types.Int32:
		return &goast.Ident{Name: gotypes.TypeString(gotypes.Typ[gotypes.Int32], nil)}
	case types.Int64:
		return &goast.Ident{Name: gotypes.TypeString(gotypes.Typ[gotypes.Int64], nil)}
	case types.OptionKind:
		optionType, ok := t.(*types.Option)
		if !ok {
			panic("unable to assert option type")
		}

		valueType := convertType(optionType.Value)

		return &goast.IndexExpr{
			X: &goast.SelectorExpr{
				X:   &goast.Ident{Name: "cog"},
				Sel: &goast.Ident{Name: "Option"},
			},
			Index: valueType,
		}
	case types.Uint8:
		return &goast.Ident{Name: gotypes.TypeString(gotypes.Typ[gotypes.Uint8], nil)}
	case types.Uint16:
		return &goast.Ident{Name: gotypes.TypeString(gotypes.Typ[gotypes.Uint16], nil)}
	case types.Uint32:
		return &goast.Ident{Name: gotypes.TypeString(gotypes.Typ[gotypes.Uint32], nil)}
	case types.Uint64:
		return &goast.Ident{Name: gotypes.TypeString(gotypes.Typ[gotypes.Uint64], nil)}
	case types.SetKind:
		setType, ok := t.(*types.Set)
		if !ok {
			panic("unable to assert set type")
		}

		return &goast.IndexExpr{
			X: &goast.SelectorExpr{
				X:   &goast.Ident{Name: "cog"},
				Sel: &goast.Ident{Name: "Set"},
			},
			Index: convertType(setType.Element),
		}
	case types.StructKind:
		structType, ok := t.(*types.Struct)
		if !ok {
			panic("unable to assert struct type")
		}

		fields := make([]*goast.Field, len(structType.Fields))

		for i := range structType.Fields {
			fields[i] = convertField(structType.Fields[i])
		}

		return &goast.StructType{
			Fields: &goast.FieldList{
				List: fields,
			},
		}
	case types.TupleKind:
		tupleType, ok := t.(*types.Tuple)
		if !ok {
			panic("unable to assert tuple type")
		}

		fields := make([]*goast.Field, 0, len(tupleType.Types))

		for i := range tupleType.Types {
			fields = append(fields, &goast.Field{
				Names: []*goast.Ident{{Name: convertExport("t"+strconv.Itoa(i), tupleType.Exported)}},
				Type:  convertType(tupleType.Types[i]),
			})
		}

		return &goast.StructType{
			Fields: &goast.FieldList{
				List: fields,
			},
		}
	case types.UnionKind:
		unionType, ok := t.(*types.Union)
		if !ok {
			panic("unable to assert union type")
		}

		return &goast.StructType{
			Fields: &goast.FieldList{
				List: []*goast.Field{
					{
						Names: []*goast.Ident{{Name: "Either"}},
						Type:  convertType(unionType.Either),
					},
					{
						Names: []*goast.Ident{{Name: "Or"}},
						Type:  convertType(unionType.Or),
					},
					{
						Names: []*goast.Ident{{Name: "Tag"}},
						Type:  &goast.Ident{Name: "bool"},
					},
				},
			},
		}
	case types.UTF8:
		return &goast.SelectorExpr{
			X:   &goast.Ident{Name: "cog"},
			Sel: &goast.Ident{Name: "UTF8"},
		}
	default:
		panic(fmt.Sprintf("unknown type %q", t))
	}
}
