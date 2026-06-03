package parser

import (
	"context"
	"fmt"
	"slices"

	"github.com/samborkent/cog/internal/ast"
	"github.com/samborkent/cog/internal/tokens"
	"github.com/samborkent/cog/internal/types"
)

// TODO: base this heuristics.
const (
	arrayLiteralPreallocationSize = 4
	mapLiteralPreallocationSize   = 4
	setLiteralPreallocationSize   = 4
	sliceLiteralPreallocationSize = 4
)

func (p *Parser) primary(ctx context.Context, typeToken types.Type) ast.ExprIndex {
	// During globals pass, if the expected type is an unresolved forward alias
	// and we're about to parse a {}-expression, defer immediately before the
	// alias unwrapping below would lose the type info.
	if p.globalsPass && p.lex.This().Type == tokens.LBrace && typeToken != nil {
		if alias, ok := typeToken.(*types.Alias); ok && !alias.IsTypeParam() && types.IsNone(alias.Derived) {
			offset := p.lex.Offset()
			p.lex.SkipBody()
			return p.ast.NewDeferredExpr(p.lex.Peek(-1), offset, typeToken)
		}
	}

	if typeToken != nil {
		aliasType, ok := typeToken.(*types.Alias)
		if ok && !aliasType.IsTypeParam() {
			typeToken = aliasType.Underlying()
		}

		switch typeToken.Kind() {
		case types.OptionKind:
			// Handle option literal.
			optionType, ok := typeToken.(*types.Option)
			if !ok {
				p.error(p.lex.This(), "unable to assert option type", "primary")
				return ast.ZeroExprIndex
			}

			// TODO: handle none type
			typeToken = optionType.Value
		case types.EitherKind:
			// Handle either literal.
			eitherType, ok := typeToken.(*types.Either)
			if !ok {
				p.error(p.lex.This(), "unable to assert either type", "primary")
				return ast.ZeroExprIndex
			}

			token := p.lex.This()

			// Infer type.
			expr := p.primary(ctx, types.None)
			if expr == ast.ZeroExprIndex {
				return ast.ZeroExprIndex
			}

			exprType := p.ast.Expr(expr).Type()

			var isRight bool

			if types.Equal(exprType, eitherType.Left) {
				// matched left
			} else if types.Equal(exprType, eitherType.Right) {
				isRight = true
			} else {
				p.error(p.lex.This(), fmt.Sprintf("expression of type %q not in either type %q", exprType, eitherType), "primary")
				return ast.ZeroExprIndex
			}

			return p.ast.NewEitherLiteral(token, eitherType, expr, isRight)
		}
	}

	if p.match(p.lex.This(), tokens.LBracket, tokens.Map, tokens.Set) {
		// Literal with type annotation.
		literalType := p.parseType(ctx)

		// TODO: should probably use [types.Equal]? This could just compare name for [*types.Alias].
		if typeToken != types.None && literalType.String() != typeToken.String() {
			p.error(p.lex.This(), fmt.Sprintf("literal type %q does not match expected type %q", literalType, typeToken), "primary")
			return ast.ZeroExprIndex
		}

		typeToken = literalType
	}

	switch p.lex.This().Type {
	case tokens.Builtin:
		t := p.lex.This()

		if t.Literal == "go" {
			return p.parseGoCallExpression(ctx)
		}

		p.lex.Step() // consume @

		builtinParser, ok := p.builtins[t.Literal]
		if !ok {
			p.error(t, "unknown builtin function", "primary")
			return ast.ZeroExprIndex
		}

		node := builtinParser(ctx, t, typeToken)
		if node == ast.ZeroExprIndex {
			return ast.ZeroExprIndex
		}

		return node
	case tokens.FloatLiteral,
		tokens.IntLiteral,
		tokens.StringLiteral:
		return p.parseLiteral(typeToken)
	case tokens.False, tokens.True:
		p.lex.Step() // consume literal
		return p.ast.NewBoolLiteral(p.lex.Peek(-1))
	case tokens.LParen: // Grouped expression
		lparenToken := p.lex.This()
		p.lex.Step() // consume '('

		expr := p.expression(ctx, typeToken)

		if p.lex.This().Type != tokens.RParen {
			p.error(p.lex.This(), "expected ')' after grouped expression", "primary")
			return ast.ZeroExprIndex
		}

		p.lex.Step() // consume ')'

		var exprType types.Type = types.None
		if expr != ast.ZeroExprIndex {
			innerExpr := p.ast.Expr(expr)
			if innerExpr != nil {
				exprType = innerExpr.Type()
			}
		}

		return p.ast.NewGrouped(lparenToken, expr, exprType)
	case tokens.Identifier:
		symbol, ok := p.symbols.Resolve(p.lex.This().Literal)
		if !ok {
			// Check if this is an imported cog package name.
			imp, isImport := p.symbols.ResolveCogImport(p.lex.This().Literal)
			if isImport {
				return p.parsePkgSelector(ctx, imp)
			}

			if p.globalsPass {
				// Forward value reference: create a stub that resolves
				// when the real declaration is encountered later.
				ident := &ast.Identifier{
					Token:     p.lex.This(),
					Qualifier: ast.QualifierImmutable,
					Global:    true,
					ValueType: types.None,
				}

				p.symbols.DefineGlobal(ident)
				p.lex.Step() // consume identifier

				return p.ast.AddExpr(ident)
			}

			p.error(p.lex.This(), "undefined identifier", "primary")

			return ast.ZeroExprIndex
		}

		p.symbols.MarkUsed(p.lex.This().Literal)
		p.lex.Step() // consume identifier

		if symbol.Identifier.Qualifier == ast.QualifierType && p.lex.This().Type == tokens.LBrace {
			// Named struct literal
			literal := p.primary(ctx, symbol.Type())
			if literal == ast.ZeroExprIndex {
				return ast.ZeroExprIndex
			}

			literalExpr := p.ast.Expr(literal)

			literalExpr.(*ast.StructLiteral).StructType = &types.Alias{
				Name:     symbol.Identifier.Token.Literal,
				Derived:  literalExpr.Type(),
				Exported: symbol.Identifier.Exported,
				Global:   symbol.Identifier.Global,
			}

			return literal
		}

		switch p.lex.This().Type {
		case tokens.LParen:
			callToken := p.lex.This()

			// Function call
			procType, ok := symbol.Identifier.ValueType.(*types.Procedure)
			if !ok {
				p.error(p.lex.This(), "identifier is not callable", "primary")
				return ast.ZeroExprIndex
			}

			identExpr := p.ast.AddExpr(symbol.Identifier)

			args := p.parseCallArguments(ctx, procType)
			if args == nil {
				return ast.ZeroExprIndex
			}

			// Rules 7 & 8: func borrows args, proc consumes args.
			p.applyCallOwnership(callToken, procType, args)

			if len(procType.TypeParams) > 0 {
				// Generic call with type inference.
				typeArgs, returnType := p.inferTypeArgs(procType, args)
				if typeArgs == nil {
					return ast.ZeroExprIndex
				}

				return p.ast.NewCall(callToken, identExpr, args, returnType, typeArgs...)
			}

			return p.ast.NewCall(callToken, identExpr, args, procType.ReturnType)
		case tokens.LT:
			// Explicit type arguments on generic call: genFunc<utf8>("hello")
			procType, ok := symbol.Identifier.ValueType.(*types.Procedure)
			if !ok || len(procType.TypeParams) == 0 {
				// Not a generic callable — let comparison() handle '<'.
				return p.ast.AddExpr(symbol.Identifier)
			}

			typeArgs := p.parseTypeArguments(ctx)
			if typeArgs == nil {
				return ast.ZeroExprIndex
			}

			callToken := p.lex.This()

			if p.lex.This().Type != tokens.LParen {
				p.error(p.lex.This(), "expected '(' after type arguments in generic call", "primary")
				return ast.ZeroExprIndex
			}

			args := p.parseCallArguments(ctx, procType)
			returnType := p.validateExplicitTypeArgs(procType, typeArgs, args)

			// Validation failed (nil) but proc has a return type — error already reported.
			if returnType == nil && procType.ReturnType != nil {
				return ast.ZeroExprIndex
			}

			return p.ast.NewCall(callToken, p.ast.AddExpr(symbol.Identifier), args, returnType, typeArgs...)
		case tokens.Dot:
			symbolType := symbol.Type()
			kind := symbolType.Kind()

			if symbol.Identifier.Qualifier == ast.QualifierType &&
				kind != types.EnumKind && kind != types.ErrorKind {
				p.error(p.lex.This(), fmt.Sprintf("%q is a type, not a value: cannot invoke methods on types", symbol.Identifier.Token.Literal), "primary")
				return ast.ZeroExprIndex
			}

			// Selector expression
			selector := p.lex.This()

			expr := p.ast.AddExpr(symbol.Identifier)

			var selExpr *ast.Selector

			for p.lex.This().Type == tokens.Dot && p.lex.This().Type != tokens.EOF {
				p.lex.Step() // consume .

				if p.lex.This().Type != tokens.Identifier {
					p.error(p.lex.This(), "expected field identifier after . selector", "primary")
					return ast.ZeroExprIndex
				}

				if selExpr == nil {
					selExpr = ast.New[ast.Selector](p.ast)
					selExpr.Token = selector
					// Add the base identifier (e.g., 'p' in 'p.x') to Fields
					selExpr.Fields = append(selExpr.Fields, symbol.Identifier)
				}

				var typName string

				switch kind {
				case types.EnumKind, types.ErrorKind:
					typName = symbol.Identifier.Token.Literal
				default:
					typName = symbolType.String()
				}

				field, ok := p.symbols.ResolveField(typName, p.lex.This().Literal)
				if !ok {
					p.error(p.lex.This(), fmt.Sprintf("undefined field %q for selector %q", p.lex.This().Literal, typName), "primary")
					return ast.ZeroExprIndex
				}

				field.Identifier.Token = p.lex.This()

				p.lex.Step() // consume field identifier

				// For enum selectors, wrap the field type in an alias so the enum
				// type can be inferred downstream.  For struct fields, preserve the
				// original field type (e.g. float64) so arithmetic works correctly.
				if field.Scope == EnumScope {
					field.Identifier.ValueType = &types.Alias{
						Name:     symbol.Identifier.Token.Literal,
						Derived:  symbol.Type(),
						Exported: symbol.Identifier.Exported,
						Global:   symbol.Identifier.Global,
					}
				}

				// Add selected field to selector expression.
				selExpr.Fields = append(selExpr.Fields, field.Identifier)

				// Update symbolType for chained selector expressions.
				symbolType = field.Type()
			}

			if selExpr != nil {
				expr = p.ast.AddExpr(selExpr)
			}

			if p.match(p.lex.This(), tokens.LParen, tokens.LT) {
				exprType := p.ast.Expr(expr).Type()

				// Method call expression
				if exprType.Kind() != types.ProcedureKind {
					p.error(p.lex.Peek(-1), fmt.Sprintf("cannot call expression: expression of type %q is not a function", exprType))
					return ast.ZeroExprIndex
				}

				procType, ok := exprType.(*types.Procedure)
				if !ok {
					panic("unable to cast procedure kind expressions to type in call parsing")
				}

				var typeArgs []types.Type

				if p.lex.This().Type == tokens.LT {
					typeArgs = p.parseTypeArguments(ctx)
					if typeArgs == nil {
						return ast.ZeroExprIndex
					}
				}

				callToken := p.lex.This()

				args := p.parseCallArguments(ctx, procType)
				if args == nil {
					return ast.ZeroExprIndex
				}

				// Rules 7 & 8: func borrows args, proc consumes args.
				if !p.applyCallOwnership(callToken, procType, args) {
					return ast.ZeroExprIndex
				}

				return p.ast.NewCall(callToken, expr, args, procType.ReturnType, typeArgs...)
			}

			return expr
		default:
			// Variable reference
			if symbol.Identifier == nil {
				p.error(p.lex.This(), "nil identifier in variable reference", "primary")
				return ast.ZeroExprIndex
			}

			if symbol.Identifier.ValueType != nil &&
				typeToken.Kind() != types.Invalid &&
				symbol.Identifier.ValueType.Kind() != typeToken.Kind() {
				// Allow option-typed identifiers when the inner type matches the expected type.
				optType, isOption := symbol.Identifier.ValueType.(*types.Option)
				if !isOption || optType.Value.Kind() != typeToken.Kind() {
					p.error(p.lex.This(), fmt.Sprintf("type of identifier %q (%s) does not match expected type (%s)", symbol.Identifier.Token.Literal, symbol.Identifier.ValueType, typeToken), "primary")
					return ast.ZeroExprIndex
				}
			}

			// TODO: allocate identifiers in arena, and cache.
			return p.ast.AddExpr(symbol.Identifier)
		}
	case tokens.LBrace:
		switch t := typeToken.(type) {
		case *types.Alias:
			expr := p.primary(ctx, t.Derived)
			if expr == ast.ZeroExprIndex {
				return ast.ZeroExprIndex
			}

			// Place back type alias
			switch literal := p.ast.Expr(expr).(type) {
			case *ast.ArrayLiteral:
				// TODO: why does array need special handling?
				literal.ArrayType = t.Derived.Underlying().(*types.Array)
			case *ast.MapLiteral:
				literal.MapType = t
			case *ast.SetLiteral:
				literal.SetType = t
			case *ast.SliceLiteral:
				literal.SliceType = t
			case *ast.StructLiteral:
				literal.StructType = t
			case *ast.TupleLiteral:
				literal.TupleType = t
			case *ast.EitherLiteral:
				literal.EitherType = t
			}

			return expr
		case *types.Array:
			arrayToken := p.lex.This()
			// TODO: see if it's possible to evaluate array length
			values := make([]ast.ExprIndex, 0, arrayLiteralPreallocationSize)

			p.lex.Step() // consume {

			for !p.match(p.lex.This(), tokens.RBrace, tokens.EOF) {
				if ctx.Err() != nil {
					return ast.ZeroExprIndex
				}

				value := p.expression(ctx, t.Element)
				if value != ast.ZeroExprIndex {
					values = append(values, value)
				}

				if p.lex.This().Type == tokens.Comma {
					p.lex.Step() // consume ','
				}
			}

			if p.lex.This().Type != tokens.RBrace {
				p.error(arrayToken, "array literal is missing closing }", "primary")
				return ast.ZeroExprIndex
			}

			p.lex.Step() // consume }

			return p.ast.NewArrayLiteral(arrayToken, t, values)
		case *types.Map:
			mapToken := p.lex.This()
			pairs := make([]ast.KeyValue, 0, mapLiteralPreallocationSize)

			p.lex.Step() // consume {

			for p.lex.This().Type != tokens.EOF && ctx.Err() == nil {
				tok := p.lex.This()
				if tok.Type == tokens.RBrace {
					break
				}

				key := p.expression(ctx, t.Key)
				if key != ast.ZeroExprIndex {
					// TODO: optimize
					for i := range pairs {
						if p.ExprString(pairs[i].Key) == p.ExprString(key) {
							p.error(p.lex.Peek(-1), "duplicate key in map literal", "primary")
							return ast.ZeroExprIndex
						}
					}
				}

				if p.lex.This().Type != tokens.Colon {
					p.error(p.lex.This(), "expected colon after key in map literal", "primary")
					return ast.ZeroExprIndex
				}

				p.lex.Step() // consume :

				val := p.expression(ctx, t.Value)
				if val == ast.ZeroExprIndex {
					return ast.ZeroExprIndex
				}

				pairs = append(pairs, ast.KeyValue{
					Key:   key,
					Value: val,
				})

				if p.lex.This().Type == tokens.Comma {
					p.lex.Step() // consume ,
				}
			}

			if p.lex.This().Type != tokens.RBrace {
				p.error(mapToken, "map literal is missing closing }", "primary")
				return ast.ZeroExprIndex
			}

			p.lex.Step() // consume }

			return p.ast.NewMapLiteral(mapToken, t, pairs)
		case *types.Procedure:
			procToken := p.lex.This()

			// During globals pass, defer the body: capture offset, skip the block.
			if p.globalsPass {
				offset := p.lex.Offset()
				p.lex.SkipBody()
				bodyIdx := p.ast.NewDeferredBody(procToken, offset, p.currentReceiver)
				return p.ast.NewProcedureLiteral(procToken, t, bodyIdx)
			}

			// Re-enter type parameter scope so methods are visible in the body.
			if len(t.TypeParams) > 0 {
				p.symbols = NewEnclosedSymbolTable(p.symbols)

				for _, tp := range t.TypeParams {
					p.symbols.Define(&ast.Identifier{
						Token: tokens.Token{
							Type:    tokens.Identifier,
							Literal: tp.Name,
						},
						ValueType: tp,
						Qualifier: ast.QualifierType,
					})

					// Register interface methods from the constraint.
					iface, ok := tp.Underlying().(*types.Interface)
					if ok {
						for _, method := range iface.Methods {
							p.symbols.DefineMethod(tp.Name, &ast.Identifier{
								Token: tokens.Token{
									Type:    tokens.Identifier,
									Literal: method.Name,
								},
								ValueType: method.Procedure,
								Qualifier: ast.QualifierMethod,
							})
						}
					}
				}
			}

			if len(t.Parameters) > 0 {
				// Enter parameter scope
				p.symbols = NewEnclosedSymbolTable(p.symbols)

				for _, param := range t.Parameters {
					qualifier := ast.QualifierImmutable
					if param.Mutable {
						qualifier = ast.QualifierVariable
					}
					p.symbols.Define(&ast.Identifier{
						Token: tokens.Token{
							Type:    tokens.Identifier,
							Literal: param.Name,
						},
						ValueType: param.Type,
						Qualifier: qualifier,
					})
					p.symbols.MarkUsed(param.Name)
				}
			}

			// Track the return type for result-aware return parsing.
			prevReturnType := p.currentReturnType
			p.currentReturnType = t.ReturnType

			prevInPureFunc := p.inPureFunc
			p.inPureFunc = p.inPureFunc || t.Function

			body := p.parseBlockStatement(ctx)

			p.inPureFunc = prevInPureFunc
			p.currentReturnType = prevReturnType

			if len(t.Parameters) > 0 {
				// Leave parameter scope
				p.symbols = p.symbols.Outer
			}

			if len(t.TypeParams) > 0 {
				// Leave type parameter scope
				p.symbols = p.symbols.Outer
			}

			if body == nil {
				return ast.ZeroExprIndex
			}

			return p.ast.NewProcedureLiteral(procToken, t, p.ast.AddNode(body))
		case *types.Set:
			setToken := p.lex.This()
			values := make([]ast.ExprIndex, 0, setLiteralPreallocationSize)

			p.lex.Step() // consume {

			for p.lex.This().Type != tokens.EOF && ctx.Err() == nil {
				tok := p.lex.This()
				if tok.Type == tokens.RBrace {
					break
				}

				value := p.expression(ctx, t.Element)
				if value != ast.ZeroExprIndex {
					for i := range values {
						// TODO: optimize
						if p.ExprString(values[i]) == p.ExprString(value) {
							p.error(p.lex.Peek(-1), "duplicate key in set literal", "primary")
							return ast.ZeroExprIndex
						}
					}

					values = append(values, value)
				}

				if p.lex.This().Type == tokens.Comma {
					p.lex.Step() // consume ','
				}
			}

			if p.lex.This().Type != tokens.RBrace {
				p.error(setToken, "set literal is missing closing }", "primary")
				return ast.ZeroExprIndex
			}

			p.lex.Step() // consume }

			return p.ast.NewSetLiteral(setToken, t, values)
		case *types.Slice:
			sliceToken := p.lex.This()
			values := make([]ast.ExprIndex, 0, sliceLiteralPreallocationSize)

			p.lex.Step() // consume {

			for p.lex.This().Type != tokens.EOF && ctx.Err() == nil {
				tok := p.lex.This()
				if tok.Type == tokens.RBrace {
					break
				}

				value := p.expression(ctx, t.Element)
				if value != ast.ZeroExprIndex {
					values = append(values, value)
				}

				if p.lex.This().Type == tokens.Comma {
					p.lex.Step() // consume ','
				}
			}

			if p.lex.This().Type != tokens.RBrace {
				p.error(sliceToken, "slice literal is missing closing }", "primary")
				return ast.ZeroExprIndex
			}

			p.lex.Step() // consume }

			return p.ast.NewSliceLiteral(sliceToken, t, values)
		case *types.Struct:
			structToken := p.lex.This()
			values := make([]ast.FieldValue, 0, len(t.Fields))

			p.lex.Step() // consume {

			for p.lex.This().Type != tokens.EOF && ctx.Err() == nil {
				tok := p.lex.This()
				if tok.Type == tokens.RBrace {
					break
				}

				if tok.Type != tokens.Identifier {
					p.error(tok, "expected identifier at in struct literal", "primary")
					return ast.ZeroExprIndex
				}

				index := slices.IndexFunc(t.Fields, func(f *types.Field) bool {
					return f.Name == tok.Literal
				})

				if index == -1 {
					p.error(tok, "unknown field found in struct literal", "primary")
					return ast.ZeroExprIndex
				}

				fieldValue := ast.FieldValue{
					Name: tok.Literal,
				}

				p.lex.Step() // consume identifier

				if p.lex.This().Type != tokens.Assign {
					p.error(p.lex.This(), "expected = after identifier in struct literal", "primary")
					return ast.ZeroExprIndex
				}

				p.lex.Step() // consume =

				startToken := p.lex.This()

				value := p.expression(ctx, t.Fields[index].Type)
				if value == ast.ZeroExprIndex {
					p.error(startToken, "failed to parse field expression in struct literal", "primary")
					return ast.ZeroExprIndex
				}

				fieldValue.Value = value

				if p.lex.This().Type == tokens.Comma {
					p.lex.Step() // consume ','
				}

				values = append(values, fieldValue)
			}

			if p.lex.This().Type != tokens.RBrace {
				p.error(structToken, "struct literal is missing closing }", "primary")
				return ast.ZeroExprIndex
			}

			p.lex.Step() // consume }

			return p.ast.NewStructLiteral(structToken, t, values)
		case *types.Tuple:
			tupleToken := p.lex.This()
			values := make([]ast.ExprIndex, 0, len(t.Types))

			p.lex.Step() // consume {

			for i := range t.Types {
				startToken := p.lex.This()

				value := p.expression(ctx, t.Index(i))
				if value == ast.ZeroExprIndex {
					p.error(startToken, "failed to parse expression in tuple literal", "primary")
					return ast.ZeroExprIndex
				}

				values = append(values, value)

				if i < len(t.Types)-1 {
					if p.lex.This().Type != tokens.Comma {
						p.error(p.lex.This(), "expected , after expression in tuple literal", "primary")
						return ast.ZeroExprIndex
					}

					p.lex.Step() // consume ','
				}
			}

			if p.lex.This().Type != tokens.RBrace {
				p.error(tupleToken, "tuple literal is missing closing }", "primary")
				return ast.ZeroExprIndex
			}

			p.lex.Step() // consume }

			return p.ast.NewTupleLiteral(tupleToken, t, values)
		case *types.Basic:
			if t.Kind() != types.Complex32 {
				p.error(p.lex.This(), fmt.Sprintf("unexpected basic type %q for expression starting with {", t.String()), "primary")
				return ast.ZeroExprIndex
			}

			token := p.lex.This()
			p.lex.Step() // consume {

			realPart := p.expression(ctx, types.Basics[types.Float16])
			if realPart == ast.ZeroExprIndex {
				return ast.ZeroExprIndex
			}

			if p.lex.This().Type != tokens.Comma {
				p.error(p.lex.This(), "expected , after real part in complex32 literal", "primary")
				return ast.ZeroExprIndex
			}

			p.lex.Step() // consume ,

			imagPart := p.expression(ctx, types.Basics[types.Float16])
			if imagPart == ast.ZeroExprIndex {
				return ast.ZeroExprIndex
			}

			if p.lex.This().Type != tokens.RBrace {
				p.error(p.lex.This(), "expected } after imaginary part in complex32 literal", "primary")
				return ast.ZeroExprIndex
			}

			p.lex.Step() // consume }

			realLit, realOk := p.ast.Expr(realPart).(*ast.Float16Literal)
			imagLit, imagOk := p.ast.Expr(imagPart).(*ast.Float16Literal)

			if !realOk || !imagOk {
				p.error(token, "complex32 literal requires float16 literal values", "primary")
				return ast.ZeroExprIndex
			}

			return p.ast.NewComplex32Literal(token, ast.Complex32{realLit.Value, imagLit.Value})
		default:
			if typeToken == nil || typeToken == types.None {
				p.error(p.lex.Peek(-1), "cannot infer type for untyped literal", "primary")
				p.lex.Step() // consume {

				for p.lex.This().Type != tokens.EOF && ctx.Err() == nil {
					if p.lex.This().Type == tokens.RBrace {
						break
					}
					p.lex.Step()
				}

				if p.lex.This().Type == tokens.RBrace {
					p.lex.Step() // consume }
				}

				return ast.ZeroExprIndex
			}

			p.error(p.lex.This(), fmt.Sprintf("unexpected type %q for expression starting with {", typeToken.String()), "primary")

			return ast.ZeroExprIndex
		}
	default:
		p.error(p.lex.This(), "unexpected token encountered while parsing expression", "primary")
		return ast.ZeroExprIndex
	}
}

func (p *Parser) match(t tokens.Token, types ...tokens.Type) bool {
	return slices.Contains(types, t.Type)
}
