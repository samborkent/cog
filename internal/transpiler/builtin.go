package transpiler

import (
	"errors"
	"fmt"
	goast "go/ast"

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
		if node.Arguments[0].Type().Underlying().Kind() == types.EnumKind {
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
	default:
		return nil, fmt.Errorf("unknown builtin function '%s'", node.Name)
	}
}
