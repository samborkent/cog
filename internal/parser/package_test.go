package parser_test

import "testing"

func TestParsePackage(t *testing.T) {
	t.Parallel()

	t.Run("valid", func(t *testing.T) {
		t.Parallel()
		f := parse(t, `package mypackage
main : proc() = {}`)
		if f.Package == nil {
			t.Fatal("expected package node")
		}
		if f.Package.Identifier.Name != "mypackage" {
			t.Errorf("expected package name 'mypackage', got %q", f.Package.Identifier.Name)
		}
	})

	t.Run("missing", func(t *testing.T) {
		t.Parallel()
		parseShouldError(t, `main : proc() = {}`)
	})
}
