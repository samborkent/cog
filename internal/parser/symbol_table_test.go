package parser

import (
	"testing"

	"github.com/samborkent/cog/internal/ast"
	"github.com/samborkent/cog/internal/tokens"
	"github.com/samborkent/cog/internal/types"
)

func makeIdent(name string, vt types.Type) *ast.Identifier {
	return &ast.Identifier{
		Token:     tokens.Token{Type: tokens.Identifier, Literal: name},
		Name:      name,
		ValueType: vt,
	}
}

func TestNewSymbolTable(t *testing.T) {
	t.Parallel()
	s := NewSymbolTable()

	sym, ok := s.Resolve("_")
	if !ok {
		t.Fatal("expected blank identifier")
	}
	if sym.Scope != LocalScope {
		t.Errorf("blank scope = %v, want LocalScope", sym.Scope)
	}
}

func TestDefineAndResolve(t *testing.T) {
	t.Parallel()
	s := NewSymbolTable()

	ident := makeIdent("x", types.Basics[types.Int64])
	s.Define(ident)

	sym, ok := s.Resolve("x")
	if !ok {
		t.Fatal("expected to resolve x")
	}
	if sym.Scope != GlobalScope {
		t.Errorf("scope = %v, want GlobalScope", sym.Scope)
	}
	if sym.Type().Kind() != types.Int64 {
		t.Errorf("type = %v, want int64", sym.Type())
	}
}

func TestDefineNilType(t *testing.T) {
	t.Parallel()
	s := NewSymbolTable()

	ident := &ast.Identifier{
		Token: tokens.Token{Type: tokens.Identifier, Literal: "y"},
		Name:  "y",
	}
	s.Define(ident)

	sym, ok := s.Resolve("y")
	if !ok {
		t.Fatal("expected to resolve y")
	}
	if !types.IsNone(sym.Type()) {
		t.Errorf("expected None type, got %v", sym.Type())
	}
}

func TestResolveNotFound(t *testing.T) {
	t.Parallel()
	s := NewSymbolTable()

	_, ok := s.Resolve("nonexistent")
	if ok {
		t.Fatal("expected resolve to fail")
	}
}

func TestEnclosedSymbolTable(t *testing.T) {
	t.Parallel()
	outer := NewSymbolTable()
	outer.Define(makeIdent("a", types.Basics[types.UTF8]))

	inner := NewEnclosedSymbolTable(outer)
	inner.Define(makeIdent("b", types.Basics[types.Int64]))

	if _, ok := inner.Resolve("b"); !ok {
		t.Fatal("expected inner to resolve b")
	}

	sym, ok := inner.Resolve("a")
	if !ok {
		t.Fatal("expected inner to resolve a from outer")
	}
	if sym.Type().Kind() != types.UTF8 {
		t.Errorf("type = %v, want utf8", sym.Type())
	}

	bsym, _ := inner.Resolve("b")
	if bsym.Scope != LocalScope {
		t.Errorf("inner scope = %v, want LocalScope", bsym.Scope)
	}
}

func TestDefineGlobal(t *testing.T) {
	t.Parallel()
	s := NewSymbolTable()

	ident := makeIdent("g", types.Basics[types.Float64])
	s.DefineGlobal(ident)

	sym, ok := s.Resolve("g")
	if !ok {
		t.Fatal("expected to resolve g")
	}
	if sym.Scope != ScanScope {
		t.Errorf("scope = %v, want ScanScope", sym.Scope)
	}
}

func TestDefineAndResolveGoImport(t *testing.T) {
	t.Parallel()
	s := NewSymbolTable()

	ident := makeIdent("strings", types.None)
	s.DefineGoImport(ident)

	got, ok := s.ResolveGoImport("strings")
	if !ok {
		t.Fatal("expected to resolve go import")
	}
	if got.Name != "strings" {
		t.Errorf("name = %q, want strings", got.Name)
	}
}

func TestGoImportSharedInEnclosed(t *testing.T) {
	t.Parallel()
	outer := NewSymbolTable()
	outer.DefineGoImport(makeIdent("fmt", types.None))

	inner := NewEnclosedSymbolTable(outer)

	if _, ok := inner.ResolveGoImport("fmt"); !ok {
		t.Fatal("expected enclosed table to see outer go import")
	}
}

func TestDefineCogImportAndResolve(t *testing.T) {
	t.Parallel()
	s := NewSymbolTable()

	imp := &CogImport{
		Path:    "geom",
		Name:    "geom",
		Exports: make(map[string]Symbol),
	}
	s.DefineCogImport(imp)

	got, ok := s.ResolveCogImport("geom")
	if !ok {
		t.Fatal("expected to resolve cog import")
	}
	if got.Path != "geom" {
		t.Errorf("path = %q, want geom", got.Path)
	}
}

func TestCogImportNotFound(t *testing.T) {
	t.Parallel()
	s := NewSymbolTable()

	_, ok := s.ResolveCogImport("missing")
	if ok {
		t.Fatal("expected resolve to fail")
	}
}

