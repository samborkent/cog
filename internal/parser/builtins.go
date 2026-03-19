package parser

import (
	"context"
	"fmt"

	"github.com/samborkent/cog/internal/ast"
	"github.com/samborkent/cog/internal/tokens"
	"github.com/samborkent/cog/internal/types"
)

type BuiltinParser func(ctx context.Context, t tokens.Token, tokenType types.Type) *ast.Builtin

func (p *Parser) parseBuiltinIf(ctx context.Context, t tokens.Token, tokenType types.Type) *ast.Builtin {
	var typArgs []types.Type

	if p.this().Type == tokens.LT {
		typArgs = p.parseTypeArguments(ctx)
		if typArgs == nil {
			return nil
		}

		if len(typArgs) > 2 {
			p.error(p.this(), "@if expected at most 2 type arguments", "parseBuiltinIf")
			return nil
		}

		// If a type argument if provided, check it's the same as the expected type if any.
		if len(typArgs) >= 1 && tokenType.Kind() != types.Invalid && typArgs[0].Kind() != tokenType.Kind() {
			p.error(p.this(), "@if type argument does not match expected type", "parseBuiltinIf")
			return nil
		}

		if len(typArgs) == 2 && typArgs[1].Kind() != types.Bool {
			p.error(p.this(), "@if second type argument must be of type ~bool", "parseBuiltinIf")
			return nil
		}

		tokenType = typArgs[0]
	}

	if p.this().Type != tokens.LParen {
		p.error(p.this(), "expected '(' after @if", "parseIf")
		return nil
	}

	p.advance("parseIf (") // consume (

	// condition := p.expression(ctx, types.Basics[types.Bool])
	condition := p.expression(ctx, types.None)
	if condition == nil {
		return nil
	}

	if p.this().Type != tokens.Comma {
		p.error(p.this(), "expected ',' after condition in @if", "parseIf")
		return nil
	}

	p.advance("parseIf , condition") // consume ,

	thenExpr := p.expression(ctx, tokenType)
	if thenExpr == nil {
		return nil
	}

	args := []ast.Expression{condition, thenExpr}

	if p.this().Type == tokens.Comma {
		p.advance("parseIf , then") // consume ,

		elseExpr := p.expression(ctx, tokenType)
		if elseExpr == nil {
			return nil
		}

		if thenExpr.Type().Kind() != elseExpr.Type().Kind() {
			p.error(t, fmt.Sprintf("type mismatch in @if branches: then is %q, else is %q", thenExpr.Type(), elseExpr.Type()), "parseIf")
			return nil
		}

		args = append(args, elseExpr)
	}

	if p.this().Type != tokens.RParen {
		p.error(p.this(), "expected ')' after else expression in @if", "parseIf")
		return nil
	}

	p.advance("parseIf )") // consume ')'

	return &ast.Builtin{
		Token:         t,
		Name:          "if",
		TypeArguments: typArgs,
		Arguments:     args,
		ReturnType:    thenExpr.Type(),
	}
}

