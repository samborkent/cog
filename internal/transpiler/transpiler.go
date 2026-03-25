package transpiler

import (
	"errors"
	"fmt"
	goast "go/ast"
	gotoken "go/token"
	"maps"
	"slices"

	"github.com/samborkent/cog/internal/ast"
	"github.com/samborkent/cog/internal/transpiler/component"
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

	symbols        *SymbolTable
	dynDefaults    map[string]ast.Expression // Default expressions for dynamic variables
	inFunc         bool
	usesDyn        bool // set during body conversion when a dyn var is read or written
	needsContext   bool
	ifLabelCounter uint32

	typeCache    map[types.Type]goast.Expr
	dynComments  map[string]string   // dyn field name → trailing comment text
	skipComments map[uint64]struct{} // hashes of comments consumed by dyn fields
}

func NewTranspiler(f *ast.File) *Transpiler {
	nodes := make(map[uint64]ast.Node)

	nodes[f.Hash()] = f
	nodes[f.Package.Hash()] = f.Package

	for _, stmt := range f.Statements {
		nodes[stmt.Hash()] = stmt
	}

	return &Transpiler{
		file:         f,
		fset:         gotoken.NewFileSet(),
		nodes:        nodes,
		symbols:      NewSymbolTable(),
		dynDefaults:  make(map[string]ast.Expression),
		typeCache:    make(map[types.Type]goast.Expr),
		dynComments:  make(map[string]string),
		skipComments: make(map[uint64]struct{}),
	}
}

func (t *Transpiler) Transpile() (*goast.File, error) {
	gofile := &goast.File{
		Name:  goast.NewIdent(t.file.Package.Identifier.Name),
		Decls: make([]goast.Decl, 0, len(t.file.Statements)),
	}
	errs := make([]error, 0)

	// Predeclare globals and determine whether context is needed.
	for i, stmt := range t.file.Statements {
		switch s := stmt.(type) {
		case *ast.Declaration:
			name := convertExport(s.Assignment.Identifier.Name, s.Assignment.Identifier.Exported)

			if s.Assignment.Identifier.Qualifier == ast.QualifierDynamic {
				if err := t.symbols.DefineDynamic(s.Assignment.Identifier); err != nil {
					errs = append(errs, fmt.Errorf("defining dynamic variable %q: %w", name, err))
					continue
				}

				if s.Assignment.Expression != nil {
					t.dynDefaults[name] = s.Assignment.Expression
				}

				// Check if the next statement is a comment on the same line.
				if i+1 < len(t.file.Statements) {
					if comment, ok := t.file.Statements[i+1].(*ast.Comment); ok {
						declLn, _ := s.Pos()
						commentLn, _ := comment.Pos()
						if commentLn == declLn {
							t.dynComments[name] = comment.Text
							t.skipComments[comment.Hash()] = struct{}{}
						}
					}
				}
			} else {
				t.symbols.Define(name)
			}

			// A non-main procedure requires context propagation.
			if s.Assignment.Identifier.Name != "main" && s.Assignment.Expression != nil {
				if procType, ok := s.Assignment.Expression.Type().(*types.Procedure); ok && !procType.Function {
					t.needsContext = true
				}
			}
		case *ast.Type:
			t.symbols.Define(convertExport(s.Identifier.Name, s.Identifier.Exported))
		}
	}

	t.imports = make(map[string]*goast.ImportSpec)

	// Generate dynamic variable struct types if needed.
	if len(t.symbols.dynamics) > 0 {
		fields := make([]*goast.Field, 0, len(t.symbols.dynamics))

		for name, ident := range t.symbols.dynamics {
			fieldType, err := t.convertType(ident.ValueType)
			if err != nil {
				return nil, fmt.Errorf("converting dynamic variable %q type: %w", name, err)
			}

			field := &goast.Field{
				Names: []*goast.Ident{{Name: name}},
				Type:  fieldType,
			}

			if commentText, ok := t.dynComments[name]; ok {
				field.Comment = &goast.CommentGroup{
					List: []*goast.Comment{{Text: commentText}},
				}
			}

			fields = append(fields, field)
		}

		gofile.Decls = append(gofile.Decls,
			&goast.GenDecl{
				Tok: gotoken.TYPE,
				Specs: []goast.Spec{
					&goast.TypeSpec{
						Name: &goast.Ident{Name: "cogDynKey"},
						Type: &goast.StructType{Fields: &goast.FieldList{}},
					},
				},
			},
			&goast.GenDecl{
				Tok: gotoken.TYPE,
				Specs: []goast.Spec{
					&goast.TypeSpec{
						Name: &goast.Ident{Name: "cogDyn"},
						Type: &goast.StructType{
							Fields: &goast.FieldList{List: fields},
						},
					},
				},
			},
		)
	}

	for _, stmt := range t.file.Statements {
		// Skip comments already consumed by dyn field annotations.
		if comment, ok := stmt.(*ast.Comment); ok {
			if _, skip := t.skipComments[comment.Hash()]; skip {
				continue
			}
		}

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

			// Attach a //line directive mapping generated decls back to the original Cog node.
			// Skip comments — they carry their own Doc and a //line pointing at a
			// block comment in the .cog source would confuse the Go compiler.
			if _, isComment := s.(*ast.Comment); !isComment {
				t.attachLineDecl(gonodes, s)
			}

			gofile.Decls = append(gofile.Decls, gonodes...)
		}
	}

	gofile.Imports = slices.Collect(maps.Values(t.imports))

	specs := make([]goast.Spec, len(gofile.Imports))
	for i := range gofile.Imports {
		specs[i] = gofile.Imports[i]
	}

	if len(gofile.Imports) > 0 {
		gofile.Decls = append([]goast.Decl{&goast.GenDecl{
			Tok:   gotoken.IMPORT,
			Specs: specs,
		}}, gofile.Decls...)
	}

	if err := errors.Join(errs...); err != nil {
		return nil, fmt.Errorf("transpiler errors:\n%w", err)
	}

	return gofile, nil
}