func TestCogImports(t *testing.T) {
	t.Parallel()
	s := NewSymbolTable()

	s.DefineCogImport(&CogImport{Path: "a", Name: "a", Exports: make(map[string]Symbol)})
	s.DefineCogImport(&CogImport{Path: "b/c", Name: "c", Exports: make(map[string]Symbol)})

	imports := s.CogImports()
	if len(imports) != 2 {
		t.Fatalf("expected 2 imports, got %d", len(imports))
	}
	if _, ok := imports["a"]; !ok {
		t.Error("missing import a")
	}
	if _, ok := imports["c"]; !ok {
		t.Error("missing import c")
	}
}

func TestCogImportSharedInEnclosed(t *testing.T) {
	t.Parallel()
	outer := NewSymbolTable()
	outer.DefineCogImport(&CogImport{Path: "pkg", Name: "pkg", Exports: make(map[string]Symbol)})

	inner := NewEnclosedSymbolTable(outer)

	if _, ok := inner.ResolveCogImport("pkg"); !ok {
		t.Fatal("expected enclosed table to see outer cog import")
	}
}

func TestForEachGlobal(t *testing.T) {
	t.Parallel()
	outer := NewSymbolTable()
	outer.Define(makeIdent("a", types.Basics[types.Int64]))
	outer.Define(makeIdent("b", types.Basics[types.UTF8]))

	inner := NewEnclosedSymbolTable(outer)
	inner.Define(makeIdent("c", types.Basics[types.Bool]))

	var names []string
	inner.ForEachGlobal(func(name string, sym Symbol) {
		if name != "_" {
			names = append(names, name)
		}
	})

	if len(names) != 2 {
		t.Fatalf("expected 2 globals, got %d: %v", len(names), names)
	}
}

func TestUpdate(t *testing.T) {
	t.Parallel()
	s := NewSymbolTable()
	s.Define(makeIdent("x", types.Basics[types.Int64]))

	s.Update("x", types.Basics[types.Float64])

	sym, ok := s.Resolve("x")
	if !ok {
		t.Fatal("expected to resolve x")
	}
	if sym.Type().Kind() != types.Float64 {
		t.Errorf("updated type = %v, want float64", sym.Type())
	}
}

func TestUpdateNonexistent(t *testing.T) {
	t.Parallel()
	s := NewSymbolTable()

	// Should not panic when updating a non-existent symbol.
	s.Update("missing", types.Basics[types.Int64])
}

func TestResolveField(t *testing.T) {
	t.Parallel()
	s := NewSymbolTable()

	structType := &types.Struct{
		Fields: []*types.Field{
			{Name: "x", Type: types.Basics[types.Float64]},
			{Name: "y", Type: types.Basics[types.Float64]},
		},
	}
	ident := makeIdent("point", structType)
	ident.Qualifier = ast.QualifierVariable
	s.Define(ident)

	sym, ok := s.ResolveField("point", "x")
	if !ok {
		t.Fatal("expected to resolve field x")
	}
	if sym.Type().Kind() != types.Float64 {
		t.Errorf("field type = %v, want float64", sym.Type())
	}
	if sym.Scope != StructScope {
		t.Errorf("field scope = %v, want StructScope", sym.Scope)
	}
}

func TestResolveFieldNotFound(t *testing.T) {
	t.Parallel()
	s := NewSymbolTable()

	_, ok := s.ResolveField("point", "z")
	if ok {
		t.Fatal("expected resolve to fail for missing type")
	}
}

func TestResolveFieldFromOuter(t *testing.T) {
	t.Parallel()
	outer := NewSymbolTable()
	structType := &types.Struct{
		Fields: []*types.Field{
			{Name: "x", Type: types.Basics[types.Int64]},
		},
	}
	ident := makeIdent("pt", structType)
	ident.Qualifier = ast.QualifierVariable
	outer.Define(ident)

	inner := NewEnclosedSymbolTable(outer)

	sym, ok := inner.ResolveField("pt", "x")
	if !ok {
		t.Fatal("expected enclosed table to resolve field from outer")
	}
	if sym.Type().Kind() != types.Int64 {
		t.Errorf("field type = %v, want int64", sym.Type())
	}
}

func TestDefineEnumValue(t *testing.T) {
	t.Parallel()
	s := NewSymbolTable()

	valIdent := makeIdent("Open", types.Basics[types.UTF8])
	s.DefineEnumValue("Status", valIdent)

	sym, ok := s.ResolveField("Status", "Open")
	if !ok {
		t.Fatal("expected to resolve enum value")
	}
	if sym.Scope != EnumScope {
		t.Errorf("scope = %v, want EnumScope", sym.Scope)
	}
}

func TestSymbolType(t *testing.T) {
	t.Parallel()
	s := NewSymbolTable()

	ident := makeIdent("v", types.Basics[types.Bool])
	s.Define(ident)

	sym, _ := s.Resolve("v")
	if sym.Type().Kind() != types.Bool {
		t.Errorf("Type() = %v, want bool", sym.Type())
	}
}
