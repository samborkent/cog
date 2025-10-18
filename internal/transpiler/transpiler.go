package transpiler

import (
	"errors"
	"fmt"
	goast "go/ast"
	gotoken "go/token"
	gotypes "go/types"
	"maps"
	"slices"
	"strconv"
	"strings"
	"unicode"

	"github.com/samborkent/cog/internal/ast"
	"github.com/samborkent/cog/internal/types"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

var titleCaser = cases.Title(language.English)

type Transpiler struct {
	file *ast.File
	fset *gotoken.FileSet

	nodes   map[uint64]ast.Node
	imports map[string]*goast.ImportSpec // Key: import name
}

func NewTranspiler(f *ast.File) *Transpiler {
	nodes := make(map[uint64]ast.Node)

	nodes[f.Hash()] = f
	nodes[f.Package.Hash()] = f.Package

	for _, stmt := range f.Statements {
		nodes[stmt.Hash()] = stmt
	}

	return &Transpiler{
		file:  f,
		fset:  gotoken.NewFileSet(),
		nodes: nodes,
	}
}

func (t *Transpiler) Transpile() (*goast.File, error) {
	gofile := &goast.File{
		Name:  goast.NewIdent(t.file.Package.Identifier.Name),
		Decls: make([]goast.Decl, 0, len(t.file.Statements)),
	}
	errs := make([]error, 0)

	// Predeclare constants
	for _, stmt := range t.file.Statements {
		switch s := stmt.(type) {
		case *ast.Declaration:
			if !s.Constant {
				continue
			}

			name := convertExport(s.Assignment.Identifier.Name, s.Assignment.Identifier.Exported)

			// Create a copy.
			ident := *s.Assignment.Identifier
			ident.Name = "_" // Start off as unused.

			identifiers[name] = ident.Go()
		}
	}

	t.imports = make(map[string]*goast.ImportSpec)

	// Base import
	t.imports["cog"] = &goast.ImportSpec{
		Name: &goast.Ident{Name: "cog"},
		Path: &goast.BasicLit{
			Kind:  gotoken.STRING,
			Value: `"github.com/samborkent/cog"`,
		},
	}

	for _, stmt := range t.file.Statements {
		switch s := stmt.(type) {
		case *ast.GoImport:
			for _, imprt := range s.Imports {
				t.imports[imprt.Name] = &goast.ImportSpec{
					Path: &goast.BasicLit{
						Kind:  gotoken.STRING,
						Value: `"` + imprt.Name + `"`,
					},
				}
			}
		default:
			gonodes, err := t.convertDecl(s)
			if err != nil {
				errs = append(errs, fmt.Errorf("\t%s: %w", s.String(), err))
				continue
			}

			gofile.Decls = append(gofile.Decls, gonodes...)
		}
	}

	gofile.Imports = slices.Collect(maps.Values(t.imports))

	specs := make([]goast.Spec, len(gofile.Imports))
	for i := range gofile.Imports {
		specs[i] = gofile.Imports[i]
	}

	gofile.Decls = append([]goast.Decl{&goast.GenDecl{
		Tok:   gotoken.IMPORT,
		Specs: specs,
	}}, gofile.Decls...)

	if err := errors.Join(errs...); err != nil {
		return nil, fmt.Errorf("transpiler errors:\n%w", err)
	}

	return gofile, nil
}

func convertType(t types.Type) goast.Expr {
	if alias, ok := t.(*types.Alias); ok {
		if alias.Derived.Underlying().Kind() == types.EnumKind {
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
						Names: []*goast.Ident{{Name: convertExport("either", unionType.Exported)}},
						Type:  &goast.StarExpr{X: convertType(unionType.Either)},
					},
					{
						Names: []*goast.Ident{{Name: convertExport("or", unionType.Exported)}},
						Type:  &goast.StarExpr{X: convertType(unionType.Or)},
					},
					{
						Names: []*goast.Ident{{Name: convertExport("tag", unionType.Exported)}},
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
		// Assume user defined type.
		if alias, ok := t.(*types.Alias); ok {
			return &goast.Ident{Name: convertExport(alias.Name, alias.Exported)}
		}

		panic(fmt.Sprintf("unknown type %q", t))
	}
}

func convertExport(ident string, exported bool) string {
	r := rune(ident[0])
	str := string(r)

	if exported {
		// If exported, ensure first letter is uppercase.
		str = strings.ToUpper(str)
	} else if unicode.IsUpper(r) {
		// If not exported, but first letter is uppercase, prefix it with underscore.
		str = "_" + str
	}

	if len(ident) > 1 {
		str += ident[1:]
	}

	return str
}

func convertField(field *types.Field) *goast.Field {
	return &goast.Field{
		Names: []*goast.Ident{{Name: convertExport(field.Name, field.Exported)}},
		Type:  convertType(field.Type),
	}
}
