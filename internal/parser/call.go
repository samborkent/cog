package parser

import (
	"context"
	"fmt"

	"github.com/samborkent/cog/internal/ast"
	"github.com/samborkent/cog/internal/tokens"
	"github.com/samborkent/cog/internal/types"
)

// TODO: pre-allocate based on heuristics
const argumentPreallocationSize = 2

func (p *Parser) parseCallArguments(ctx context.Context, procType *types.Procedure) []ast.ExprIndex {
	if p.this().Type != tokens.LParen {
		p.error(p.this(), "expected '(' after call identifier", "parseCallArguments")
		return nil
	}

	p.advance("parseCallArguments (") // consume '('

	if p.this().Type == tokens.RParen {
		p.advance("parseCallArguments )") // consume ')'
		return []ast.ExprIndex{}
	}

	args := make([]ast.ExprIndex, 0, argumentPreallocationSize)

	for i := 0; p.this().Type != tokens.RParen && p.this().Type != tokens.EOF; i++ {
		if ctx.Err() != nil {
			return nil
		}

		var arg ast.ExprIndex

		if procType == nil {
			arg = p.expression(ctx, types.None)
			if arg == ast.ZeroExprIndex {
				return nil
			}
		} else {
			if i >= len(procType.Parameters) {
				p.error(p.this(), "too many arguments in function call", "parseCallArguments")
				return nil
			}

			paramType := procType.Parameters[i].Type

			// When the parameter type is a type param alias, let the expression
			// infer its own type (like an untyped declaration).
			if alias, ok := paramType.(*types.Alias); ok && alias.IsTypeParam() {
				paramType = types.None
			}

			arg = p.expression(ctx, paramType)
			if arg == ast.ZeroExprIndex {
				return nil
			}
		}

		args = append(args, arg)

		if p.this().Type == tokens.Comma {
			p.advance("parseCallArguments ,") // consume ','
		}
	}

	if p.this().Type != tokens.RParen {
		p.error(p.this(), "expected ')' after function call arguments", "parseCallArguments")
		return nil
	}

	p.advance("parseCallArguments )") // consume ')'

	return args
}

// inferTypeArgs infers type arguments for a generic procedure call from the
// actual argument types. Returns the inferred type args (ordered by TypeParams)
// and the substituted return type. Reports parser errors on failure.
func (p *Parser) inferTypeArgs(
	procType *types.Procedure,
	args []ast.ExprIndex,
) ([]types.Type, types.Type) {
	argMap := make(map[string]types.Type, len(procType.TypeParams))

	// Match each argument to its parameter's type param.
	for i, param := range procType.Parameters {
		if i >= len(args) {
			break
		}

		tp, ok := param.Type.(*types.Alias)
		if !ok || !tp.IsTypeParam() {
			continue
		}

		argType := p.ast.Expr(args[i]).Type()

		if existing, ok := argMap[tp.Name]; ok {
			// Already inferred — check consistency.
			if !types.Equal(existing, argType) {
				p.error(p.this(), fmt.Sprintf(
					"conflicting types for type parameter %q: %s vs %s",
					tp.Name, existing, argType), "inferTypeArgs")

				return nil, nil
			}

			continue
		}

		// Validate constraint satisfaction.
		if !tp.SatisfiedBy(argType) {
			p.error(p.this(), fmt.Sprintf(
				"type %q does not satisfy constraint %q for parameter %q",
				argType, tp.ConstraintString(), tp.Name), "inferTypeArgs")

			return nil, nil
		}

		argMap[tp.Name] = argType
	}

	// Ensure all type params were inferred.
	typeArgs := make([]types.Type, len(procType.TypeParams))
	for i, tp := range procType.TypeParams {
		concrete, ok := argMap[tp.Name]
		if !ok {
			p.error(p.this(), fmt.Sprintf(
				"cannot infer type parameter %q from arguments", tp.Name), "inferTypeArgs")

			return nil, nil
		}

		typeArgs[i] = concrete
	}

	// Substitute return type.
	var returnType types.Type
	if procType.ReturnType != nil {
		returnType = types.SubstituteType(procType.ReturnType, argMap)
	}

	return typeArgs, returnType
}

// validateExplicitTypeArgs checks explicit type arguments against the procedure's
// type params. Returns the substituted return type. Reports errors on failure.
func (p *Parser) validateExplicitTypeArgs(
	procType *types.Procedure,
	typeArgs []types.Type,
	args []ast.ExprIndex,
) types.Type {
	if len(typeArgs) != len(procType.TypeParams) {
		p.error(p.this(), fmt.Sprintf(
			"wrong number of type arguments: expected %d, got %d",
			len(procType.TypeParams), len(typeArgs)), "validateExplicitTypeArgs")

		return nil
	}

	argMap := make(map[string]types.Type, len(typeArgs))

	for i, tp := range procType.TypeParams {
		if !tp.SatisfiedBy(typeArgs[i]) {
			p.error(p.this(), fmt.Sprintf(
				"type argument %q does not satisfy constraint %q for parameter %q",
				typeArgs[i], tp.ConstraintString(), tp.Name), "validateExplicitTypeArgs")

			return nil
		}

		argMap[tp.Name] = typeArgs[i]
	}

	// Validate argument types match the substituted parameter types.
	for i, param := range procType.Parameters {
		if i >= len(args) {
			break
		}

		expectedType := types.SubstituteType(param.Type, argMap)
		argType := p.ast.Expr(args[i]).Type()

		if !types.Equal(expectedType, argType) && !types.AssignableTo(argType, expectedType) {
			p.error(p.this(), fmt.Sprintf(
				"argument type %q does not match parameter type %q (after substitution)",
				argType, expectedType), "validateExplicitTypeArgs")

			return nil
		}
	}

	if procType.ReturnType != nil {
		return types.SubstituteType(procType.ReturnType, argMap)
	}

	return nil
}
