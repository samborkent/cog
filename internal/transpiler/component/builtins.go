package component

import (
	goast "go/ast"
)

const (
	builtinPackage = "builtin"
	builtinIf      = "If"
	builtinMap     = "Map"
	builtinPrint   = "Print"
)

func BuiltinIf(ifType goast.Expr, args ...goast.Expr) *goast.CallExpr {
	return &goast.CallExpr{
		Fun: &goast.IndexExpr{
			X: &goast.SelectorExpr{
				X:   &goast.Ident{Name: builtinPackage},
				Sel: &goast.Ident{Name: builtinIf},
			},
			Index: ifType,
		},
		Args: args,
	}
}

func BuiltinMap(keyType, valueType, capacity goast.Expr) *goast.CallExpr {
	var args []goast.Expr

	if capacity != nil {
		args = append(args, capacity)
	}

	return &goast.CallExpr{
		Fun: &goast.IndexListExpr{
			X: &goast.SelectorExpr{
				X:   &goast.Ident{Name: builtinPackage},
				Sel: &goast.Ident{Name: builtinMap},
			},
			Indices: []goast.Expr{keyType, valueType},
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
