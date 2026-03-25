package transpiler

import (
	"errors"
	"fmt"
	goast "go/ast"
	"math"

	"github.com/samborkent/cog/internal/ast"
	"github.com/samborkent/cog/internal/transpiler/component"
	"github.com/samborkent/cog/internal/types"
)

type Builtins string

const (
	BuiltinIf    Builtins = "if"
	BuiltinMap   Builtins = "map"
	BuiltinPrint Builtins = "print"
	BuiltinPtr   Builtins = "ptr"
	BuiltinSet   Builtins = "set"
	BuiltinSlice Builtins = "slice"
)

func (t *Transpiler) convertBuiltin(node *ast.Builtin) (*goast.CallExpr, error) {
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
			return nil, fmt.Errorf("@set expects 2 or 3 type arguments, got %d", len(node.TypeArguments))
		}

		if len(node.Arguments) > 1 {
			return nil, fmt.Errorf("@set expects at most 1 argument, got %d", len(node.Arguments))
		}

		keyType, err := t.convertType(node.TypeArguments[0])
		if err != nil {
			return nil, fmt.Errorf("converting @map builtin key type: %w", err)
		}

		valueType, err := t.convertType(node.TypeArguments[1])
		if err != nil {
			return nil, fmt.Errorf("converting @map builtin value type: %w", err)
		}

		var capType goast.Expr

		if len(node.TypeArguments) == 3 {
			capType, err = t.convertType(node.TypeArguments[2])
			if err != nil {
				return nil, fmt.Errorf("converting @map capacity type: %w", err)
			}
		}

		t.addBuiltinImport()

		var size goast.Expr

		if len(node.Arguments) == 1 {
			size, err = t.convertExpr(node.Arguments[0])
			if err != nil {
				return nil, fmt.Errorf("converting @map builtin size argument: %w", err)
			}

			switch n := node.Arguments[0].(type) {
			case *ast.Prefix:
				return nil, errors.New("@map capacity must be positive")
			case *ast.Int64Literal:
				typ := "uint64"

				if n.Value < 0 {
				} else if n.Value <= math.MaxUint8 {
					typ = "uint8"
				} else if n.Value <= math.MaxUint16 {
					typ = "uint16"
				} else if n.Value <= math.MaxUint32 {
					typ = "uint32"
				}

				size = &goast.CallExpr{
					Fun:  &goast.Ident{Name: typ},
					Args: []goast.Expr{size},
				}
			}
		}

		return component.BuiltinMap(keyType, valueType, capType, size), nil
	case BuiltinPrint:
		if len(node.Arguments) != 1 {
			return nil, fmt.Errorf("print expects 1 argument, got %d", len(node.Arguments))
		}

		arg, err := t.convertExpr(node.Arguments[0])
		if err != nil {
			return nil, fmt.Errorf("converting print argument: %w", err)
		}

		// Print underlying value of enum instead of enum itself.
		if node.Arguments[0].Type().Underlying().Kind() == types.EnumKind {
			enumType, ok := node.Arguments[0].Type().(*types.Alias)
			if !ok {
				panic("unable to cast enum to alias")
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
			return nil, fmt.Errorf("@ptr expects 1 type arguments, got %d", len(node.TypeArguments))
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

		var capType goast.Expr

		if len(node.TypeArguments) == 2 {
			capType, err = t.convertType(node.TypeArguments[1])
			if err != nil {
				return nil, fmt.Errorf("converting @set capacity type: %w", err)
			}
		}

		t.addBuiltinImport()

		var size goast.Expr

		if len(node.Arguments) == 1 {
			size, err = t.convertExpr(node.Arguments[0])
			if err != nil {
				return nil, fmt.Errorf("converting @set builtin size argument: %w", err)
			}

			switch n := node.Arguments[0].(type) {
			case *ast.Prefix:
				return nil, errors.New("@set capacity must be positive")
			case *ast.Int64Literal:
				typ := "uint64"

				if n.Value < 0 {
				} else if n.Value <= math.MaxUint8 {
					typ = "uint8"
				} else if n.Value <= math.MaxUint16 {
					typ = "uint16"
				} else if n.Value <= math.MaxUint32 {
					typ = "uint32"
				}

				size = &goast.CallExpr{
					Fun:  &goast.Ident{Name: typ},
					Args: []goast.Expr{size},
				}
			}
		}

		return component.BuiltinSet(keyType, capType, size), nil
	case BuiltinSlice:
		if len(node.TypeArguments) < 1 || len(node.TypeArguments) > 2 {
			return nil, fmt.Errorf("@slice expects 1 or 2 type arguments, got %d", len(node.TypeArguments))
		}

		if len(node.Arguments) < 1 {
			return nil, fmt.Errorf("@slice expects at lest 1 argument, got %d", len(node.Arguments))
		}

		elemType, err := t.convertType(node.TypeArguments[0])
		if err != nil {
			return nil, fmt.Errorf("converting @slice element type: %w", err)
		}

		var lenType goast.Expr

		if len(node.TypeArguments) == 2 {
			lenType, err = t.convertType(node.TypeArguments[1])
			if err != nil {
				return nil, fmt.Errorf("converting @slice length type: %w", err)
			}
		}

		length, err := t.convertExpr(node.Arguments[0])
		if err != nil {
			return nil, fmt.Errorf("converting @slice length argument: %w", err)
		}

		switch n := node.Arguments[0].(type) {
		case *ast.Prefix:
			return nil, errors.New("@slice length must be positive")
		case *ast.Int64Literal:
			typ := "uint64"

			if n.Value < 0 {
			} else if n.Value <= math.MaxUint8 {
				typ = "uint8"
			} else if n.Value <= math.MaxUint16 {
				typ = "uint16"
			} else if n.Value <= math.MaxUint32 {
				typ = "uint32"
			}

			length = &goast.CallExpr{
				Fun:  &goast.Ident{Name: typ},
				Args: []goast.Expr{length},
			}
		}

		var capacity goast.Expr

		if len(node.Arguments) == 2 {
			capacity, err = t.convertExpr(node.Arguments[1])
			if err != nil {
				return nil, fmt.Errorf("converting @slice capacity argument: %w", err)
			}

			switch n := node.Arguments[1].(type) {
			case *ast.Prefix:
				return nil, errors.New("@slice capacity must be positive")
			case *ast.Int64Literal:
				typ := "uint64"

				if n.Value < 0 {
				} else if n.Value <= math.MaxUint8 {
					typ = "uint8"
				} else if n.Value <= math.MaxUint16 {
					typ = "uint16"
				} else if n.Value <= math.MaxUint32 {
					typ = "uint32"
				}

				capacity = &goast.CallExpr{
					Fun:  &goast.Ident{Name: typ},
					Args: []goast.Expr{capacity},
				}
			}
		}

		return component.BuiltinSlice(elemType, lenType, length, capacity), nil
	default:
		return nil, fmt.Errorf("unknown builtin function '%s'", node.Name)
	}
}
