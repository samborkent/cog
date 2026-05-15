package component

import (
	goast "go/ast"
	"go/printer"
	"go/token"
	"strings"
	"testing"
)

func TestZeroValue(t *testing.T) {
	typ := &goast.Ident{Name: "int"}
	expr := ZeroValue(typ)

	var buf strings.Builder
	fset := token.NewFileSet()
	if err := printer.Fprint(&buf, fset, expr); err != nil {
		t.Fatal(err)
	}

	got := buf.String()
	if got != "*new(int)" {
		t.Fatalf("got %q, want *new(int)", got)
	}
}
