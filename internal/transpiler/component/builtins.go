package component

import (
	goast "go/ast"
	gotypes "go/types"
)

// Pre-allocated builtin selectors.
var (
	builtinPkg = &goast.Ident{Name: "builtin"}

	builtinIfSel = &goast.SelectorExpr{
		X:   builtinPkg,
		Sel: &goast.Ident{Name: "If"},
	}
	builtinMapSel = &goast.SelectorExpr{
		X:   builtinPkg,
		Sel: &goast.Ident{Name: "Map"},
	}
	builtinPrintSel = &goast.SelectorExpr{
		X:   builtinPkg,
		Sel: &goast.Ident{Name: "Print"},
	}
	builtinPtrSel = &goast.SelectorExpr{
		X:   builtinPkg,
		Sel: &goast.Ident{Name: "Ptr"},
	}
	builtinSetSel = &goast.SelectorExpr{
		X:   builtinPkg,
		Sel: &goast.Ident{Name: "Set"},
	}
	builtinSliceSel = &goast.SelectorExpr{
		X:   builtinPkg,
		Sel: &goast.Ident{Name: "Slice"},
	}

	defaultCapType = &goast.Ident{Name: gotypes.Typ[gotypes.Uint8].String()}
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

func BuiltinMap(keyType, valueType, capType, capacity goast.Expr) *goast.CallExpr {
	var args []goast.Expr

	if capacity != nil {
		args = append(args, capacity)
	}

	indices := []goast.Expr{keyType, valueType}

	if capType != nil {
		indices = append(indices, capType)
	} else if capacity == nil {
		indices = append(indices, defaultCapType)
	}

	return &goast.CallExpr{
		Fun: &goast.IndexListExpr{
			X:       builtinMapSel,
			Indices: indices,
		},
		Args: args,
	}
}

func BuiltinPrint(arg goast.Expr) *goast.CallExpr {
	return &goast.CallExpr{
		Fun:  builtinPrintSel,
		Args: []goast.Expr{arg},
	}
}

func BuiltinPtr(valueType goast.Expr) *goast.CallExpr {
	return &goast.CallExpr{
		Fun: &goast.IndexExpr{
			X:     builtinPtrSel,
			Index: valueType,
		},
	}
}

func BuiltinSet(keyType, capType, capacity goast.Expr) *goast.CallExpr {
	var args []goast.Expr

	if capacity != nil {
		args = append(args, capacity)
	}

	indices := []goast.Expr{keyType}

	if capType != nil {
		indices = append(indices, capType)
	} else if capacity == nil {
		indices = append(indices, defaultCapType)
	}

	return &goast.CallExpr{
		Fun: &goast.IndexListExpr{
			X:       builtinSetSel,
			Indices: indices,
		},
		Args: args,
	}
}

func BuiltinSlice(elemType, lenType, length, capacity goast.Expr) *goast.CallExpr {
	args := []goast.Expr{length}

	if capacity != nil {
		args = append(args, capacity)
	}

	indices := []goast.Expr{elemType}
	if lenType != nil {
		indices = append(indices, lenType)
	}

	return &goast.CallExpr{
		Fun: &goast.IndexListExpr{
			X:       builtinSliceSel,
			Indices: indices,
		},
		Args: args,
	}
}