func (p *Parser) parseBuiltinMap(ctx context.Context, t tokens.Token, tokenType types.Type) *ast.Builtin {
	if tokenType.Kind() != types.Invalid && tokenType.Kind() != types.MapKind {
		// If type is supplied, check if it's a map.
		p.error(p.this(), "expected map type", "parseBuiltinMap")
		return nil
	}

	typArgs := p.parseTypeArguments(ctx)
	if typArgs == nil {
		return nil
	}

	if len(typArgs) < 2 || len(typArgs) > 3 {
		p.error(p.this(), "@map wrong number of type arguments", "parseBuiltinMap")
		return nil
	}

	if tokenType.Kind() != types.Invalid {
		mapType, ok := tokenType.Underlying().(*types.Map)
		if !ok {
			p.error(p.this(), "unable to cast supplied map type", "parseBuiltinMap")
			return nil
		}

		if mapType.Key.Kind() != typArgs[0].Kind() {
			p.error(p.this(), "type mismatch in @map key", "parseBuiltinMap")
			return nil
		}

		if mapType.Value.Kind() != typArgs[1].Kind() {
			p.error(p.this(), "type mismatch in @map value", "parseBuiltinMap")
			return nil
		}
	}

	if p.this().Type != tokens.LParen {
		p.error(p.this(), "expected '(' after @map", "parseBuiltinMap")
		return nil
	}

	p.advance("parseBuiltinMap (") // consume (

	var args []ast.Expression

	if p.this().Type != tokens.RParen {
		var capType types.Type = types.None

		if len(typArgs) == 3 {
			capType = typArgs[2]
		}

		capArg := p.expression(ctx, capType)
		if capArg == nil {
			return nil
		}

		args = append(args, capArg)
	}

	if p.this().Type != tokens.RParen {
		p.error(p.this(), "expected ')' after argument in @map", "parseBuiltinMap")
		return nil
	}

	p.advance("parseBuiltinMap )") // consume ')'

	return &ast.Builtin{
		Token:         t,
		Name:          "map",
		TypeArguments: typArgs,
		Arguments:     args,
		ReturnType:    &types.Map{Key: typArgs[0], Value: typArgs[1]},
	}
}

func (p *Parser) parseBuiltinPrint(ctx context.Context, t tokens.Token, tokenType types.Type) *ast.Builtin {
	if p.this().Type != tokens.LParen {
		p.error(p.this(), "expected '(' after @print", "parsePrint")
		return nil
	}

	p.advance("parsePrint (") // consume (

	arg := p.expression(ctx, tokenType)
	if arg == nil {
		return nil
	}

	// TODO: implement string formatting

	if p.this().Type != tokens.RParen {
		p.error(p.this(), "expected ')' after argument in @print", "parsePrint")
		return nil
	}

	p.advance("parsePrint )") // consume ')'

	return &ast.Builtin{
		Token:      t,
		Name:       "print",
		ReturnType: types.None,
		Arguments:  []ast.Expression{arg},
	}
}

func (p *Parser) parseBuiltinPtr(ctx context.Context, t tokens.Token, tokenType types.Type) *ast.Builtin {
	if tokenType.Kind() != types.Invalid && tokenType.Kind() != types.PointerKind {
		// If type is supplied, check if it's a pointer.
		p.error(p.this(), "expected pointer type", "parseBuiltinPtr")
		return nil
	}

	typArgs := p.parseTypeArguments(ctx)
	if typArgs == nil {
		return nil
	}

	if len(typArgs) != 1 {
		p.error(p.this(), "@ptr requires one type argument", "parseBuiltinPtr")
		return nil
	}

	if tokenType.Kind() != types.Invalid {
		ptrType, ok := tokenType.Underlying().(*types.Pointer)
		if !ok {
			p.error(p.this(), "unable to cast supplied pointer type", "parseBuiltinPtr")
			return nil
		}

		if ptrType.Value.Kind() != typArgs[0].Kind() {
			p.error(p.this(), "type mismatch in @ptr type", "parseBuiltinPtr")
			return nil
		}
	}

	if p.this().Type != tokens.LParen {
		p.error(p.this(), "expected '(' after @ptr", "parseBuiltinPtr")
		return nil
	}

	p.advance("parseBuiltinPtr (") // consume (

	if p.this().Type != tokens.RParen {
		p.error(p.this(), "expected ')' after argument in @ptr", "parseBuiltinPtr")
		return nil
	}

	p.advance("parseBuiltinPtr )") // consume ')'

	return &ast.Builtin{
		Token:         t,
		Name:          "ptr",
		TypeArguments: typArgs,
		ReturnType:    &types.Pointer{Value: typArgs[0]},
	}
}

