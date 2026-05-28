package transpiler

import (
	"errors"
	"fmt"
	goast "go/ast"
	gotoken "go/token"
	"strconv"

	"github.com/samborkent/cog/internal/ast"
	"github.com/samborkent/cog/internal/transpiler/component"
	"github.com/samborkent/cog/internal/types"
)

type Builtins string

const (
	BuiltinAs    Builtins = "as"
	BuiltinCast  Builtins = "cast"
	BuiltinIf    Builtins = "if"
	BuiltinMap   Builtins = "map"
	BuiltinPrint Builtins = "print"
	BuiltinRef   Builtins = "ref"
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

		condition, err := t.convertExpr(t.Expr(node.Arguments[0]))
		if err != nil {
			return nil, fmt.Errorf("converting @if builtin condition expression: %w", err)
		}

		args = append(args, condition)

		consequenceExpr := t.Expr(node.Arguments[1])

		consequence, err := t.convertExpr(consequenceExpr)
		if err != nil {
			return nil, fmt.Errorf("converting @if builtin consequence: %w", err)
		}

		args = append(args, consequence)

		if len(node.Arguments) == 3 {
			alternative, err := t.convertExpr(t.Expr(node.Arguments[2]))
			if err != nil {
				return nil, fmt.Errorf("converting @if builtin alternative: %w", err)
			}

			args = append(args, alternative)
		}

		expectedIfType := consequenceExpr.Type()

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
			capacityExpr := t.Expr(node.Arguments[0])

			capacity, err = t.convertExpr(capacityExpr)
			if err != nil {
				return nil, fmt.Errorf("converting @map builtin capacity argument: %w", err)
			}

			switch capacityExpr.(type) {
			case *ast.Prefix:
				return nil, errors.New("@map capacity must be positive")
			}
		}

		return component.BuiltinMap(keyType, valueType, capacity), nil
	case BuiltinPrint:
		if len(node.Arguments) != 1 {
			return nil, fmt.Errorf("print expects 1 argument, got %d", len(node.Arguments))
		}

		argExpr := t.Expr(node.Arguments[0])

		arg, err := t.convertExpr(argExpr)
		if err != nil {
			return nil, fmt.Errorf("converting print argument: %w", err)
		}

		// Print underlying value of enum/error instead of enum itself.
		if argExpr.Type().Kind() == types.EnumKind ||
			argExpr.Type().Kind() == types.ErrorKind {
			enumType, ok := argExpr.Type().(*types.Alias)
			if !ok {
				return nil, fmt.Errorf("unable to cast enum to alias for @print argument")
			}

			arg = &goast.IndexExpr{
				X:     &goast.Ident{Name: component.ConvertExport(enumType.Name, enumType.Exported, enumType.Global)},
				Index: arg,
			}
		}

		t.addBuiltinImport()

		return component.BuiltinPrint(arg), nil
	case BuiltinRef:
		if len(node.TypeArguments) != 1 {
			return nil, fmt.Errorf("@ref expects 1 type argument, got %d", len(node.TypeArguments))
		}

		if len(node.Arguments) > 0 {
			return nil, fmt.Errorf("@ref cannot take any arguments, got %d", len(node.Arguments))
		}

		valueType, err := t.convertType(node.TypeArguments[0])
		if err != nil {
			return nil, fmt.Errorf("converting @ref value type: %w", err)
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
			capacityExpr := t.Expr(node.Arguments[0])

			capacity, err = t.convertExpr(capacityExpr)
			if err != nil {
				return nil, fmt.Errorf("converting @set builtin capacity argument: %w", err)
			}

			switch capacityExpr.(type) {
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

		lenExpr := t.Expr(node.Arguments[0])

		length, err := t.convertExpr(lenExpr)
		if err != nil {
			return nil, fmt.Errorf("converting @slice length argument: %w", err)
		}

		switch lenExpr.(type) {
		case *ast.Prefix:
			return nil, errors.New("@slice length must be positive")
		}

		var capacity goast.Expr

		if len(node.Arguments) == 2 {
			capacityExpr := t.Expr(node.Arguments[1])

			capacity, err = t.convertExpr(capacityExpr)
			if err != nil {
				return nil, fmt.Errorf("converting @slice capacity argument: %w", err)
			}

			switch capacityExpr.(type) {
			case *ast.Prefix:
				return nil, errors.New("@slice capacity must be positive")
			}
		}

		return component.BuiltinSlice(elemType, length, capacity), nil
	case BuiltinAs:
		if len(node.TypeArguments) == 0 {
			return nil, fmt.Errorf("@as requires at least 1 type argument")
		}

		if len(node.Arguments) != 1 {
			return nil, fmt.Errorf("@as expects 1 argument, got %d", len(node.Arguments))
		}

		argExpr := t.Expr(node.Arguments[0])

		arg, err := t.convertExpr(argExpr)
		if err != nil {
			return nil, fmt.Errorf("converting @as argument: %w", err)
		}

		srcType := argExpr.Type()
		dstType := node.TypeArguments[0]

		return t.convertAs(arg, srcType, dstType)
	case BuiltinCast:
		if len(node.TypeArguments) == 0 {
			return nil, fmt.Errorf("@cast requires at least 1 type argument")
		}

		if len(node.Arguments) != 1 {
			return nil, fmt.Errorf("@cast expects 1 argument, got %d", len(node.Arguments))
		}

		argExpr := t.Expr(node.Arguments[0])

		arg, err := t.convertExpr(argExpr)
		if err != nil {
			return nil, fmt.Errorf("converting @cast argument: %w", err)
		}

		srcKind := argExpr.Type().Kind()
		dstKind := node.TypeArguments[0].Kind()

		targetGoType, err := t.convertType(node.TypeArguments[0])
		if err != nil {
			return nil, fmt.Errorf("converting @cast target type: %w", err)
		}

		t.addCogImport()

		srcBits := types.Size(srcKind)
		dstBits := types.Size(dstKind)

		if srcBits > dstBits {
			return component.BuiltinNone(targetGoType), nil
		}

		castExpr, err := t.convertCast(arg, srcKind, dstKind)
		if err != nil {
			return nil, err
		}

		return component.BuiltinSome(targetGoType, castExpr), nil
	default:
		return nil, fmt.Errorf("unknown builtin function '%s'", node.Name)
	}
}

