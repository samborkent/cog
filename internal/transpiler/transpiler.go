package transpiler

import (
	"errors"
	"fmt"
	goast "go/ast"
	gotoken "go/token"
	"maps"
	"path"
	"slices"
	"strings"

	"github.com/samborkent/cog/internal/ast"
	"github.com/samborkent/cog/internal/transpiler/component"
	"github.com/samborkent/cog/internal/types"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

type Transpiler struct {
	files  *ast.MergedAST
	file   *ast.File // current file being processed (for line directives)
	fileID uint16    // current file ID for context tracking
	fset   *gotoken.FileSet

	imports      map[string]*goast.ImportSpec // Key: import name
	goModulePath string                       // Go module path for resolving cog import paths

	symbols        *SymbolTable
	dynDefaults    map[string]ast.Expr // Default expressions for dynamic variables
	inFunc         bool
	inMethod       bool            // set when transpiling a method body
	usesDyn        bool            // set during body conversion when a dyn var is read or written
	needsContext   map[uint16]bool // per-file tracking of context requirement by file ID
	ifLabelCounter uint32

	typeCache      map[types.Type]goast.Expr
	dynComments    map[string]string   // dyn field name → trailing comment text
	skipComments   map[uint64]struct{} // hashes of comments consumed by dyn fields
	lastSourceLine uint32              // tracks the source line of the previous statement

	titleCaser cases.Caser
}

type TranspilerOption func(*Transpiler)

func NewTranspiler(files *ast.MergedAST, opts ...TranspilerOption) *Transpiler {
	return newTranspilerWithOptions("", files, opts...)
}

func NewTranspilerWithModule(goModulePath string, files *ast.MergedAST, opts ...TranspilerOption) *Transpiler {
	return newTranspilerWithOptions(goModulePath, files, opts...)
}

func newTranspilerWithOptions(goModulePath string, files *ast.MergedAST, opts ...TranspilerOption) *Transpiler {
	t := &Transpiler{
		files:        files,
		fset:         gotoken.NewFileSet(),
		goModulePath: goModulePath,
		symbols:      NewSymbolTable(),
		dynDefaults:  make(map[string]ast.Expr),
		needsContext: make(map[uint16]bool),
		typeCache:    make(map[types.Type]goast.Expr),
		dynComments:  make(map[string]string),
		skipComments: make(map[uint64]struct{}),
		titleCaser:   cases.Title(language.English),
	}

	for _, opt := range opts {
		opt(t)
	}

	return t
}

func (t *Transpiler) Transpile() (*goast.File, error) {
	defer t.files.Free()

	if err := t.predeclareGlobals(); err != nil {
		return nil, err
	}

	// Set current file.
	t.file = t.files.Nodes[0][1].(*ast.File)

	// Count total statements across all files.
	totalStmts := len(t.file.Statements)

	gofile := &goast.File{
		Name:  goast.NewIdent(t.file.Package.Identifier.Name),
		Decls: make([]goast.Decl, 0, totalStmts),
	}
	errs := make([]error, 0)

	t.imports = make(map[string]*goast.ImportSpec)

	// Generate dynamic variable struct types if needed.
	dynDecls := t.buildDynDecls()
	gofile.Decls = append(gofile.Decls, dynDecls...)

	for _, stmt := range t.file.Statements {
		// Skip comments already consumed by dyn field annotations.
		if comment, ok := t.files.Nodes[0][stmt].(*ast.Comment); ok {
			if _, skip := t.skipComments[comment.Hash()]; skip {
				continue
			}
		}

		switch s := t.files.Nodes[0][stmt].(type) {
		case *ast.GoImport:
			for _, imprt := range s.Imports {
				t.addStdLibImport(imprt.Name)
			}
		case *ast.Import:
			t.addCogImports(s)
		default:
			gonodes, err := t.convertDecl(s)
			if err != nil {
				errs = append(errs, fmt.Errorf("\t%s: %w", s.String(), err))
				continue
			}

			if _, isComment := s.(*ast.Comment); !isComment {
				t.attachLineDecl(gonodes, s)
			}

			ln, _ := s.Pos()
			t.lastSourceLine = ln

			gofile.Decls = append(gofile.Decls, gonodes...)
		}
	}

	t.finalizeImports(gofile)

	if err := errors.Join(errs...); err != nil {
		return nil, fmt.Errorf("transpiler errors:\n%w", err)
	}

	return gofile, nil
}

// TranspileFiles produces one *goast.File per input *ast.File.
// Shared constructs (dyn struct types) are emitted in the first file.
// Each output file gets its own import declarations.
// currentFileNeedsContext returns true if the current file being processed needs context support
func (t *Transpiler) currentFileNeedsContext() bool {
	if t.file == nil {
		return false
	}

	// Find the file ID for the current file
	var fileID uint16

	for id, f := range t.files.Nodes {
		if f[1] == t.file {
			fileID = uint16(id)
			break
		}
	}

	return t.needsContext[fileID]
}

func (t *Transpiler) TranspileFiles() ([]*goast.File, error) {
	defer t.files.Free()

	if err := t.predeclareGlobals(); err != nil {
		return nil, err
	}

	errs := make([]error, 0)
	gofiles := make([]*goast.File, len(t.files.Nodes))

	for i := range t.files.Nodes {
		t.file = t.files.Nodes[i][1].(*ast.File)
		t.fileID = uint16(i)
		t.imports = make(map[string]*goast.ImportSpec)
		t.lastSourceLine = 0

		gofile := &goast.File{
			Name:  goast.NewIdent(t.file.Package.Identifier.Name),
			Decls: make([]goast.Decl, 0, len(t.file.Statements)),
		}

		if t.file.ContainsMain {
			gofile.Decls = append(gofile.Decls, t.setMemoryLimit())
		}

		// Emit dyn struct types in the first file only.
		if i == 0 {
			gofile.Decls = append(gofile.Decls, t.buildDynDecls()...)
		}

		for _, stmt := range t.file.Statements {
			if comment, ok := t.files.Nodes[i][stmt].(*ast.Comment); ok {
				if _, skip := t.skipComments[comment.Hash()]; skip {
					continue
				}
			}

			switch s := t.files.Nodes[i][stmt].(type) {
			case *ast.GoImport:
				for _, imprt := range s.Imports {
					t.addStdLibImport(imprt.Name)
				}
			case *ast.Import:
				t.addCogImports(s)
			default:
				gonodes, err := t.convertDecl(s)
				if err != nil {
					errs = append(errs, fmt.Errorf("\t%s: %w", s.String(), err))
					continue
				}

				if _, isComment := s.(*ast.Comment); !isComment {
					t.attachLineDecl(gonodes, s)
				}

				ln, _ := s.Pos()
				t.lastSourceLine = ln

				gofile.Decls = append(gofile.Decls, gonodes...)
			}
		}

		t.finalizeImports(gofile)
		gofiles[i] = gofile
	}

	if err := errors.Join(errs...); err != nil {
		return nil, fmt.Errorf("transpiler errors:\n%w", err)
	}

	return gofiles, nil
}

// TranspileScript transpiles a script file (.cogs) into a single Go file.
// All statements are placed inside a func main() body. Type aliases and
// enum declarations are emitted as top-level declarations.
func (t *Transpiler) TranspileScript() (*goast.File, error) {
	defer t.files.Free()

	t.imports = make(map[string]*goast.ImportSpec)
	t.lastSourceLine = 0

	// Set current file.
	t.file = t.files.Nodes[0][1].(*ast.File)

	gofile := &goast.File{
		Name:  goast.NewIdent("main"),
		Decls: make([]goast.Decl, 0),
	}
	errs := make([]error, 0)

	// Collect statements into the main body, separating top-level type
	// declarations and imports which stay at file level.
	mainBody := make([]goast.Stmt, 0)

	for _, stmt := range t.file.Statements {
		switch s := t.files.Nodes[0][stmt].(type) {
		case *ast.GoImport:
			for _, imprt := range s.Imports {
				t.addStdLibImport(imprt.Name)
			}
		case *ast.Import:
			t.addCogImports(s)
		case *ast.Method, *ast.Type:
			gonodes, err := t.convertDecl(s)
			if err != nil {
				errs = append(errs, fmt.Errorf("\t%s: %w", s.String(), err))
				continue
			}

			gofile.Decls = append(gofile.Decls, gonodes...)
		default:
			goStmts, err := t.convertStmt(t.Node(stmt))
			if err != nil {
				errs = append(errs, fmt.Errorf("\t%s: %w", s.String(), err))
				continue
			}

			mainBody = append(mainBody, goStmts...)
		}
	}

	t.addStdLibImport("context")
	t.addStdLibImport("os/signal")
	t.addStdLibImport("syscall")

	// Only pass existing ctx to Signal when dyn init creates one for proc propagation.
	passCtx := t.usesDyn && t.currentFileNeedsContext()

	ctxIdent := t.symbols.Define("ctx")

	if err := t.symbols.MarkUsed("ctx"); err != nil {
		return nil, fmt.Errorf("marking ctx used: %w", err)
	}

	// Wrap everything in func main().
	adjustedBody := append([]goast.Stmt{component.AdaptiveGC(ctxIdent)}, mainBody...)

	mainFunc := &goast.FuncDecl{
		Name: &goast.Ident{Name: "main"},
		Type: &goast.FuncType{Params: &goast.FieldList{}},
		Body: &goast.BlockStmt{
			List: append(component.Signal(ctxIdent, passCtx), adjustedBody...),
		},
	}

	t.injectArena(mainFunc.Body)

	gofile.Decls = append(gofile.Decls, t.setMemoryLimit(), mainFunc)

	t.finalizeImports(gofile)

	if err := errors.Join(errs...); err != nil {
		return nil, fmt.Errorf("transpiler errors:\n%w", err)
	}

	return gofile, nil
}

// predeclareGlobals scans all files to populate symbols, dynDefaults, and needsContext.
func (t *Transpiler) predeclareGlobals() error {
	errs := make([]error, 0)

	for id := range t.files.Nodes {
		f := t.files.Nodes[id][1].(*ast.File)

		for i, stmt := range f.Statements {
			switch s := t.files.Nodes[id][stmt].(type) {
			case *ast.Declaration:
				name := component.ConvertExport(s.Assignment.Identifier.Name, s.Assignment.Identifier.Exported, s.Assignment.Identifier.Global)

				if s.Assignment.Identifier.Qualifier == ast.QualifierDynamic {
					if err := t.symbols.DefineDynamic(s.Assignment.Identifier); err != nil {
						errs = append(errs, fmt.Errorf("defining dynamic variable %q: %w", name, err))
						continue
					}

					if s.Assignment.Expr != ast.ZeroExprIndex {
						t.dynDefaults[name] = t.files.Exprs[id][s.Assignment.Expr]
					}

					if i+1 < len(f.Statements) {
						if comment, ok := t.files.Nodes[id][f.Statements[i+1]].(*ast.Comment); ok {
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

					// Exported symbols must keep their Go name even when unused
					// within the package, since other packages may reference them.
					if s.Assignment.Identifier.Exported {
						_ = t.symbols.MarkUsed(name)
					}
				}

				if s.Assignment.Identifier.Name != "main" && s.Assignment.Expr != ast.ZeroExprIndex {
					if procType, ok := t.files.Exprs[id][s.Assignment.Expr].Type().(*types.Procedure); ok && !procType.Function {
						// Find the file ID for this file
						for id, file := range t.files.Nodes {
							if file[1] == f {
								t.needsContext[uint16(id)] = true
								break
							}
						}
					}
				}
			case *ast.Method:
				decl := t.files.Nodes[id][s.Declaration].(*ast.Declaration)
				assignType := t.files.Exprs[id][decl.Assignment.Expr].Type()

				procType, ok := assignType.(*types.Procedure)
				if ok && !procType.Function {
					// Find the file ID for this file
					for id, file := range t.files.Nodes {
						if file[1] == f {
							t.needsContext[uint16(id)] = true
							break
						}
					}
				}
			case *ast.Type:
				t.symbols.Define(component.ConvertExport(s.Identifier.Name, s.Identifier.Exported, s.Identifier.Global))
			}
		}
	}

	if err := errors.Join(errs...); err != nil {
		return fmt.Errorf("transpiler errors:\n%w", err)
	}

	return nil
}

// addCogImports registers cog import paths as Go imports with the proper module prefix.
func (t *Transpiler) addCogImports(node *ast.Import) {
	for _, imprt := range node.Imports {
		importPath := imprt.Name

		goPath := importPath
		if t.goModulePath != "" {
			goPath = t.goModulePath + "/" + importPath
		}

		pkgName := importPath
		if i := len(importPath) - 1; i >= 0 {
			for i > 0 && importPath[i-1] != '/' {
				i--
			}

			pkgName = importPath[i:]
		}

		t.imports[pkgName] = &goast.ImportSpec{
			Path: &goast.BasicLit{
				Kind:  gotoken.STRING,
				Value: `"` + goPath + `"`,
			},
		}
	}
}

// buildDynDecls generates the cogDynKey and cogDyn struct type declarations.
func (t *Transpiler) buildDynDecls() []goast.Decl {
	if len(t.symbols.dynamics) == 0 {
		return nil
	}

	fields := make([]*goast.Field, 0, len(t.symbols.dynamics))

	for name, ident := range t.symbols.dynamics {
		fieldType, err := t.convertType(ident.ValueType)
		if err != nil {
			panic(fmt.Sprintf("buildDynDecls: converting type for %q: %v", name, err))
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

	return []goast.Decl{
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
	}
}

// finalizeImports collects accumulated imports and prepends them to the file.
func (t *Transpiler) finalizeImports(gofile *goast.File) {
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
			Name: &goast.Ident{Name: goStdLibAlias(name)},
			Path: &goast.BasicLit{
				Kind:  gotoken.STRING,
				Value: `"` + name + `"`,
			},
		}
	}
}

func (t *Transpiler) addGoImport(name string) {
	_, ok := t.imports[name]
	if !ok {
		t.imports[name] = &goast.ImportSpec{
			Name: &goast.Ident{Name: "_" + strings.ReplaceAll(path.Base(name), "-", "")},
			Path: &goast.BasicLit{
				Kind:  gotoken.STRING,
				Value: `"` + name + `"`,
			},
		}
	}
}

// goStdLibAlias returns the aliased import name for a Go standard library package.
// For example, "strings" becomes "go_strings" and "path/filepath" becomes "go_filepath".
func goStdLibAlias(importPath string) string {
	return "go_" + path.Base(importPath)
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

func (t *Transpiler) setMemoryLimit() *goast.FuncDecl {
	t.addStdLibImport("runtime/debug")
	t.addGoImport("github.com/pbnjay/memory")
	t.addGoImport("github.com/samborkent/adaptive-gc")

	return component.SetMemoryLimit()
}

func (t *Transpiler) Node(i ast.NodeIndex) ast.Node {
	return t.files.Nodes[t.fileID][i]
}

func (t *Transpiler) Expr(i ast.ExprIndex) ast.Expr {
	return t.files.Exprs[t.fileID][i]
}