func (p *Parser) parseBuiltinSet(ctx context.Context, t tokens.Token, tokenType types.Type) *ast.Builtin {
	if tokenType.Kind() != types.Invalid && tokenType.Kind() != types.SetKind {
		// If type is supplied, check if it's a set.
		p.error(p.this(), "expected set type", "parseBuiltinSet")
		return nil
	}

	typArgs := p.parseTypeArguments(ctx)
	if typArgs == nil {
		return nil
	}

	if len(typArgs) == 0 || len(typArgs) > 2 {
		p.error(p.this(), "@set wrong number of type arguments", "parseBuiltinSet")
		return nil
	}

	if tokenType.Kind() != types.Invalid {
		setType, ok := tokenType.Underlying().(*types.Set)
		if !ok {
			p.error(p.this(), "unable to cast supplied set type", "parseBuiltinSet")
			return nil
		}

		if setType.Element.Kind() != typArgs[0].Kind() {
			p.error(p.this(), "type mismatch in @set element", "parseBuiltinSet")
			return nil
		}
	}

	if p.this().Type != tokens.LParen {
		p.error(p.this(), "expected '(' after @set", "parseBuiltinSet")
		return nil
	}

	p.advance("parseBuiltinSet (") // consume (

	var args []ast.Expression

	if p.this().Type != tokens.RParen {
		var capType types.Type = types.None

		if len(typArgs) > 1 {
			capType = typArgs[1]
		}

		capArg := p.expression(ctx, capType)
		if capArg == nil {
			return nil
		}

		args = append(args, capArg)
	}

	if p.this().Type != tokens.RParen {
		p.error(p.this(), "expected ')' after argument in @set", "parseBuiltinSet")
		return nil
	}

	p.advance("parseBuiltinSet )") // consume ')'

	return &ast.Builtin{
		Token:         t,
		Name:          "set",
		TypeArguments: typArgs,
		Arguments:     args,
		ReturnType:    &types.Set{Element: typArgs[0]},
	}
}

func (p *Parser) parseBuiltinSlice(ctx context.Context, t tokens.Token, tokenType types.Type) *ast.Builtin {
	if tokenType.Kind() != types.Invalid && tokenType.Kind() != types.SliceKind {
		// If type is supplied, check if it's a slice.
		p.error(p.this(), "expected slice type", "parseBuiltinSlice")
		return nil
	}

	typArgs := p.parseTypeArguments(ctx)
	if typArgs == nil {
		return nil
	}

	if len(typArgs) < 1 || len(typArgs) > 2 {
		p.error(p.this(), "@slice requires one or two type arguments", "parseBuiltinSlice")
		return nil
	}

	if tokenType.Kind() != types.Invalid {
		sliceType, ok := tokenType.Underlying().(*types.Slice)
		if !ok {
			p.error(p.this(), "unable to cast supplied slice type", "parseBuiltinSlice")
			return nil
		}

		if sliceType.Element.Kind() != typArgs[0].Kind() {
			p.error(p.this(), "type mismatch in @slice element", "parseBuiltinSlice")
			return nil
		}
	}

	if p.this().Type != tokens.LParen {
		p.error(p.this(), "expected '(' after @slice", "parseBuiltinSlice")
		return nil
	}

	p.advance("parseBuiltinSlice (") // consume (

	var lenType types.Type = types.None

	if len(typArgs) == 2 {
		lenType = typArgs[1]
	}

	lenArg := p.expression(ctx, lenType)
	if lenArg == nil {
		return nil
	}

	args := []ast.Expression{lenArg}

	if p.this().Type == tokens.Comma {
		capArg := p.expression(ctx, lenType)
		if capArg == nil {
			return nil
		}

		args = append(args, capArg)
	}

	if p.this().Type != tokens.RParen {
		p.error(p.this(), "expected ')' after argument in @slice", "parseBuiltinSlice")
		return nil
	}

	p.advance("parseBuiltinSlice )") // consume ')'

	return &ast.Builtin{
		Token:         t,
		Name:          "slice",
		TypeArguments: typArgs,
		Arguments:     args,
		ReturnType:    &types.Slice{Element: typArgs[0]},
	}
}
