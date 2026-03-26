package transpiler

import (
	"errors"
	"fmt"
	goast "go/ast"
	token "go/token"

	"github.com/samborkent/cog/internal/ast"
	"github.com/samborkent/cog/internal/transpiler/component"
	"github.com/samborkent/cog/internal/types"
)

type Builtins string

const (
	BuiltinCast  Builtins = "cast"
	BuiltinIf    Builtins = "if"
	BuiltinMap   Builtins = "map"
	BuiltinPrint Builtins = "print"
	BuiltinPtr   Builtins = "ptr"
	BuiltinSet   Builtins = "set"
	BuiltinSlice Builtins = "slice"
)

func (t *Transpiler) convertBuiltin(node *ast.Builtin) (goast.Expr, error) {
	switch Builtins(node.Name) {
	case BuiltinIf:
		if len(node.Arguments) == 0 || len(node.Arguments) > 3 {
			return nil, fmt.Errorf("wrong number of arguments, got %d", len(node.Arguments))
		}

		args := make([]goast.Expr, 0, len(node.Arguments))

		condition, err := t.convertExpr(node.Arguments[0])
		if err != nil {
			return nil, fmt.Errorf("converting @if builtin condition expression: %w", err)
		}

		args = append(args, condition)

		consequence, err := t.convertExpr(node.Arguments[1])
		if err != nil {
			return nil, fmt.Errorf("converting @if builtin consequence: %w", err)
		}

		args = append(args, consequence)

		if len(node.Arguments) == 3 {
			alternative, err := t.convertExpr(node.Arguments[2])
			if err != nil {
				return nil, fmt.Errorf("converting @if builtin alternative: %w", err)
			}

			args = append(args, alternative)
		}

		expectedIfType := node.Arguments[1].Type()

		if len(node.TypeArguments) >= 1 {
			expectedIfType = node.TypeArguments[0]
		}

		ifType, err := t.convertType(expectedIfType)
		if err != nil {
			return nil, fmt.Errorf("converting @if return type: %w", err)
		}

		var boolType goast.Expr

		if len(node.TypeArguments) == 2 {
			boolType, err = t.convertType(node.TypeArguments[1])
			if err != nil {
				return nil, fmt.Errorf("converting @if bool type: %w", err)
			}
		}

		t.addBuiltinImport()

		return component.BuiltinIf(ifType, boolType, args...), nil
	case BuiltinMap:
		if len(node.TypeArguments) < 2 || len(node.TypeArguments) > 3 {
			return nil, fmt.Errorf("@map expects 2 or 3 type arguments, got %d", len(node.TypeArguments))
		}

		if len(node.Arguments) > 1 {
			return nil, fmt.Errorf("@map expects at most 1 argument, got %d", len(node.Arguments))
		}

		keyType, err := t.convertType(node.TypeArguments[0])
		if err != nil {
			return nil, fmt.Errorf("converting @map builtin key type: %w", err)
		}

		valueType, err := t.convertType(node.TypeArguments[1])
		if err != nil {
			return nil, fmt.Errorf("converting @map builtin value type: %w", err)
		}

		var capacity goast.Expr

		if len(node.Arguments) == 1 {
			capacity, err = t.convertExpr(node.Arguments[0])
			if err != nil {
				return nil, fmt.Errorf("converting @map builtin capacity argument: %w", err)
			}

			switch node.Arguments[0].(type) {
			case *ast.Prefix:
				return nil, errors.New("@map capacity must be positive")
			}
		}

		return component.BuiltinMap(keyType, valueType, capacity), nil
	case BuiltinPrint:
		if len(node.Arguments) != 1 {
			return nil, fmt.Errorf("print expects 1 argument, got %d", len(node.Arguments))
		}

		arg, err := t.convertExpr(node.Arguments[0])
		if err != nil {
			return nil, fmt.Errorf("converting print argument: %w", err)
		}

		// Print underlying value of enum instead of enum itself.
		if node.Arguments[0].Type().Kind() == types.EnumKind {
			enumType, ok := node.Arguments[0].Type().(*types.Alias)
			if !ok {
				return nil, fmt.Errorf("unable to cast enum to alias for @print argument")
			}

			arg = &goast.IndexExpr{
				X:     &goast.Ident{Name: convertExport(enumType.Name, enumType.Exported)},
				Index: arg,
			}
		}

		t.addBuiltinImport()

		return component.BuiltinPrint(arg), nil
	case BuiltinPtr:
		if len(node.TypeArguments) != 1 {
			return nil, fmt.Errorf("@ptr expects 1 type argument, got %d", len(node.TypeArguments))
		}

		if len(node.Arguments) > 0 {
			return nil, fmt.Errorf("@ptr cannot take any arguments, got %d", len(node.Arguments))
		}

		valueType, err := t.convertType(node.TypeArguments[0])
		if err != nil {
			return nil, fmt.Errorf("converting @ptr value type: %w", err)
		}

		return component.BuiltinPtr(valueType), nil
	case BuiltinSet:
		if len(node.TypeArguments) < 1 || len(node.TypeArguments) > 2 {
			return nil, fmt.Errorf("@set expects 1 or 2 type arguments, got %d", len(node.TypeArguments))
		}

		if len(node.Arguments) > 1 {
			return nil, fmt.Errorf("@set expects at most 1 argument, got %d", len(node.Arguments))
		}

		keyType, err := t.convertType(node.TypeArguments[0])
		if err != nil {
			return nil, fmt.Errorf("converting @set builtin key type: %w", err)
		}

		t.addCogImport()

		var capacity goast.Expr

		if len(node.Arguments) == 1 {
			capacity, err = t.convertExpr(node.Arguments[0])
			if err != nil {
				return nil, fmt.Errorf("converting @set builtin capacity argument: %w", err)
			}

			switch node.Arguments[0].(type) {
			case *ast.Prefix:
				return nil, errors.New("@set capacity must be positive")
			}
		}

		return component.BuiltinSet(keyType, capacity), nil
	case BuiltinSlice:
		if len(node.TypeArguments) < 1 || len(node.TypeArguments) > 2 {
			return nil, fmt.Errorf("@slice expects 1 or 2 type arguments, got %d", len(node.TypeArguments))
		}

		if len(node.Arguments) < 1 {
			return nil, fmt.Errorf("@slice expects at least 1 argument, got %d", len(node.Arguments))
		}

		elemType, err := t.convertType(node.TypeArguments[0])
		if err != nil {
			return nil, fmt.Errorf("converting @slice element type: %w", err)
		}

		length, err := t.convertExpr(node.Arguments[0])
		if err != nil {
			return nil, fmt.Errorf("converting @slice length argument: %w", err)
		}

		switch node.Arguments[0].(type) {
		case *ast.Prefix:
			return nil, errors.New("@slice length must be positive")
		}

		var capacity goast.Expr

		if len(node.Arguments) == 2 {
			capacity, err = t.convertExpr(node.Arguments[1])
			if err != nil {
				return nil, fmt.Errorf("converting @slice capacity argument: %w", err)
			}

			switch node.Arguments[1].(type) {
			case *ast.Prefix:
				return nil, errors.New("@slice capacity must be positive")
			}
		}

		return component.BuiltinSlice(elemType, length, capacity), nil
	case BuiltinCast:
		if len(node.TypeArguments) == 0 {
			return nil, fmt.Errorf("@cast requires at least 1 type argument")
		}

		if len(node.Arguments) != 1 {
			return nil, fmt.Errorf("@cast expects 1 argument, got %d", len(node.Arguments))
		}

		arg, err := t.convertExpr(node.Arguments[0])
		if err != nil {
			return nil, fmt.Errorf("converting @cast argument: %w", err)
		}

		srcKind := node.Arguments[0].Type().Kind()
		dstKind := node.TypeArguments[0].Kind()

		return t.convertCast(arg, srcKind, dstKind)
	default:
		return nil, fmt.Errorf("unknown builtin function '%s'", node.Name)
	}
}

