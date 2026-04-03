package component

import (
	goast "go/ast"
	gotoken "go/token"

	"github.com/samborkent/cog/internal/tokens"
)

var (
	ImportCmp  = &goast.Ident{Name: "go_cmp"}
	CmpOrdered = &goast.SelectorExpr{
		X:   ImportCmp,
		Sel: &goast.Ident{Name: "Ordered"},
	}

	Any        = &goast.Ident{Name: "any"}
	Comparable = &goast.Ident{Name: "comparable"}
)

// Go-native type names that can appear in generic tilde-union constraints.
// Indexed by cog token for a single source-of-truth. Types backed by
// external libraries (float16, complex32, int128, uint128) and ascii
// (Go []byte) have no Go-native tilde representation and are omitted.
var (
	GoInt     = []string{tokens.Int8.String(), tokens.Int16.String(), tokens.Int32.String(), tokens.Int64.String()}
	GoUint    = []string{tokens.Uint8.String(), tokens.Uint16.String(), tokens.Uint32.String(), tokens.Uint64.String()}
	GoFloat   = []string{tokens.Float32.String(), tokens.Float64.String()}
	GoComplex = []string{tokens.Complex64.String(), tokens.Complex128.String()}
	GoString  = []string{"string"} // tokens.UTF8 maps to Go string; tokens.ASCII ([]byte) cannot appear in tilde constraints.
)

// TildeUnion builds a Go AST expression for ~name1 | ~name2 | ... .
func TildeUnion(names ...string) goast.Expr {
	expr := goast.Expr(&goast.UnaryExpr{Op: gotoken.TILDE, X: &goast.Ident{Name: names[0]}})
	for _, name := range names[1:] {
		expr = &goast.BinaryExpr{
			X:  expr,
			Op: gotoken.OR,
			Y:  &goast.UnaryExpr{Op: gotoken.TILDE, X: &goast.Ident{Name: name}},
		}
	}

	return expr
}
