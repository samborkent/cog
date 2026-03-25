package component

import (
	goast "go/ast"
	"sync"
)

// Package-level cache for reusable Go AST identifier nodes.
var idents sync.Map

// cachedIdent returns a cached *goast.Ident for the given name,
// creating and caching one if it doesn't exist yet.
func cachedIdent(name string) *goast.Ident {
	if v, ok := idents.Load(name); ok {
		return v.(*goast.Ident)
	}

	ident := &goast.Ident{Name: name}
	actual, _ := idents.LoadOrStore(name, ident)

	return actual.(*goast.Ident)
}

// ResetCache clears the component cache. Should be called between transpilation runs.
func ResetCache() {
	idents.Clear()
}
