package parser

import (
	"context"
	"fmt"

	"github.com/samborkent/cog/internal/ast"
	"github.com/samborkent/cog/internal/tokens"
	"github.com/samborkent/cog/internal/types"
)

type BuiltinParser func(ctx context.Context, t tokens.Token, tokenType types.Type) ast.ExprIndex

func (p *Parser) parseBuiltinIf(ctx context.Context, t tokens.Token, tokenType types.Type) ast.ExprIndex {
	var typArgs []types.Type

	if p.this().Type == tokens.LT {
		typArgs = p.parseTypeArguments(ctx)
		if typArgs == nil {
			return ast.ZeroExprIndex
		}

		if len(typArgs) > 2 {
			p.error(p.this(), "@if expected at most 2 type arguments", "parseBuiltinIf")
			return ast.ZeroExprIndex
		}

		// If a type argument if provided, check it's the same as the expected type if any.
		if len(typArgs) >= 1 && tokenType.Kind() != types.Invalid && typArgs[0].Kind() != tokenType.Kind() {
			p.error(p.this(), "@if type argument does not match expected type", "parseBuiltinIf")
			return ast.ZeroExprIndex
		}

		if len(typArgs) == 2 && typArgs[1].Kind() != types.Bool {
			p.error(p.this(), "@if second type argument must be of type ~bool", "parseBuiltinIf")
			return ast.ZeroExprIndex
		}

		tokenType = typArgs[0]
	}

	if p.this().Type != tokens.LParen {
		p.error(p.this(), "expected '(' after @if", "parseIf")
		return ast.ZeroExprIndex
	}

	p.advance("parseIf (") // consume (

	// condition := p.expression(ctx, types.Basics[types.Bool])
	condition := p.expression(ctx, types.None)
	if condition == ast.ZeroExprIndex {
		return ast.ZeroExprIndex
	}

	if p.this().Type != tokens.Comma {
		p.error(p.this(), "expected ',' after condition in @if", "parseIf")
		return ast.ZeroExprIndex
	}

	p.advance("parseIf , condition") // consume ,

	thenExpr := p.expression(ctx, tokenType)
	if thenExpr == ast.ZeroExprIndex {
		return ast.ZeroExprIndex
	}

	thenType := p.ast.Expr(thenExpr).Type()

	args := []ast.ExprIndex{condition, thenExpr}

	if p.this().Type == tokens.Comma {
		p.advance("parseIf , then") // consume ,

		elseExpr := p.expression(ctx, tokenType)
		if elseExpr == ast.ZeroExprIndex {
			return ast.ZeroExprIndex
		}

		elseType := p.ast.Expr(elseExpr).Type()

		if thenType.Kind() != elseType.Kind() {
			p.error(t, fmt.Sprintf("type mismatch in @if branches: then is %q, else is %q", thenType, elseType), "parseIf")
			return ast.ZeroExprIndex
		}

		args = append(args, elseExpr)
	}

	if p.this().Type != tokens.RParen {
		p.error(p.this(), "expected ')' after else expression in @if", "parseIf")
		return ast.ZeroExprIndex
	}

	p.advance("parseIf )") // consume ')'

	return p.ast.NewBuiltin(t, "if", typArgs, args, thenType)
}