func (t *Transpiler) addCogImport() {
	_, ok := t.imports["cog"]
	if !ok {
		t.imports["cog"] = &goast.ImportSpec{
			Path: &goast.BasicLit{
				Kind:  gotoken.STRING,
				Value: `"github.com/samborkent/cog"`,
			},
		}
	}
}

func (t *Transpiler) addBuiltinImport() {
	_, ok := t.imports["builtin"]
	if !ok {
		t.imports["builtin"] = &goast.ImportSpec{
			Path: &goast.BasicLit{
				Kind:  gotoken.STRING,
				Value: `"github.com/samborkent/cog/builtin"`,
			},
		}
	}
}

func (t *Transpiler) addStdLibImport(name string) {
	_, ok := t.imports[name]
	if !ok {
		t.imports[name] = &goast.ImportSpec{
			Path: &goast.BasicLit{
				Kind:  gotoken.STRING,
				Value: `"` + name + `"`,
			},
		}
	}
}

func (t *Transpiler) addFloat16Import() {
	_, ok := t.imports["f16"]
	if !ok {
		t.imports["f16"] = &goast.ImportSpec{
			Name: &goast.Ident{Name: "f16"},
			Path: &goast.BasicLit{
				Kind:  gotoken.STRING,
				Value: `"github.com/x448/float16"`,
			},
		}
	}
}

// attachLineDecl adds a //line directive comment to the first declaration in decls
// so that compiler errors refer back to the originating Cog source location.
func (t *Transpiler) attachLineDecl(decls []goast.Decl, node ast.Node) {
	if t.file.Name == "" || len(decls) == 0 || node == nil {
		return
	}

	ln, _ := node.Pos()
	comment := &goast.Comment{Text: fmt.Sprintf("//line %s:%d", t.file.Name, ln)}

	// Attach to the first declaration where a Doc comment is applicable.
	for i := range decls {
		switch d := decls[i].(type) {
		case *goast.GenDecl:
			if d.Doc == nil {
				d.Doc = &goast.CommentGroup{List: []*goast.Comment{comment}}
			} else {
				// Prepend so the line directive appears immediately before decl.
				d.Doc.List = append([]*goast.Comment{comment}, d.Doc.List...)
			}

			return
		case *goast.FuncDecl:
			if d.Doc == nil {
				d.Doc = &goast.CommentGroup{List: []*goast.Comment{comment}}
			} else {
				d.Doc.List = append([]*goast.Comment{comment}, d.Doc.List...)
			}

			return
		}
	}

	// If no suitable decl found, as a fallback add a GenDecl with the comment.
	decls[0] = &goast.GenDecl{
		Tok: gotoken.IMPORT,
		Doc: &goast.CommentGroup{List: []*goast.Comment{comment}},
	}
}

func convertExport(ident string, exported bool) string {
	return component.ConvertExport(ident, exported)
}