// convertCast generates Go AST for a bitwise type cast from srcKind to dstKind.
// For same-family lossless casts, direct Go primitive conversions are used.
// For cross-family casts, the strategy is: normalize src → unsigned int of
// src bit width, widen if needed, then denormalize to the target type.
func (t *Transpiler) convertCast(arg goast.Expr, srcKind, dstKind types.Kind) (goast.Expr, error) {
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
		return component.BuiltinIf(&goast.Ident{Name: "uint8"}, nil, arg, &goast.BasicLit{Kind: gotoken.INT, Value: "1"}, &goast.BasicLit{Kind: gotoken.INT, Value: "0"}), nil
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
		return &goast.BinaryExpr{X: arg, Op: gotoken.NEQ, Y: &goast.BasicLit{Kind: gotoken.INT, Value: "0"}}, nil
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

// convertAs generates Go AST for @as semantic type conversion.
// Unlike @cast (bitwise reinterpretation), @as does value-preserving
// conversions (e.g. int → string, string → int, bool → int, etc).
func (t *Transpiler) convertAs(arg goast.Expr, srcType, dstType types.Type) (goast.Expr, error) {
	srcKind := srcType.Kind()
	dstKind := dstType.Kind()

	dstGoType, err := t.convertType(dstType)
	if err != nil {
		return nil, fmt.Errorf("converting @as target type: %w", err)
	}

	// same kind → passthrough (identity)
	if srcKind == dstKind {
		return arg, nil
	}

	// Helper: convert bool to string (used for both ascii and utf8)
	if srcKind == types.Bool && types.IsString(dstType) {
		t.addBuiltinImport()
		if dstKind == types.ASCII {
			t.addCogImport()
			return &goast.CallExpr{
				Fun: &goast.SelectorExpr{X: &goast.Ident{Name: "cog"}, Sel: &goast.Ident{Name: "ASCII"}},
				Args: []goast.Expr{
					component.BuiltinIf(&goast.Ident{Name: "string"}, nil, arg,
						&goast.BasicLit{Kind: gotoken.STRING, Value: `"true"`},
						&goast.BasicLit{Kind: gotoken.STRING, Value: `"false"`}),
				},
			}, nil
		}
		return component.BuiltinIf(&goast.Ident{Name: "string"}, nil, arg,
			&goast.BasicLit{Kind: gotoken.STRING, Value: `"true"`},
			&goast.BasicLit{Kind: gotoken.STRING, Value: `"false"`}), nil
	}

	// ascii ↔ utf8: wrap in string() since cog.ASCII is []byte
	if srcKind == types.ASCII && dstKind == types.UTF8 {
		return &goast.CallExpr{Fun: &goast.Ident{Name: "string"}, Args: []goast.Expr{arg}}, nil
	}
	if srcKind == types.UTF8 && dstKind == types.ASCII {
		t.addCogImport()
		return &goast.CallExpr{
			Fun: &goast.SelectorExpr{X: &goast.Ident{Name: "cog"}, Sel: &goast.Ident{Name: "ASCII"}},
			Args: []goast.Expr{
				&goast.CallExpr{
					Fun:  &goast.Ident{Name: "string"},
					Args: []goast.Expr{arg},
				},
			},
		}, nil
	}

	// bool ↔ any numeric (non-complex)
	if srcKind == types.Bool && types.IsReal(dstType) {
		t.addBuiltinImport()
		return component.BuiltinIf(dstGoType, nil, arg,
			&goast.BasicLit{Kind: gotoken.INT, Value: "1"},
			&goast.BasicLit{Kind: gotoken.INT, Value: "0"}), nil
	}
	if dstKind == types.Bool && types.IsReal(srcType) {
		return &goast.BinaryExpr{X: arg, Op: gotoken.NEQ, Y: &goast.BasicLit{Kind: gotoken.INT, Value: "0"}}, nil
	}

	// string → bool
	if types.IsString(srcType) && dstKind == types.Bool {
		t.addStdLibImport("strconv")
		return &goast.CallExpr{
			Fun: &goast.FuncLit{
				Type: &goast.FuncType{
					Results: &goast.FieldList{List: []*goast.Field{{Type: dstGoType}}},
				},
				Body: &goast.BlockStmt{
					List: []goast.Stmt{
						&goast.AssignStmt{
							Lhs: []goast.Expr{&goast.Ident{Name: "_v"}, &goast.Ident{Name: "_"}},
							Tok: gotoken.DEFINE,
							Rhs: []goast.Expr{
								&goast.CallExpr{
									Fun: &goast.SelectorExpr{X: &goast.Ident{Name: goStdLibAlias("strconv")}, Sel: &goast.Ident{Name: "ParseBool"}},
									Args: []goast.Expr{
										&goast.CallExpr{Fun: &goast.Ident{Name: "string"}, Args: []goast.Expr{arg}},
									},
								},
							},
						},
						&goast.ReturnStmt{Results: []goast.Expr{&goast.Ident{Name: "_v"}}},
					},
				},
			},
		}, nil
	}

	// string → integer
	if types.IsString(srcType) && types.IsInt(dstType) {
		t.addStdLibImport("strconv")
		return &goast.CallExpr{
			Fun: &goast.FuncLit{
				Type: &goast.FuncType{
					Results: &goast.FieldList{List: []*goast.Field{{Type: dstGoType}}},
				},
				Body: &goast.BlockStmt{
					List: []goast.Stmt{
						&goast.AssignStmt{
							Lhs: []goast.Expr{&goast.Ident{Name: "_v"}, &goast.Ident{Name: "_"}},
							Tok: gotoken.DEFINE,
							Rhs: []goast.Expr{
								&goast.CallExpr{
									Fun: &goast.SelectorExpr{X: &goast.Ident{Name: goStdLibAlias("strconv")}, Sel: &goast.Ident{Name: "ParseInt"}},
									Args: []goast.Expr{
										&goast.CallExpr{Fun: &goast.Ident{Name: "string"}, Args: []goast.Expr{arg}},
										&goast.BasicLit{Kind: gotoken.INT, Value: "10"},
										&goast.BasicLit{Kind: gotoken.INT, Value: "64"},
									},
								},
							},
						},
						&goast.ReturnStmt{Results: []goast.Expr{&goast.CallExpr{Fun: dstGoType, Args: []goast.Expr{&goast.Ident{Name: "_v"}}}}},
					},
				},
			},
		}, nil
	}

	// string → unsigned integer
	if types.IsString(srcType) && types.IsUint(dstType) {
		t.addStdLibImport("strconv")
		return &goast.CallExpr{
			Fun: &goast.FuncLit{
				Type: &goast.FuncType{
					Results: &goast.FieldList{List: []*goast.Field{{Type: dstGoType}}},
				},
				Body: &goast.BlockStmt{
					List: []goast.Stmt{
						&goast.AssignStmt{
							Lhs: []goast.Expr{&goast.Ident{Name: "_v"}, &goast.Ident{Name: "_"}},
							Tok: gotoken.DEFINE,
							Rhs: []goast.Expr{
								&goast.CallExpr{
									Fun: &goast.SelectorExpr{X: &goast.Ident{Name: goStdLibAlias("strconv")}, Sel: &goast.Ident{Name: "ParseUint"}},
									Args: []goast.Expr{
										&goast.CallExpr{Fun: &goast.Ident{Name: "string"}, Args: []goast.Expr{arg}},
										&goast.BasicLit{Kind: gotoken.INT, Value: "10"},
										&goast.BasicLit{Kind: gotoken.INT, Value: "64"},
									},
								},
							},
						},
						&goast.ReturnStmt{Results: []goast.Expr{&goast.CallExpr{Fun: dstGoType, Args: []goast.Expr{&goast.Ident{Name: "_v"}}}}},
					},
				},
			},
		}, nil
	}

	// string → float
	if types.IsString(srcType) && types.IsFloat(dstType) {
		t.addStdLibImport("strconv")
		bits := types.Size(dstKind)
		return &goast.CallExpr{
			Fun: &goast.FuncLit{
				Type: &goast.FuncType{
					Results: &goast.FieldList{List: []*goast.Field{{Type: dstGoType}}},
				},
				Body: &goast.BlockStmt{
					List: []goast.Stmt{
						&goast.AssignStmt{
							Lhs: []goast.Expr{&goast.Ident{Name: "_v"}, &goast.Ident{Name: "_"}},
							Tok: gotoken.DEFINE,
							Rhs: []goast.Expr{
								&goast.CallExpr{
									Fun: &goast.SelectorExpr{X: &goast.Ident{Name: goStdLibAlias("strconv")}, Sel: &goast.Ident{Name: "ParseFloat"}},
									Args: []goast.Expr{
										&goast.CallExpr{Fun: &goast.Ident{Name: "string"}, Args: []goast.Expr{arg}},
										&goast.BasicLit{Kind: gotoken.INT, Value: strconv.Itoa(bits)},
									},
								},
							},
						},
						&goast.ReturnStmt{Results: []goast.Expr{&goast.CallExpr{Fun: dstGoType, Args: []goast.Expr{&goast.Ident{Name: "_v"}}}}},
					},
				},
			},
		}, nil
	}

	// integer → string (ascii or utf8)
	if types.IsInt(srcType) && types.IsString(dstType) {
		t.addStdLibImport("strconv")
		result := &goast.CallExpr{
			Fun: &goast.SelectorExpr{X: &goast.Ident{Name: goStdLibAlias("strconv")}, Sel: &goast.Ident{Name: "FormatInt"}},
			Args: []goast.Expr{
				&goast.CallExpr{Fun: &goast.Ident{Name: "int64"}, Args: []goast.Expr{arg}},
				&goast.BasicLit{Kind: gotoken.INT, Value: "10"},
			},
		}
		if dstKind == types.ASCII {
			t.addCogImport()
			return &goast.CallExpr{
				Fun: &goast.SelectorExpr{X: &goast.Ident{Name: "cog"}, Sel: &goast.Ident{Name: "ASCII"}},
				Args: []goast.Expr{result},
			}, nil
		}
		return result, nil
	}

	// unsigned integer → string (ascii or utf8)
	if types.IsUint(srcType) && types.IsString(dstType) {
		t.addStdLibImport("strconv")
		result := &goast.CallExpr{
			Fun: &goast.SelectorExpr{X: &goast.Ident{Name: goStdLibAlias("strconv")}, Sel: &goast.Ident{Name: "FormatUint"}},
			Args: []goast.Expr{
				&goast.CallExpr{Fun: &goast.Ident{Name: "uint64"}, Args: []goast.Expr{arg}},
				&goast.BasicLit{Kind: gotoken.INT, Value: "10"},
			},
		}
		if dstKind == types.ASCII {
			t.addCogImport()
			return &goast.CallExpr{
				Fun: &goast.SelectorExpr{X: &goast.Ident{Name: "cog"}, Sel: &goast.Ident{Name: "ASCII"}},
				Args: []goast.Expr{result},
			}, nil
		}
		return result, nil
	}

	// float → string (ascii or utf8)
	if types.IsFloat(srcType) && types.IsString(dstType) {
		t.addStdLibImport("strconv")
		bits := types.Size(srcKind)
		result := &goast.CallExpr{
			Fun: &goast.SelectorExpr{X: &goast.Ident{Name: goStdLibAlias("strconv")}, Sel: &goast.Ident{Name: "FormatFloat"}},
			Args: []goast.Expr{
				arg,
				&goast.BasicLit{Kind: gotoken.CHAR, Value: "'f'"},
				&goast.BasicLit{Kind: gotoken.INT, Value: "-1"},
				&goast.BasicLit{Kind: gotoken.INT, Value: strconv.Itoa(bits)},
			},
		}
		if dstKind == types.ASCII {
			t.addCogImport()
			return &goast.CallExpr{
				Fun: &goast.SelectorExpr{X: &goast.Ident{Name: "cog"}, Sel: &goast.Ident{Name: "ASCII"}},
				Args: []goast.Expr{result},
			}, nil
		}
		return result, nil
	}

	// bool → string (ascii or utf8) — handled above, this is for complex → string fallback too

	// integer ↔ integer (same or wider target) — direct conversion
	if types.IsInt(srcType) && types.IsInt(dstType) {
		srcBits := types.Size(srcKind)
		dstBits := types.Size(dstKind)
		if dstBits >= srcBits {
			return &goast.CallExpr{Fun: dstGoType, Args: []goast.Expr{arg}}, nil
		}
		// narrowing integer: check for overflow
		narrowed, err := t.convertNarrowingInt(arg, dstGoType, srcKind, dstKind)
		if err != nil {
			return nil, err
		}
		return narrowed, nil
	}

	// unsigned ↔ unsigned (same or wider target) — direct conversion
	if types.IsUint(srcType) && types.IsUint(dstType) {
		srcBits := types.Size(srcKind)
		dstBits := types.Size(dstKind)
		if dstBits >= srcBits {
			return &goast.CallExpr{Fun: dstGoType, Args: []goast.Expr{arg}}, nil
		}
		narrowed, err := t.convertNarrowingInt(arg, dstGoType, srcKind, dstKind)
		if err != nil {
			return nil, err
		}
		return narrowed, nil
	}

	// signed ↔ unsigned cross-family: use wider conversion
	if (types.IsInt(srcType) && types.IsUint(dstType)) || (types.IsUint(srcType) && types.IsInt(dstType)) {
		srcBits := types.Size(srcKind)
		dstBits := types.Size(dstKind)
		if dstBits >= srcBits {
			return &goast.CallExpr{Fun: dstGoType, Args: []goast.Expr{arg}}, nil
		}
		narrowed, err := t.convertNarrowingInt(arg, dstGoType, srcKind, dstKind)
		if err != nil {
			return nil, err
		}
		return narrowed, nil
	}

	// integer → float (always safe)
	if types.IsFixed(srcType) && types.IsFloat(dstType) {
		return &goast.CallExpr{Fun: dstGoType, Args: []goast.Expr{arg}}, nil
	}

	// float → integer: check NaN, infinity, fraction
	if types.IsFloat(srcType) && (types.IsInt(dstType) || types.IsUint(dstType)) {
		t.addStdLibImport("math")
		mathAlias := goStdLibAlias("math")
		return &goast.CallExpr{
			Fun: &goast.FuncLit{
				Type: &goast.FuncType{
					Results: &goast.FieldList{
						List: []*goast.Field{{Type: dstGoType}},
					},
				},
				Body: &goast.BlockStmt{
					List: []goast.Stmt{
						&goast.IfStmt{
							Cond: &goast.BinaryExpr{
								X: &goast.CallExpr{
									Fun: &goast.SelectorExpr{X: &goast.Ident{Name: mathAlias}, Sel: &goast.Ident{Name: "IsNaN"}},
									Args: []goast.Expr{arg},
								},
								Op:  gotoken.LOR,
								Y: &goast.BinaryExpr{
									X:  arg,
									Op: gotoken.NEQ,
									Y: &goast.CallExpr{
										Fun: &goast.SelectorExpr{X: &goast.Ident{Name: mathAlias}, Sel: &goast.Ident{Name: "Trunc"}},
										Args: []goast.Expr{arg},
									},
								},
							},
							Body: &goast.BlockStmt{
								List: []goast.Stmt{
									&goast.ReturnStmt{
										Results: []goast.Expr{&goast.CallExpr{Fun: dstGoType, Args: []goast.Expr{&goast.BasicLit{Kind: gotoken.INT, Value: "0"}}}},
									},
								},
							},
						},
						&goast.ReturnStmt{
							Results: []goast.Expr{&goast.CallExpr{Fun: dstGoType, Args: []goast.Expr{arg}}},
						},
					},
				},
			},
		}, nil
	}

	// float → float: wider is direct, narrower is direct Go conversion (may be ±Inf)
	if types.IsFloat(srcType) && types.IsFloat(dstType) {
		return &goast.CallExpr{Fun: dstGoType, Args: []goast.Expr{arg}}, nil
	}

	// complex with non-zero imag → anything → zero value
	// complex with zero imag → number → extract real
	if types.IsComplex(srcType) {
		if types.IsComplex(dstType) {
			// complex → complex: extract real, check imag, convert
			return &goast.CallExpr{
				Fun: &goast.FuncLit{
					Type: &goast.FuncType{
						Results: &goast.FieldList{
							List: []*goast.Field{{Type: dstGoType}},
						},
					},
					Body: &goast.BlockStmt{
						List: []goast.Stmt{
							&goast.IfStmt{
								Cond: &goast.BinaryExpr{
									X: &goast.CallExpr{
										Fun:  &goast.Ident{Name: "imag"},
										Args: []goast.Expr{arg},
									},
									Op:  gotoken.NEQ,
									Y: &goast.BasicLit{Kind: gotoken.FLOAT, Value: "0"},
								},
								Body: &goast.BlockStmt{
									List: []goast.Stmt{
										&goast.ReturnStmt{
											Results: []goast.Expr{&goast.CallExpr{Fun: dstGoType, Args: []goast.Expr{&goast.BasicLit{Kind: gotoken.INT, Value: "0"}}}},
										},
									},
								},
							},
							&goast.ReturnStmt{
								Results: []goast.Expr{&goast.CallExpr{Fun: dstGoType, Args: []goast.Expr{&goast.CallExpr{Fun: &goast.Ident{Name: "real"}, Args: []goast.Expr{arg}}}}},
							},
						},
					},
				},
			}, nil
		}
		if types.IsNumber(dstType) {
			// complex → real/number: extract real
			realExpr := &goast.CallExpr{Fun: &goast.Ident{Name: "real"}, Args: []goast.Expr{arg}}
			checkImag := &goast.BinaryExpr{
				X:  &goast.CallExpr{Fun: &goast.Ident{Name: "imag"}, Args: []goast.Expr{arg}},
				Op: gotoken.NEQ,
				Y:  &goast.BasicLit{Kind: gotoken.FLOAT, Value: "0"},
			}
			return &goast.CallExpr{
				Fun: &goast.FuncLit{
					Type: &goast.FuncType{
						Results: &goast.FieldList{List: []*goast.Field{{Type: dstGoType}}},
					},
					Body: &goast.BlockStmt{
						List: []goast.Stmt{
							&goast.IfStmt{
								Cond: checkImag,
								Body: &goast.BlockStmt{List: []goast.Stmt{&goast.ReturnStmt{Results: []goast.Expr{&goast.CallExpr{Fun: dstGoType, Args: []goast.Expr{&goast.BasicLit{Kind: gotoken.INT, Value: "0"}}}}}}},
							},
							&goast.ReturnStmt{
								Results: []goast.Expr{&goast.CallExpr{Fun: dstGoType, Args: []goast.Expr{realExpr}}},
							},
						},
					},
				},
			}, nil
		}
	}

	// number → complex: fill real, imag zero
	if types.IsNumber(srcType) && types.IsComplex(dstType) {
		srcIsComplex := types.IsComplex(srcType)
		var realPart goast.Expr
		if srcIsComplex {
			realPart = &goast.CallExpr{Fun: &goast.Ident{Name: "real"}, Args: []goast.Expr{arg}}
		} else {
			realPart = arg
		}
		return &goast.CallExpr{
			Fun:  dstGoType,
			Args: []goast.Expr{realPart, &goast.BasicLit{Kind: gotoken.INT, Value: "0"}},
		}, nil
	}

	// bool → string (for ascii) — handled above already, this is catch-all

	// unsupported pair → zero value of B
	return &goast.CallExpr{Fun: dstGoType, Args: []goast.Expr{&goast.BasicLit{Kind: gotoken.INT, Value: "0"}}}, nil
}

// convertNarrowingInt generates overflow-safe narrowing: if T(v) != v { 0 } else { T(v) }
func (t *Transpiler) convertNarrowingInt(arg goast.Expr, dstGoType goast.Expr, srcKind, dstKind types.Kind) (goast.Expr, error) {
	narrowed := &goast.CallExpr{Fun: dstGoType, Args: []goast.Expr{arg}}
	check := &goast.BinaryExpr{
		X:  &goast.CallExpr{Fun: dstGoType, Args: []goast.Expr{arg}},
		Op: gotoken.NEQ,
		Y:  &goast.CallExpr{Fun: dstGoType, Args: []goast.Expr{arg}},
	}
	return &goast.CallExpr{
		Fun: &goast.FuncLit{
			Type: &goast.FuncType{
				Results: &goast.FieldList{List: []*goast.Field{{Type: dstGoType}}},
			},
			Body: &goast.BlockStmt{
				List: []goast.Stmt{
					&goast.IfStmt{
						Cond: check,
						Body: &goast.BlockStmt{List: []goast.Stmt{&goast.ReturnStmt{Results: []goast.Expr{&goast.CallExpr{Fun: dstGoType, Args: []goast.Expr{&goast.BasicLit{Kind: gotoken.INT, Value: "0"}}}}}}},
					},
					&goast.ReturnStmt{Results: []goast.Expr{narrowed}},
				},
			},
		},
	}, nil
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