func (p *Parser) parseBuiltinMap(ctx context.Context, t tokens.Token, tokenType types.Type) ast.ExprIndex {
	if tokenType.Kind() != types.Invalid && tokenType.Kind() != types.MapKind {
		// If type is supplied, check if it's a map.
		p.error(p.this(), "expected map type", "parseBuiltinMap")
		return ast.ZeroExprIndex
	}

	typArgs := p.parseTypeArguments(ctx)
	if typArgs == nil {
		return ast.ZeroExprIndex
	}

	if len(typArgs) < 2 || len(typArgs) > 3 {
		p.error(p.this(), "@map wrong number of type arguments", "parseBuiltinMap")
		return ast.ZeroExprIndex
	}

	if tokenType.Kind() != types.Invalid {
		mapType, ok := tokenType.Underlying().(*types.Map)
		if !ok {
			p.error(p.this(), "unable to cast supplied map type", "parseBuiltinMap")
			return ast.ZeroExprIndex
		}

		if mapType.Key.Kind() != typArgs[0].Kind() {
			p.error(p.this(), "type mismatch in @map key", "parseBuiltinMap")
			return ast.ZeroExprIndex
		}

		if mapType.Value.Kind() != typArgs[1].Kind() {
			p.error(p.this(), "type mismatch in @map value", "parseBuiltinMap")
			return ast.ZeroExprIndex
		}
	}

	if p.this().Type != tokens.LParen {
		p.error(p.this(), "expected '(' after @map", "parseBuiltinMap")
		return ast.ZeroExprIndex
	}

	p.advance("parseBuiltinMap (") // consume (

	args := make([]ast.ExprIndex, 0, 1)

	if p.this().Type != tokens.RParen {
		var capType types.Type = types.None

		if len(typArgs) == 3 {
			capType = typArgs[2]
		}

		capArg := p.expression(ctx, capType)
		if capArg == ast.ZeroExprIndex {
			return ast.ZeroExprIndex
		}

		args = append(args, capArg)
	}

	if p.this().Type != tokens.RParen {
		p.error(p.this(), "expected ')' after argument in @map", "parseBuiltinMap")
		return ast.ZeroExprIndex
	}

	p.advance("parseBuiltinMap )") // consume ')'

	return p.ast.NewBuiltin(t, "map", typArgs, args, &types.Map{Key: typArgs[0], Value: typArgs[1]})
}

func (p *Parser) parseBuiltinPrint(ctx context.Context, t tokens.Token, tokenType types.Type) ast.ExprIndex {
	if p.this().Type != tokens.LParen {
		p.error(p.this(), "expected '(' after @print", "parsePrint")
		return ast.ZeroExprIndex
	}

	p.advance("parsePrint (") // consume (

	if p.this().Type == tokens.RParen {
		p.error(p.this(), "expected argument in @print", "parsePrint")
		return ast.ZeroExprIndex
	}

	arg := p.expression(ctx, tokenType)
	if arg == ast.ZeroExprIndex {
		return ast.ZeroExprIndex
	}

	// TODO: implement string formatting

	if p.this().Type != tokens.RParen {
		p.error(p.this(), "expected ')' after argument in @print", "parsePrint")
		return ast.ZeroExprIndex
	}

	p.advance("parsePrint )") // consume ')'

	// TODO: handle ascii / utf8 type args.
	return p.ast.NewBuiltin(t, "print", nil, []ast.ExprIndex{arg}, types.None)
}

// TODO: possibly remove this, why do we need reference allocator? maybe better that is works like `@ref<T any>(x T) &T`, but not needed if we allow `&literal`
func (p *Parser) parseBuiltinRef(ctx context.Context, t tokens.Token, tokenType types.Type) ast.ExprIndex {
	if tokenType.Kind() != types.Invalid && tokenType.Kind() != types.ReferenceKind {
		// If type is supplied, check if it's a pointer.
		p.error(p.this(), "expected pointer type", "parseBuiltinPtr")
		return ast.ZeroExprIndex
	}

	typArgs := p.parseTypeArguments(ctx)
	if typArgs == nil {
		return ast.ZeroExprIndex
	}

	if len(typArgs) != 1 {
		p.error(p.this(), "@ref requires one type argument", "parseBuiltinRef")
		return ast.ZeroExprIndex
	}

	if tokenType.Kind() != types.Invalid {
		refType, ok := tokenType.Underlying().(*types.Reference)
		if !ok {
			p.error(p.this(), "unable to cast supplied reference type", "parseBuiltinRef")
			return ast.ZeroExprIndex
		}

		if refType.Value.Kind() != typArgs[0].Kind() {
			p.error(p.this(), "type mismatch in @ref type", "parseBuiltinRef")
			return ast.ZeroExprIndex
		}
	}

	if p.this().Type != tokens.LParen {
		p.error(p.this(), "expected '(' after @ref", "parseBuiltinRef")
		return ast.ZeroExprIndex
	}

	p.advance("parseBuiltinRef (") // consume (

	if p.this().Type != tokens.RParen {
		p.error(p.this(), "expected ')' after argument in @ref", "parseBuiltinPtr")
		return ast.ZeroExprIndex
	}

	p.advance("parseBuiltinRef )") // consume ')'

	return p.ast.NewBuiltin(t, "ref", typArgs, nil, &types.Reference{Value: typArgs[0]})
}