// convertCast generates Go AST for a bitwise @cast from srcKind to dstKind.
// For same-family lossless casts, direct Go primitive conversions are used.
// For cross-family casts, the strategy is: normalize src → unsigned int of
// src bit width, widen if needed, then denormalize to the target type.
func (t *Transpiler) convertCast(arg goast.Expr, srcKind, dstKind types.Kind) (goast.Expr, error) {
	// Special case: ascii → utf8.
	if srcKind == types.ASCII && dstKind == types.UTF8 {
		return &goast.CallExpr{
			Fun:  &goast.Ident{Name: "string"},
			Args: []goast.Expr{arg},
		}, nil
	}

	// Fast path: direct Go primitive casts that are lossless.
	if goType, ok := directCastType(srcKind, dstKind); ok {
		return &goast.CallExpr{
			Fun:  &goast.Ident{Name: goType},
			Args: []goast.Expr{arg},
		}, nil
	}

	srcBits := types.Size(srcKind)
	dstBits := types.Size(dstKind)

	// Step 1: Normalize source to unsigned int of same bit width.
	normalized, err := t.castNormalize(arg, srcKind)
	if err != nil {
		return nil, err
	}

	// Step 2: Widen unsigned int if target is larger.
	widened := normalized
	if srcBits < dstBits {
		widened = t.castWiden(normalized, srcBits, dstBits)
	}

	// Step 3: Denormalize unsigned int to target type.
	return t.castDenormalize(widened, dstKind)
}

