package parser

import (
	"runtime"
	"testing"

	"github.com/samborkent/cog/internal/ast"
	"github.com/samborkent/cog/internal/tokens"
	"github.com/samborkent/cog/internal/types"
)

func makeIdent(name string, vt types.Type) *ast.Identifier {
	return &ast.Identifier{
		Token:     tokens.Token{Type: tokens.Identifier, Literal: name},
		ValueType: vt,
	}
}

func makeType(name string, vt types.Type) *ast.Type {
	return &ast.Type{
		Token: tokens.Token{Type: tokens.Identifier, Literal: name},
		Alias: &types.Alias{
			Derived: vt,
		},
	}
}

func TestNewSymbolTable(t *testing.T) {
	t.Parallel()

	s := NewSymbolTable()

	sym, ok := s.ResolveIdent("_")
	if !ok {
		t.Fatal("expected blank identifier")
	}

	if sym.Scope != LocalScope {
		t.Errorf("blank scope = %v, want LocalScope", sym.Scope)
	}
}

func TestNewSymbolTableAuto(t *testing.T) {
	// Below cutoff should always be single-thread mode.
	small := NewSymbolTableAuto(2, 4)
	if small.concurrent {
		t.Fatal("expected small table to be non-concurrent")
	}

	// At/above cutoff uses concurrent mode only when GOMAXPROCS > 1.
	big := NewSymbolTableAuto(4, 4)
	wantConcurrent := runtime.GOMAXPROCS(-1) > 1
	if big.concurrent != wantConcurrent {
		t.Fatalf("concurrent = %v, want %v", big.concurrent, wantConcurrent)
	}
}

func TestDefineAndResolve(t *testing.T) {
	t.Parallel()

	s := NewSymbolTable()

	ident := makeIdent("x", types.Basics[types.Int64])
	s.DefineIdent(ident)

	sym, ok := s.ResolveIdent("x")
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
	}
	s.DefineIdent(ident)

	sym, ok := s.ResolveIdent("y")
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

	_, ok := s.ResolveIdent("nonexistent")
	if ok {
		t.Fatal("expected resolve to fail")
	}
}

func TestEnclosedSymbolTable(t *testing.T) {
	t.Parallel()

	outer := NewSymbolTable()
	outer.DefineIdent(makeIdent("a", types.Basics[types.UTF8]))

	inner := NewEnclosedSymbolTable(outer)
	inner.DefineIdent(makeIdent("b", types.Basics[types.Int64]))

	if _, ok := inner.ResolveIdent("b"); !ok {
		t.Fatal("expected inner to resolve b")
	}

	sym, ok := inner.ResolveIdent("a")
	if !ok {
		t.Fatal("expected inner to resolve a from outer")
	}

	if sym.Type().Kind() != types.UTF8 {
		t.Errorf("type = %v, want utf8", sym.Type())
	}

	bsym, _ := inner.ResolveIdent("b")
	if bsym.Scope != LocalScope {
		t.Errorf("inner scope = %v, want LocalScope", bsym.Scope)
	}
}

func TestDefineGlobal(t *testing.T) {
	t.Parallel()

	s := NewSymbolTable()

	ident := makeIdent("g", types.Basics[types.Float64])
	s.DefineGlobalIdent(ident)

	sym, ok := s.ResolveIdent("g")
	if !ok {
		t.Fatal("expected to resolve g")
	}

	if sym.Scope != ScanScope {
		t.Errorf("scope = %v, want ScanScope", sym.Scope)
	}
}

func TestDefineGlobalResolvesForwardStub(t *testing.T) {
	t.Parallel()

	s := NewSymbolTable()

	// Create a forward stub (ScanScope, ValueType=None).
	stub := makeType("Point", types.None)
	s.DefineGlobalType(stub)

	// Now resolve the stub with the real struct type.
	structType := &types.Struct{
		Fields: []*types.Field{
			{Name: "x", Type: types.Basics[types.Float64], Exported: true},
			{Name: "y", Type: types.Basics[types.Float64], Exported: true},
		},
	}
	real := makeType("Point", structType)
	s.DefineGlobalType(real)

	// The stub pointer should have been updated in-place.
	sym, ok := s.ResolveType("Point")
	if !ok {
		t.Fatal("expected to resolve Point")
	}

	foundType, ok := sym.Alias.Derived.(*types.Struct)
	if !ok {
		t.Fatalf("expected StructKind, got %v", sym.Alias.Kind())
	}

	// Struct fields must be registered on the symbol table.
	xField := foundType.Field("x")
	if xField == nil {
		t.Fatal("expected to resolve field x")
	}

	if xField.Type.Kind() != types.Float64 {
		t.Errorf("field x type = %v, want Float64", xField.Type)
	}

	yField := foundType.Field("y")
	if yField == nil {
		t.Fatal("expected to resolve field y")
	}

	if yField.Type.Kind() != types.Float64 {
		t.Errorf("field y type = %v, want Float64", yField.Type)
	}
}

