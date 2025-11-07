package transpiler

import (
	"errors"
	"fmt"
	goast "go/ast"
	gotoken "go/token"
	"maps"
	"slices"
	"strings"
	"unicode"

	"github.com/samborkent/cog/internal/ast"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

var titleCaser = cases.Title(language.English)

type Transpiler struct {
	file *ast.File
	fset *gotoken.FileSet

	nodes   map[uint64]ast.Node
	imports map[string]*goast.ImportSpec // Key: import name

	symbols *SymbolTable
	inFunc  bool
}

// attachLineDirective adds a //line directive comment to the first declaration in decls
// so that compiler errors refer back to the originating Cog source location.
func (t *Transpiler) attachLineDirective(decls []goast.Decl, node ast.Node) {
	if len(decls) == 0 || node == nil {
		return
	}

	ln, _ := node.Pos()
	cogFile := t.file.Package.Identifier.Name + ".cog"

	c := &goast.Comment{Text: fmt.Sprintf("//line %s:%d", cogFile, ln)}

	// Attach to the first declaration where a Doc comment is applicable.
	for i := range decls {
		switch d := decls[i].(type) {
		case *goast.GenDecl:
			if d.Doc == nil {
				d.Doc = &goast.CommentGroup{List: []*goast.Comment{c}}
			} else {
				// Prepend so the line directive appears immediately before decl.
				d.Doc.List = append([]*goast.Comment{c}, d.Doc.List...)
			}
			return
		case *goast.FuncDecl:
			if d.Doc == nil {
				d.Doc = &goast.CommentGroup{List: []*goast.Comment{c}}
			} else {
				d.Doc.List = append([]*goast.Comment{c}, d.Doc.List...)
			}
			return
		}
	}

	// If no suitable decl found, as a fallback add a GenDecl with the comment.
	decls[0] = &goast.GenDecl{
		Tok: gotoken.IMPORT,
		Doc: &goast.CommentGroup{List: []*goast.Comment{c}},
	}
}

func NewTranspiler(f *ast.File) *Transpiler {
	nodes := make(map[uint64]ast.Node)

	nodes[f.Hash()] = f
	nodes[f.Package.Hash()] = f.Package

	for _, stmt := range f.Statements {
		nodes[stmt.Hash()] = stmt
	}

	return &Transpiler{
		file:    f,
		fset:    gotoken.NewFileSet(),
		nodes:   nodes,
		symbols: NewSymbolTable(),
	}
}

func (t *Transpiler) Transpile() (*goast.File, error) {
	gofile := &goast.File{
		Name:  goast.NewIdent(t.file.Package.Identifier.Name),
		Decls: make([]goast.Decl, 0, len(t.file.Statements)),
	}
	errs := make([]error, 0)

	// Predeclare globals
	for _, stmt := range t.file.Statements {
		switch s := stmt.(type) {
		case *ast.Declaration:
			name := convertExport(s.Assignment.Identifier.Name, s.Assignment.Identifier.Exported)

			if s.Assignment.Identifier.Qualifier == ast.QualifierDynamic {
				t.symbols.DefineDynamic(s.Assignment.Identifier)
			} else {
				t.symbols.Define(name)
			}
		case *ast.Type:
			t.symbols.Define(convertExport(s.Identifier.Name, s.Identifier.Exported))
		}
	}

	t.imports = make(map[string]*goast.ImportSpec)

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

			// Attach a //line directive mapping generated decls back to the original Cog node.
			t.attachLineDirective(gonodes, s)

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

func (t *Transpiler) addCogImport() {
	_, ok := t.imports["cog"]
	if !ok {
		t.imports["cog"] = &goast.ImportSpec{
			Name: &goast.Ident{Name: "cog"},
			Path: &goast.BasicLit{
				Kind:  gotoken.STRING,
				Value: `"github.com/samborkent/cog"`,
			},
		}
	}
}
