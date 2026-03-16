package component

import (
	goast "go/ast"
	gotypes "go/types"
)

const (
	builtinPackage = "builtin"
	builtinIf      = "If"
	builtinMap     = "Map"
	builtinPrint   = "Print"
	builtinPtr     = "Ptr"
	builtinSet     = "Set"
	builtinSlice   = "Slice"
)

func BuiltinIf(ifType, boolType goast.Expr, args ...goast.Expr) *goast.CallExpr {
	indices := []goast.Expr{ifType}

	if boolType != nil {
		indices = append(indices, boolType)
	}

	return &goast.CallExpr{
		Fun: &goast.IndexListExpr{
			X: &goast.SelectorExpr{
				X:   &goast.Ident{Name: builtinPackage},
				Sel: &goast.Ident{Name: builtinIf},
			},
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
		indices = append(indices, &goast.Ident{Name: gotypes.Typ[gotypes.Uint8].String()})
	}

	return &goast.CallExpr{
		Fun: &goast.IndexListExpr{
			X: &goast.SelectorExpr{
				X:   &goast.Ident{Name: builtinPackage},
				Sel: &goast.Ident{Name: builtinMap},
			},
			Indices: indices,
		},
		Args: args,
	}
}

func BuiltinPrint(arg goast.Expr) *goast.CallExpr {
	return &goast.CallExpr{
		Fun: &goast.SelectorExpr{
			X:   &goast.Ident{Name: builtinPackage},
			Sel: &goast.Ident{Name: builtinPrint},
		},
		Args: []goast.Expr{arg},
	}
}

func BuiltinPtr(valueType goast.Expr) *goast.CallExpr {
	return &goast.CallExpr{
		Fun: &goast.IndexExpr{
			X: &goast.SelectorExpr{
				X:   &goast.Ident{Name: builtinPackage},
				Sel: &goast.Ident{Name: builtinPtr},
			},
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
		indices = append(indices, &goast.Ident{Name: gotypes.Typ[gotypes.Uint8].String()})
	}

	return &goast.CallExpr{
		Fun: &goast.IndexListExpr{
			X: &goast.SelectorExpr{
				X:   &goast.Ident{Name: builtinPackage},
				Sel: &goast.Ident{Name: builtinSet},
			},
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
			X: &goast.SelectorExpr{
				X:   &goast.Ident{Name: builtinPackage},
				Sel: &goast.Ident{Name: builtinSlice},
			},
			Indices: indices,
		},
		Args: args,
	}
}
