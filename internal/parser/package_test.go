package parser_test

import (
	"testing"

	"github.com/samborkent/cog/internal/ast"
)

func TestParsePackage(t *testing.T) {
	t.Parallel()

	t.Run("valid", func(t *testing.T) {
		t.Parallel()

		f := parse(t, `package mypackage
main : proc() = {}`)

		file := f.Node(1).(*ast.File)

		if file.Package == nil {
			t.Fatal("expected package node")
		}

		if file.Package.Identifier.Name != "mypackage" {
			t.Errorf("expected package name 'mypackage', got %q", file.Package.Identifier.Name)
		}
	})

	t.Run("missing", func(t *testing.T) {
		t.Parallel()
		parseShouldError(t, `main : proc() = {}`)
	})
}
