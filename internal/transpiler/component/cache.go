package component

import goast "go/ast"

// Package-level cache for reusable Go AST identifier nodes.
var idents = make(map[string]*goast.Ident)

// cachedIdent returns a cached *goast.Ident for the given name,
// creating and caching one if it doesn't exist yet.
func cachedIdent(name string) *goast.Ident {
	if ident, ok := idents[name]; ok {
		return ident
	}

	ident := &goast.Ident{Name: name}
	idents[name] = ident

	return ident
}

// ResetCache clears the component cache. Should be called between transpilation runs.
func ResetCache() {
	clear(idents)
}
