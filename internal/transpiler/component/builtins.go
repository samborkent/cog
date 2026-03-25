package component

import (
	goast "go/ast"
)

// Pre-allocated builtin selectors.
var (
	builtinPkg = &goast.Ident{Name: "builtin"}

	builtinIfSel = &goast.SelectorExpr{
		X:   builtinPkg,
		Sel: &goast.Ident{Name: "If"},
	}
	builtinPrintSel = &goast.SelectorExpr{
		X:   builtinPkg,
		Sel: &goast.Ident{Name: "Print"},
	}

	cogPkg = &goast.Ident{Name: "cog"}
)

func BuiltinIf(ifType, boolType goast.Expr, args ...goast.Expr) *goast.CallExpr {
	indices := []goast.Expr{ifType}

	if boolType != nil {
		indices = append(indices, boolType)
	}

	return &goast.CallExpr{
		Fun: &goast.IndexListExpr{
			X:       builtinIfSel,
			Indices: indices,
		},
		Args: args,
	}
}

// BuiltinMap generates make(map[K]V) or make(map[K]V, cap).
func BuiltinMap(keyType, valueType, capacity goast.Expr) *goast.CallExpr {
	mapType := &goast.MapType{
		Key:   keyType,
		Value: valueType,
	}

	args := []goast.Expr{mapType}
	if capacity != nil {
		args = append(args, capacity)
	}

	return &goast.CallExpr{
		Fun:  &goast.Ident{Name: "make"},
		Args: args,
	}
}

func BuiltinPrint(arg goast.Expr) *goast.CallExpr {
	return &goast.CallExpr{
		Fun:  builtinPrintSel,
		Args: []goast.Expr{arg},
	}
}

// BuiltinPtr generates new(T).
func BuiltinPtr(valueType goast.Expr) *goast.CallExpr {
	return &goast.CallExpr{
		Fun:  &goast.Ident{Name: "new"},
		Args: []goast.Expr{valueType},
	}
}

// BuiltinSet generates make(cog.Set[K]) or make(cog.Set[K], cap).
func BuiltinSet(keyType, capacity goast.Expr) *goast.CallExpr {
	setType := &goast.IndexExpr{
		X: &goast.SelectorExpr{
			X:   cogPkg,
			Sel: &goast.Ident{Name: "Set"},
		},
		Index: keyType,
	}

	args := []goast.Expr{setType}
	if capacity != nil {
		args = append(args, capacity)
	}

	return &goast.CallExpr{
		Fun:  &goast.Ident{Name: "make"},
		Args: args,
	}
}

// BuiltinSlice generates make([]T, len) or make([]T, len, cap).
func BuiltinSlice(elemType, length, capacity goast.Expr) *goast.CallExpr {
	sliceType := &goast.ArrayType{
		Elt: elemType,
	}

	args := []goast.Expr{sliceType, length}
	if capacity != nil {
		args = append(args, capacity)
	}

	return &goast.CallExpr{
		Fun:  &goast.Ident{Name: "make"},
		Args: args,
	}
}