// directCastType returns the Go type name for a direct lossless primitive cast,
// or false if the cast requires the normalize→widen→denormalize pipeline.
func directCastType(src, dst types.Kind) (string, bool) {
	switch src {
	case types.Uint8:
		switch dst {
		case types.Int8:
			return "int8", true
		case types.Uint16:
			return "uint16", true
		case types.Uint32:
			return "uint32", true
		case types.Uint64:
			return "uint64", true
		}
	case types.Uint16:
		switch dst {
		case types.Int16:
			return "int16", true
		case types.Uint32:
			return "uint32", true
		case types.Uint64:
			return "uint64", true
		}
	case types.Uint32:
		switch dst {
		case types.Int32:
			return "int32", true
		case types.Uint64:
			return "uint64", true
		}
	case types.Uint64:
		if dst == types.Int64 {
			return "int64", true
		}
	case types.Int8:
		switch dst {
		case types.Uint8:
			return "uint8", true
		case types.Int16:
			return "int16", true
		case types.Int32:
			return "int32", true
		case types.Int64:
			return "int64", true
		}
	case types.Int16:
		switch dst {
		case types.Uint16:
			return "uint16", true
		case types.Int32:
			return "int32", true
		case types.Int64:
			return "int64", true
		}
	case types.Int32:
		switch dst {
		case types.Uint32:
			return "uint32", true
		case types.Int64:
			return "int64", true
		}
	case types.Int64:
		if dst == types.Uint64 {
			return "uint64", true
		}
	case types.Float32:
		if dst == types.Float64 {
			return "float64", true
		}
	case types.Complex64:
		if dst == types.Complex128 {
			return "complex128", true
		}
	}

	return "", false
}