func (p *Parser) parseBuiltinSet(ctx context.Context, t tokens.Token, tokenType types.Type) ast.ExprIndex {
	if tokenType.Kind() != types.Invalid && tokenType.Kind() != types.SetKind {
		// If type is supplied, check if it's a set.
		p.error(p.this(), "expected set type", "parseBuiltinSet")
		return ast.ZeroExprIndex
	}

	typArgs := p.parseTypeArguments(ctx)
	if typArgs == nil {
		return ast.ZeroExprIndex
	}

	if len(typArgs) == 0 || len(typArgs) > 2 {
		p.error(p.this(), "@set wrong number of type arguments", "parseBuiltinSet")
		return ast.ZeroExprIndex
	}

	if tokenType.Kind() != types.Invalid {
		setType, ok := tokenType.Underlying().(*types.Set)
		if !ok {
			p.error(p.this(), "unable to cast supplied set type", "parseBuiltinSet")
			return ast.ZeroExprIndex
		}

		if setType.Element.Kind() != typArgs[0].Kind() {
			p.error(p.this(), "type mismatch in @set element", "parseBuiltinSet")
			return ast.ZeroExprIndex
		}
	}

	if p.this().Type != tokens.LParen {
		p.error(p.this(), "expected '(' after @set", "parseBuiltinSet")
		return ast.ZeroExprIndex
	}

	p.advance("parseBuiltinSet (") // consume (

	args := make([]ast.ExprIndex, 0, 1)

	if p.this().Type != tokens.RParen {
		var capType types.Type = types.None

		if len(typArgs) > 1 {
			capType = typArgs[1]
		}

		capArg := p.expression(ctx, capType)
		if capArg == ast.ZeroExprIndex {
			return ast.ZeroExprIndex
		}

		args = append(args, capArg)
	}

	if p.this().Type != tokens.RParen {
		p.error(p.this(), "expected ')' after argument in @set", "parseBuiltinSet")
		return ast.ZeroExprIndex
	}

	p.advance("parseBuiltinSet )") // consume ')'

	return p.ast.NewBuiltin(t, "set", typArgs, args, &types.Set{Element: typArgs[0]})
}

func (p *Parser) parseBuiltinSlice(ctx context.Context, t tokens.Token, tokenType types.Type) ast.ExprIndex {
	if tokenType.Kind() != types.Invalid && tokenType.Kind() != types.SliceKind {
		// If type is supplied, check if it's a slice.
		p.error(p.this(), "expected slice type", "parseBuiltinSlice")
		return ast.ZeroExprIndex
	}

	typArgs := p.parseTypeArguments(ctx)
	if typArgs == nil {
		return ast.ZeroExprIndex
	}

	if len(typArgs) < 1 || len(typArgs) > 2 {
		p.error(p.this(), "@slice requires one or two type arguments", "parseBuiltinSlice")
		return ast.ZeroExprIndex
	}

	if tokenType.Kind() != types.Invalid {
		sliceType, ok := tokenType.Underlying().(*types.Slice)
		if !ok {
			p.error(p.this(), "unable to cast supplied slice type", "parseBuiltinSlice")
			return ast.ZeroExprIndex
		}

		if sliceType.Element.Kind() != typArgs[0].Kind() {
			p.error(p.this(), "type mismatch in @slice element", "parseBuiltinSlice")
			return ast.ZeroExprIndex
		}
	}

	if p.this().Type != tokens.LParen {
		p.error(p.this(), "expected '(' after @slice", "parseBuiltinSlice")
		return ast.ZeroExprIndex
	}

	p.advance("parseBuiltinSlice (") // consume (

	var lenType types.Type = types.None

	if len(typArgs) == 2 {
		lenType = typArgs[1]
	}

	lenArg := p.expression(ctx, lenType)
	if lenArg == ast.ZeroExprIndex {
		return ast.ZeroExprIndex
	}

	args := make([]ast.ExprIndex, 1, 2)
	args[0] = lenArg

	if p.this().Type == tokens.Comma {
		p.advance("parseBuiltinSlice ,") // consume ','

		capArg := p.expression(ctx, lenType)
		if capArg == ast.ZeroExprIndex {
			return ast.ZeroExprIndex
		}

		args = append(args, capArg)
	}

	if p.this().Type != tokens.RParen {
		p.error(p.this(), "expected ')' after argument in @slice", "parseBuiltinSlice")
		return ast.ZeroExprIndex
	}

	p.advance("parseBuiltinSlice )") // consume ')'

	return p.ast.NewBuiltin(t, "slice", typArgs, args, &types.Slice{Element: typArgs[0]})
}