func TestDefineGlobalResolvesForwardStubNonStruct(t *testing.T) {
	t.Parallel()

	s := NewSymbolTable()

	// Create a forward stub.
	stub := makeIdent("count", types.None)
	s.DefineGlobalIdent(stub)

	// Resolve with a non-struct type.
	real := makeIdent("count", types.Basics[types.Int64])
	s.DefineGlobalIdent(real)

	sym, ok := s.ResolveIdent("count")
	if !ok {
		t.Fatal("expected to resolve count")
	}

	if sym.Identifier.ValueType.Kind() != types.Int64 {
		t.Errorf("expected Int64, got %v", sym.Identifier.ValueType.Kind())
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

	if got.Token.Literal != "strings" {
		t.Errorf("name = %q, want strings", got.Token.Literal)
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
		Path:         "geom",
		Name:         "geom",
		ExportValues: make(map[string]Symbol),
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

	s.DefineCogImport(&CogImport{Path: "a", Name: "a", ExportValues: make(map[string]Symbol)})
	s.DefineCogImport(&CogImport{Path: "b/c", Name: "c", ExportValues: make(map[string]Symbol)})

	imports := s.CogImports()
	if imports.Len() != 2 {
		t.Fatalf("expected 2 imports, got %d", imports.Len())
	}

	if _, ok := imports.Load("a"); !ok {
		t.Error("missing import a")
	}

	if _, ok := imports.Load("c"); !ok {
		t.Error("missing import c")
	}
}

func TestCogImportSharedInEnclosed(t *testing.T) {
	t.Parallel()

	outer := NewSymbolTable()
	outer.DefineCogImport(&CogImport{Path: "pkg", Name: "pkg", ExportValues: make(map[string]Symbol)})

	inner := NewEnclosedSymbolTable(outer)

	if _, ok := inner.ResolveCogImport("pkg"); !ok {
		t.Fatal("expected enclosed table to see outer cog import")
	}
}

func TestForEachGlobal(t *testing.T) {
	t.Parallel()

	outer := NewSymbolTable()
	outer.DefineIdent(makeIdent("a", types.Basics[types.Int64]))
	outer.DefineIdent(makeIdent("b", types.Basics[types.UTF8]))

	inner := NewEnclosedSymbolTable(outer)
	inner.DefineIdent(makeIdent("c", types.Basics[types.Bool]))

	var names []string

	for symbol := range inner.Symbols() {
		if symbol.Identifier.Token.Literal != "_" {
			names = append(names, symbol.Identifier.Token.Literal)
		}
	}

	if len(names) != 3 {
		t.Fatalf("expected 3 symbols, got %d: %v", len(names), names)
	}
}

func TestUpdate(t *testing.T) {
	t.Parallel()

	s := NewSymbolTable()

	ident := makeIdent("x", types.Basics[types.Int64])

	s.DefineIdent(ident)

	ident.ValueType = types.Basics[types.Float64]

	sym, ok := s.ResolveIdent("x")
	if !ok {
		t.Fatal("expected to resolve x")
	}

	if sym.Type().Kind() != types.Float64 {
		t.Errorf("updated type = %v, want float64", sym.Type())
	}
}

func TestSymbolType(t *testing.T) {
	t.Parallel()

	s := NewSymbolTable()

	ident := makeIdent("v", types.Basics[types.Bool])
	s.DefineIdent(ident)

	sym, _ := s.ResolveIdent("v")
	if sym.Type().Kind() != types.Bool {
		t.Errorf("Type() = %v, want bool", sym.Type())
	}
}