// castNormalize converts a value to the unsigned integer of the same bit width.
func (t *Transpiler) castNormalize(arg goast.Expr, kind types.Kind) (goast.Expr, error) {
	switch kind {
	case types.Bool:
		t.addBuiltinImport()
		return component.BuiltinIf(&goast.Ident{Name: "uint8"}, nil, arg, &goast.BasicLit{Kind: token.INT, Value: "1"}, &goast.BasicLit{Kind: token.INT, Value: "0"}), nil
	case types.Uint8, types.Uint16, types.Uint32, types.Uint64:
		return arg, nil
	case types.Int8:
		return &goast.CallExpr{Fun: &goast.Ident{Name: "uint8"}, Args: []goast.Expr{arg}}, nil
	case types.Int16:
		return &goast.CallExpr{Fun: &goast.Ident{Name: "uint16"}, Args: []goast.Expr{arg}}, nil
	case types.Int32:
		return &goast.CallExpr{Fun: &goast.Ident{Name: "uint32"}, Args: []goast.Expr{arg}}, nil
	case types.Int64:
		return &goast.CallExpr{Fun: &goast.Ident{Name: "uint64"}, Args: []goast.Expr{arg}}, nil
	case types.Float16:
		return &goast.CallExpr{
			Fun: &goast.SelectorExpr{X: arg, Sel: &goast.Ident{Name: "Bits"}},
		}, nil
	case types.Float32:
		t.addStdLibImport("math")
		return &goast.CallExpr{
			Fun:  &goast.SelectorExpr{X: &goast.Ident{Name: goStdLibAlias("math")}, Sel: &goast.Ident{Name: "Float32bits"}},
			Args: []goast.Expr{arg},
		}, nil
	case types.Float64:
		t.addStdLibImport("math")
		return &goast.CallExpr{
			Fun:  &goast.SelectorExpr{X: &goast.Ident{Name: goStdLibAlias("math")}, Sel: &goast.Ident{Name: "Float64bits"}},
			Args: []goast.Expr{arg},
		}, nil
	case types.Complex32:
		t.addCogImport()
		return &goast.CallExpr{
			Fun:  &goast.SelectorExpr{X: &goast.Ident{Name: "cog"}, Sel: &goast.Ident{Name: "Complex32Bits"}},
			Args: []goast.Expr{arg},
		}, nil
	case types.Complex64:
		t.addCogImport()
		return &goast.CallExpr{
			Fun:  &goast.SelectorExpr{X: &goast.Ident{Name: "cog"}, Sel: &goast.Ident{Name: "Complex64Bits"}},
			Args: []goast.Expr{arg},
		}, nil
	case types.Complex128:
		t.addCogImport()
		return &goast.CallExpr{
			Fun:  &goast.SelectorExpr{X: &goast.Ident{Name: "cog"}, Sel: &goast.Ident{Name: "Complex128Bits"}},
			Args: []goast.Expr{arg},
		}, nil
	case types.Uint128:
		return arg, nil
	case types.Int128:
		t.addCogImport()
		return &goast.CallExpr{
			Fun:  &goast.SelectorExpr{X: &goast.Ident{Name: "cog"}, Sel: &goast.Ident{Name: "Int128ToUint128"}},
			Args: []goast.Expr{arg},
		}, nil
	default:
		return nil, fmt.Errorf("@cast: cannot normalize type kind %v", kind)
	}
}

// castWiden widens an unsigned int from srcBits to dstBits.
func (t *Transpiler) castWiden(arg goast.Expr, srcBits, dstBits int) goast.Expr {
	if dstBits == 128 {
		t.addCogImport()
		if srcBits < 64 {
			arg = &goast.CallExpr{Fun: &goast.Ident{Name: "uint64"}, Args: []goast.Expr{arg}}
		}
		return &goast.CallExpr{
			Fun:  &goast.SelectorExpr{X: &goast.Ident{Name: "cog"}, Sel: &goast.Ident{Name: "Uint128From64"}},
			Args: []goast.Expr{arg},
		}
	}

	return &goast.CallExpr{
		Fun:  &goast.Ident{Name: uintNameForBits(dstBits)},
		Args: []goast.Expr{arg},
	}
}