func (p *Parser) parseBuiltinCast(ctx context.Context, t tokens.Token, tokenType types.Type) ast.ExprIndex {
	typArgs := p.parseTypeArguments(ctx)
	if typArgs == nil {
		return ast.ZeroExprIndex
	}

	if len(typArgs) == 0 || len(typArgs) > 2 {
		p.error(t, "@cast requires 1 or 2 type arguments", "parseBuiltinCast")
		return ast.ZeroExprIndex
	}

	targetType := typArgs[0]
	targetKind := targetType.Kind()

	// Validate target type is castable.
	if !types.IsBasic(targetType) {
		p.error(t, fmt.Sprintf("@cast target type %q is not a basic type", targetType), "parseBuiltinCast")
		return ast.ZeroExprIndex
	}

	if p.this().Type != tokens.LParen {
		p.error(p.this(), "expected '(' after @cast", "parseBuiltinCast")
		return ast.ZeroExprIndex
	}

	p.advance("parseBuiltinCast (") // consume (

	// If the second type argument (source type) is provided, use it to
	// guide type inference for the argument expression (e.g. untyped literals).
	var argType types.Type = types.None
	if len(typArgs) == 2 {
		argType = typArgs[1]
	}

	arg := p.expression(ctx, argType)
	if arg == ast.ZeroExprIndex {
		return ast.ZeroExprIndex
	}

	if p.this().Type != tokens.RParen {
		p.error(p.this(), "expected ')' after argument in @cast", "parseBuiltinCast")
		return ast.ZeroExprIndex
	}

	p.advance("parseBuiltinCast )") // consume )

	// TODO: check if we can just use argType here.
	sourceType := p.ast.Expr(arg).Type()
	sourceKind := sourceType.Kind()

	// Handle ascii -> utf8 special case.
	if sourceKind == types.ASCII && targetKind == types.UTF8 {
		return p.ast.NewBuiltin(t, "cast", typArgs, []ast.ExprIndex{arg}, targetType)
	}

	// Validate source type is castable.
	if !types.IsBasic(sourceType) {
		p.error(t, fmt.Sprintf("@cast source type %q is not a basic type", sourceType), "parseBuiltinCast")
		return ast.ZeroExprIndex
	}

	// Reject any remaining string casts (only ascii -> utf8 is allowed above).
	if types.IsString(sourceType) || types.IsString(targetType) {
		p.error(t, fmt.Sprintf("@cast cannot cast between %q and %q", sourceType, targetType), "parseBuiltinCast")
		return ast.ZeroExprIndex
	}

	// Same type is not allowed.
	if sourceKind == targetKind {
		p.error(t, fmt.Sprintf("@cast source and target types are the same: %q", sourceType), "parseBuiltinCast")
		return ast.ZeroExprIndex
	}

	// Validate second type argument matches source if provided.
	if len(typArgs) == 2 {
		if typArgs[1].Kind() != sourceKind {
			p.error(t, fmt.Sprintf("@cast second type argument %q does not match source type %q", typArgs[1], sourceType), "parseBuiltinCast")
			return ast.ZeroExprIndex
		}
	}

	// Validate bit size: source must be <= target.
	srcBits := types.Size(sourceKind)
	dstBits := types.Size(targetKind)

	if srcBits > dstBits {
		p.error(t, fmt.Sprintf("@cast cannot narrow from %d-bit %q to %d-bit %q", srcBits, sourceType, dstBits, targetType), "parseBuiltinCast")
		return ast.ZeroExprIndex
	}

	return p.ast.NewBuiltin(t, "cast", typArgs, []ast.ExprIndex{arg}, targetType)
}
