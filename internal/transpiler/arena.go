package transpiler

import (
	goast "go/ast"
	gotoken "go/token"

	"github.com/samborkent/cog/internal/transpiler/component"
)

// injectArena post-processes a proc/func body to use arena allocation for
// variable-length @slice allocations that do not escape via return.
// An arena is only created when at least two eligible variable-length slice
// allocations exist, because the arena.NewArena()+Free() overhead (~80ns) is
// only recovered when multiple GC allocations are avoided. Pointers and
// literal-length slices are never arena-allocated — Go stack-allocates them.
func (t *Transpiler) injectArena(body *goast.BlockStmt) {
	if body == nil || len(body.List) == 0 {
		return
	}

	allocs := findAllocations(body)
	if len(allocs) == 0 {
		return
	}

	// Find identifiers that appear in return statements.
	returned := collectReturnedIdents(body)

	// Filter to eligible: only variable-length slices that are not returned.
	var eligible []allocation

	for _, a := range allocs {
		if !a.varLenSlice {
			continue
		}

		if _, isReturned := returned[a.name]; !isReturned {
			eligible = append(eligible, a)
		}
	}

	// Require at least 2 qualifying allocations to justify arena overhead.
	if len(eligible) < 2 {
		return
	}

	// Rewrite eligible allocations to use arena.
	for _, a := range eligible {
		component.RewriteSliceToArena(a.call)
	}

	// Prepend arena init and defer free.
	body.List = append([]goast.Stmt{
		component.ArenaInit(),
		component.ArenaDeferFree(),
	}, body.List...)

	t.addCogImport()
}

type allocation struct {
	name        string
	call        *goast.CallExpr
	varLenSlice bool // true if make([]T, <non-literal-length>)
}

// findAllocations scans top-level body statements for variable declarations
// and assignments whose value is make([]T, ...) or new(T).
func findAllocations(body *goast.BlockStmt) []allocation {
	var allocs []allocation

	for _, stmt := range body.List {
		switch s := stmt.(type) {
		case *goast.DeclStmt:
			genDecl, ok := s.Decl.(*goast.GenDecl)
			if !ok || genDecl.Tok != gotoken.VAR {
				continue
			}

			for _, spec := range genDecl.Specs {
				valSpec, ok := spec.(*goast.ValueSpec)
				if !ok || len(valSpec.Names) == 0 || len(valSpec.Values) == 0 {
					continue
				}

				if a, ok := classifyAllocCall(valSpec.Values[0], valSpec.Names[0].Name); ok {
					allocs = append(allocs, a)
				}
			}
		case *goast.AssignStmt:
			if len(s.Lhs) == 0 || len(s.Rhs) == 0 {
				continue
			}

			name := "_"
			if ident, ok := s.Lhs[0].(*goast.Ident); ok {
				name = ident.Name
			}

			if a, ok := classifyAllocCall(s.Rhs[0], name); ok {
				allocs = append(allocs, a)
			}
		}
	}

	return allocs
}

// classifyAllocCall checks if expr is a make([]T, ...) or new(T) call
// and returns a classified allocation.
func classifyAllocCall(expr goast.Expr, name string) (allocation, bool) {
	call, ok := expr.(*goast.CallExpr)
	if !ok {
		return allocation{}, false
	}

	ident, ok := call.Fun.(*goast.Ident)
	if !ok {
		return allocation{}, false
	}

	switch ident.Name {
	case "make":
		if len(call.Args) >= 2 {
			if _, isSlice := call.Args[0].(*goast.ArrayType); isSlice {
				varLen := !isIntLiteral(call.Args[1])
				return allocation{name: name, call: call, varLenSlice: varLen}, true
			}
		}
	case "new":
		if len(call.Args) == 1 {
			return allocation{name: name, call: call}, true
		}
	}

	return allocation{}, false
}

// isIntLiteral returns true if the expression is an integer literal.
func isIntLiteral(expr goast.Expr) bool {
	lit, ok := expr.(*goast.BasicLit)
	return ok && lit.Kind == gotoken.INT
}

// collectReturnedIdents walks the body (excluding nested function literals)
// and returns the set of identifier names found in return statement results.
func collectReturnedIdents(body *goast.BlockStmt) map[string]struct{} {
	returned := make(map[string]struct{})

	goast.Inspect(body, func(n goast.Node) bool {
		switch n := n.(type) {
		case *goast.FuncLit:
			return false // Don't descend into nested function literals.
		case *goast.ReturnStmt:
			for _, result := range n.Results {
				goast.Inspect(result, func(n goast.Node) bool {
					if id, ok := n.(*goast.Ident); ok {
						returned[id.Name] = struct{}{}
					}
					return true
				})
			}
		}

		return true
	})

	return returned
}