// castDenormalize converts an unsigned int to the target type.
func (t *Transpiler) castDenormalize(arg goast.Expr, kind types.Kind) (goast.Expr, error) {
	switch kind {
	case types.Bool:
		return &goast.BinaryExpr{X: arg, Op: token.NEQ, Y: &goast.BasicLit{Kind: token.INT, Value: "0"}}, nil
	case types.Uint8:
		return &goast.CallExpr{Fun: &goast.Ident{Name: "uint8"}, Args: []goast.Expr{arg}}, nil
	case types.Uint16:
		return &goast.CallExpr{Fun: &goast.Ident{Name: "uint16"}, Args: []goast.Expr{arg}}, nil
	case types.Uint32:
		return &goast.CallExpr{Fun: &goast.Ident{Name: "uint32"}, Args: []goast.Expr{arg}}, nil
	case types.Uint64:
		return &goast.CallExpr{Fun: &goast.Ident{Name: "uint64"}, Args: []goast.Expr{arg}}, nil
	case types.Int8:
		return &goast.CallExpr{Fun: &goast.Ident{Name: "int8"}, Args: []goast.Expr{arg}}, nil
	case types.Int16:
		return &goast.CallExpr{Fun: &goast.Ident{Name: "int16"}, Args: []goast.Expr{arg}}, nil
	case types.Int32:
		return &goast.CallExpr{Fun: &goast.Ident{Name: "int32"}, Args: []goast.Expr{arg}}, nil
	case types.Int64:
		return &goast.CallExpr{Fun: &goast.Ident{Name: "int64"}, Args: []goast.Expr{arg}}, nil
	case types.Float16:
		t.addCogImport()
		return &goast.CallExpr{
			Fun:  &goast.SelectorExpr{X: &goast.Ident{Name: "cog"}, Sel: &goast.Ident{Name: "Float16Frombits"}},
			Args: []goast.Expr{arg},
		}, nil
	case types.Float32:
		t.addStdLibImport("math")
		return &goast.CallExpr{
			Fun:  &goast.SelectorExpr{X: &goast.Ident{Name: goStdLibAlias("math")}, Sel: &goast.Ident{Name: "Float32frombits"}},
			Args: []goast.Expr{arg},
		}, nil
	case types.Float64:
		t.addStdLibImport("math")
		return &goast.CallExpr{
			Fun:  &goast.SelectorExpr{X: &goast.Ident{Name: goStdLibAlias("math")}, Sel: &goast.Ident{Name: "Float64frombits"}},
			Args: []goast.Expr{arg},
		}, nil
	case types.Complex32:
		t.addCogImport()
		return &goast.CallExpr{
			Fun:  &goast.SelectorExpr{X: &goast.Ident{Name: "cog"}, Sel: &goast.Ident{Name: "Complex32FromBits"}},
			Args: []goast.Expr{arg},
		}, nil
	case types.Complex64:
		t.addCogImport()
		return &goast.CallExpr{
			Fun:  &goast.SelectorExpr{X: &goast.Ident{Name: "cog"}, Sel: &goast.Ident{Name: "Complex64FromBits"}},
			Args: []goast.Expr{arg},
		}, nil
	case types.Complex128:
		t.addCogImport()
		return &goast.CallExpr{
			Fun:  &goast.SelectorExpr{X: &goast.Ident{Name: "cog"}, Sel: &goast.Ident{Name: "Complex128FromBits"}},
			Args: []goast.Expr{arg},
		}, nil
	case types.Uint128:
		// Arg is already a Uint128 from the widen step.
		return arg, nil
	case types.Int128:
		t.addCogImport()
		return &goast.CallExpr{
			Fun:  &goast.SelectorExpr{X: &goast.Ident{Name: "cog"}, Sel: &goast.Ident{Name: "Uint128ToInt128"}},
			Args: []goast.Expr{arg},
		}, nil
	default:
		return nil, fmt.Errorf("@cast: cannot denormalize to type kind %v", kind)
	}
}

// uintNameForBits returns the Go unsigned integer type name for the given bit width.
func uintNameForBits(bits int) string {
	switch bits {
	case 8:
		return "uint8"
	case 16:
		return "uint16"
	case 32:
		return "uint32"
	case 64:
		return "uint64"
	default:
		return "uint64"
	}
}
