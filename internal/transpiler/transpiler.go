package transpiler

import (
	"bytes"
	"errors"
	"fmt"
	goast "go/ast"
	goprinter "go/printer"
	gotoken "go/token"
	"maps"
	"regexp"
	"slices"
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

	symbols *SymbolTable
	// original source filename, used when emitting //line directives
	srcName string
}

func NewTranspiler(f *ast.File, srcName string) *Transpiler {
	nodes := make(map[uint64]ast.Node)

	nodes[f.Hash()] = f
	nodes[f.Package.Hash()] = f.Package

	// Collect all nodes (including nested statements) so we can map markers
	// inserted during conversion back to their original source positions.
	var collectStmt func(ast.Statement)

	collectStmt = func(s ast.Statement) {
		nodes[s.Hash()] = s

		switch n := s.(type) {
		case *ast.Block:
			for _, st := range n.Statements {
				collectStmt(st)
			}
		case *ast.IfStatement:
			if n.Consequence != nil {
				collectStmt(n.Consequence)
			}
			if n.Alternative != nil {
				collectStmt(n.Alternative)
			}
		case *ast.Switch:
			for _, c := range n.Cases {
				nodes[c.Hash()] = c
				for _, st := range c.Body {
					collectStmt(st)
				}
			}
			if n.Default != nil {
				nodes[n.Default.Hash()] = n.Default
				for _, st := range n.Default.Body {
					collectStmt(st)
				}
			}
		}
	}

	for _, stmt := range f.Statements {
		collectStmt(stmt)
	}

	// Also traverse expressions on top-level declarations to find any
	// procedure literals (function bodies) and collect their statements.
	var collectExpr func(ast.Expression)

	collectExpr = func(e ast.Expression) {
		if e == nil {
			return
		}

		switch ex := e.(type) {
		case *ast.ProcedureLiteral:
			if ex.Body != nil {
				collectStmt(ex.Body)
			}
		}
	}

	for _, stmt := range f.Statements {
		if d, ok := stmt.(*ast.Declaration); ok {
			if d.Assignment != nil && d.Assignment.Expression != nil {
				collectExpr(d.Assignment.Expression)
			}
		}
	}

	return &Transpiler{
		file:    f,
		fset:    gotoken.NewFileSet(),
		nodes:   nodes,
		symbols: NewSymbolTable(),
		srcName: srcName,
	}
}

func (t *Transpiler) Transpile() (string, error) {
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
		return "", fmt.Errorf("transpiler errors:\n%w", err)
	}

	// Print AST to buffer and post-process markers into //line directives.
	var buf bytes.Buffer
	if err := goprinter.Fprint(&buf, t.fset, gofile); err != nil {
		return "", fmt.Errorf("printing output: %w", err)
	}

	src := buf.String()

	// Build mapping from node hash -> original position
	hashMap := map[string]uint32{}
	for h, node := range t.nodes {
		ln, _ := node.Pos()
		hashMap[fmt.Sprintf("%d", h)] = ln
	}

	// Replace marker assignments with //line directives. Markers look like
	// `_ = "__COG_LINE_<hash>__"` possibly indented.
	re := regexp.MustCompile(`(?m)^[ \t]*_ = "__COG_LINE_(\d+)__"[ \t]*\n?`)
	fileForLine := t.srcName
	if fileForLine == "" {
		fileForLine = "input"
	}

	src = re.ReplaceAllStringFunc(src, func(m string) string {
		sub := re.FindStringSubmatch(m)
		if len(sub) < 2 {
			return ""
		}

		h := sub[1]
		ln, ok := hashMap[h]
		if !ok {
			// unknown hash, remove marker
			return ""
		}

		return fmt.Sprintf("//line %s:%d\n", fileForLine, ln)
	})

	return src, nil
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

func (t *Transpiler) convertField(field *types.Field) *goast.Field {
	return &goast.Field{
		Names: []*goast.Ident{{Name: convertExport(field.Name, field.Exported)}},
		Type:  t.convertType(field.Type),
	}
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
