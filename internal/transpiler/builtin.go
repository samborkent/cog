package transpiler

import (
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

		ifType, err := t.convertType(node.Arguments[1].Type())
		if err != nil {
			return nil, fmt.Errorf("converting @if builtin type: %w", err)
		}

		t.addBuiltinImport()

		return component.BuiltinIf(ifType, args...), nil
	case BuiltinMap:
		panic("TODO: implement @map transpilation")
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
		panic("TODO: implement @ptr transpilation")
	case BuiltinSet:
		panic("TODO: implement @set transpilation")
	case BuiltinSlice:
		panic("TODO: implement @slice transpilation")
	default:
		return nil, fmt.Errorf("unknown builtin function '%s'", node.Name)
	}
}
