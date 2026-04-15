package component

import (
	goast "go/ast"
	gotoken "go/token"
	"strconv"
	"strings"
	"unsafe"

	"github.com/samborkent/cog/internal/ast"
)

// Pre-allocated boolean identifiers.
var (
	goTrue  = &goast.Ident{Name: "true"}
	goFalse = &goast.Ident{Name: "false"}
)

// Pre-allocated cog.ASCII composite literal type.
var asciiType = &goast.SelectorExpr{
	X:   &goast.Ident{Name: "cog"},
	Sel: &goast.Ident{Name: "ASCII"},
}

// Ident converts a Cog identifier name into a cached Go *ast.Ident,
// adjusting the casing based on the export flag.
func Ident(ident *ast.Identifier) *goast.Ident {
	if ident == nil {
		return nil
	}

	return cachedIdent(ConvertExport(ident.Name, ident.Exported, ident.Global))
}

// IdentName converts a simple string name to a cached Go *ast.Ident.
func IdentName(name string) *goast.Ident {
	return cachedIdent(name)
}

// BoolLit returns a pre-allocated Go *ast.Ident for "true" or "false".
func BoolLit(value bool) *goast.Ident {
	if value {
		return goTrue
	}

	return goFalse
}

// UTF8Lit converts a UTF-8 string value into a Go *ast.BasicLit.
func UTF8Lit(value string) *goast.BasicLit {
	if strings.ContainsAny(value, "\n\t") {
		return &goast.BasicLit{
			Kind:  gotoken.STRING,
			Value: "`" + value + "`",
		}
	}

	return &goast.BasicLit{
		Kind:  gotoken.STRING,
		Value: `"` + value + `"`,
	}
}

// ASCIILit converts an ASCII byte slice into a Go *ast.CompositeLit (cog.ASCII{...}).
func ASCIILit(value []byte) *goast.CompositeLit {
	elems := make([]goast.Expr, len(value))

	for i := range value {
		elems[i] = &goast.BasicLit{
			Kind:  gotoken.CHAR,
			Value: unsafe.String(&[]byte{'\'', value[i], '\''}[0], 3),
		}
	}

	return &goast.CompositeLit{
		Type: asciiType,
		Elts: elems,
	}
}

// Int8Lit converts an int8 into a Go *ast.BasicLit.
func Int8Lit(value int8) *goast.BasicLit {
	return &goast.BasicLit{
		Kind:  gotoken.INT,
		Value: strconv.FormatInt(int64(value), 10),
	}
}

// Int16Lit converts an int16 into a Go *ast.BasicLit.
func Int16Lit(value int16) *goast.BasicLit {
	return &goast.BasicLit{
		Kind:  gotoken.INT,
		Value: strconv.FormatInt(int64(value), 10),
	}
}

// Int32Lit converts an int32 into a Go *ast.BasicLit.
func Int32Lit(value int32) *goast.BasicLit {
	return &goast.BasicLit{
		Kind:  gotoken.INT,
		Value: strconv.FormatInt(int64(value), 10),
	}
}

// Int64Lit converts an int64 into a Go *ast.BasicLit.
func Int64Lit(value int64) *goast.BasicLit {
	return &goast.BasicLit{
		Kind:  gotoken.INT,
		Value: strconv.FormatInt(value, 10),
	}
}

// Int128Lit converts a stringified int128 value into a Go *ast.BasicLit.
func Int128Lit(value string) *goast.BasicLit {
	return &goast.BasicLit{
		Kind:  gotoken.INT,
		Value: value,
	}
}

// Uint8Lit converts a uint8 into a Go *ast.BasicLit.
func Uint8Lit(value uint8) *goast.BasicLit {
	return &goast.BasicLit{
		Kind:  gotoken.INT,
		Value: strconv.FormatUint(uint64(value), 10),
	}
}

// Uint16Lit converts a uint16 into a Go *ast.BasicLit.
func Uint16Lit(value uint16) *goast.BasicLit {
	return &goast.BasicLit{
		Kind:  gotoken.INT,
		Value: strconv.FormatUint(uint64(value), 10),
	}
}

// Uint32Lit converts a uint32 into a Go *ast.BasicLit.
func Uint32Lit(value uint32) *goast.BasicLit {
	return &goast.BasicLit{
		Kind:  gotoken.INT,
		Value: strconv.FormatUint(uint64(value), 10),
	}
}

// Uint64Lit converts a uint64 into a Go *ast.BasicLit.
func Uint64Lit(value uint64) *goast.BasicLit {
	return &goast.BasicLit{
		Kind:  gotoken.INT,
		Value: strconv.FormatUint(value, 10),
	}
}

// Uint128Lit converts a stringified uint128 value into a Go *ast.BasicLit.
func Uint128Lit(value string) *goast.BasicLit {
	return &goast.BasicLit{
		Kind:  gotoken.INT,
		Value: value,
	}
}

// Float32Lit converts a float32 into a Go *ast.BasicLit.
func Float32Lit(value float32) *goast.BasicLit {
	return &goast.BasicLit{
		Kind:  gotoken.FLOAT,
		Value: strconv.FormatFloat(float64(value), 'g', -1, 32),
	}
}

// Float64Lit converts a float64 into a Go *ast.BasicLit.
func Float64Lit(value float64) *goast.BasicLit {
	return &goast.BasicLit{
		Kind:  gotoken.FLOAT,
		Value: strconv.FormatFloat(value, 'g', -1, 64),
	}
}
