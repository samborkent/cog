package parser

import (
	"context"
	"fmt"
	"slices"

	f16 "github.com/x448/float16"

	"github.com/samborkent/cog/internal/ast"
	"github.com/samborkent/cog/internal/tokens"
	"github.com/samborkent/cog/internal/types"
)

func (p *Parser) expression(ctx context.Context, typeToken types.Type) ast.ExprValue {
	expr := p.boolean(ctx, typeToken)

	for p.match(tokens.LBracket) {
		if ctx.Err() != nil || expr == ast.ZeroExpr {
			return ast.ZeroExpr
		}

		operator := p.this()
		p.advance("expression index operator") // consume operator
		index := p.boolean(ctx, types.None)

		expr = ast.NewExpr(ast.KindIndex, types.Invalid, &ast.Index{
			Token:      operator,
			Identifier: expr,
			Index:      index,
		})

		if p.this().Type != tokens.RBracket {
			p.error(p.this(), "expected ] after index expression", "expression")
			return ast.ZeroExpr
		}

		p.advance("expression ]") // consume ]
	}

	return expr
}

func (p *Parser) boolean(ctx context.Context, typeToken types.Type) ast.ExprValue {
	expr := p.equality(ctx, typeToken)

	for p.match(tokens.And, tokens.Or) {
		if ctx.Err() != nil || expr == ast.ZeroExpr {
			return ast.ZeroExpr
		}

		if !types.IsBool(expr.Type()) {
			p.error(p.this(), "operator requires bool type", "boolean")
			return ast.ZeroExpr
		}

		operator := p.this()
		p.advance("boolean operator") // consume operator
		right := p.equality(ctx, types.Basics[types.Bool])

		expr = ast.NewExpr(ast.KindInfix, types.Bool, &ast.Infix{
			Operator: operator,
			Left:     expr,
			Right:    right,
		})
	}

	return expr
}

func (p *Parser) equality(ctx context.Context, typeToken types.Type) ast.ExprValue {
	expr := p.comparison(ctx, typeToken)

	for p.match(tokens.Equal, tokens.NotEqual) {
		if ctx.Err() != nil || expr == ast.ZeroExpr {
			return ast.ZeroExpr
		}

		operator := p.this()
		p.advance("equality operator") // consume operator
		right := p.comparison(ctx, types.None)

		infix := &ast.Infix{
			Operator: operator,
			Left:     expr,
			Right:    right,
		}

		if infix.Left.TypeKind != infix.Right.TypeKind {
			infix.EqualizeLiteralTypes()
		}

		expr = ast.NewExpr(ast.KindInfix, types.Bool, infix)
	}

	return expr
}

func (p *Parser) comparison(ctx context.Context, typeToken types.Type) ast.ExprValue {
	expr := p.term(ctx, typeToken)

	for p.match(tokens.GT, tokens.GTEqual, tokens.LT, tokens.LTEqual) {
		if ctx.Err() != nil || expr == ast.ZeroExpr {
			return ast.ZeroExpr
		}

		if !types.IsNumber(expr.Type()) {
			p.error(p.this(), "operator requires numeric type", "comparison")
			return ast.ZeroExpr
		}

		operator := p.this()
		p.advance("comparison operator") // consume operator
		right := p.term(ctx, types.None)

		infix := &ast.Infix{
			Operator: operator,
			Left:     expr,
			Right:    right,
		}

		if infix.Left.Type().Kind() != infix.Right.Type().Kind() {
			infix.EqualizeLiteralTypes()
		}

		expr = ast.NewExpr(ast.KindInfix, types.Bool, infix)
	}

	return expr
}

func (p *Parser) term(ctx context.Context, typeToken types.Type) ast.ExprValue {
	expr := p.factor(ctx, typeToken)

	for p.match(tokens.Minus, tokens.Plus) {
		if ctx.Err() != nil || expr == ast.ZeroExpr {
			return ast.ZeroExpr
		}

		t := expr.Type()

		// TODO: this is a hack due to lack of known Go typing at compile time, figure out a better solution.
		if t != types.None {
			if p.this().Type == tokens.Plus {
				if !types.IsSummable(t) {
					p.error(p.this(), fmt.Sprintf("operator requires numeric or string type, got %q", t), "term")
					return ast.ZeroExpr
				}
			} else {
				// Minus
				if !types.IsNumber(t) {
					p.error(p.this(), fmt.Sprintf("operator requires numeric type, got %q", t), "term")
					return ast.ZeroExpr
				}
			}
		}

		operator := p.this()
		p.advance("term operator") // consume operator
		right := p.factor(ctx, t)

		expr = ast.NewExpr(ast.KindInfix, t.Kind(), &ast.Infix{
			Operator: operator,
			Left:     expr,
			Right:    right,
		})
	}

	return expr
}

func (p *Parser) factor(ctx context.Context, typeToken types.Type) ast.ExprValue {
	expr := p.unary(ctx, typeToken)

	for p.match(tokens.Asterisk, tokens.Divide) {
		if ctx.Err() != nil || expr == ast.ZeroExpr {
			return ast.ZeroExpr
		}

		t := expr.Type()

		if !types.IsNumber(t) {
			p.error(p.this(), "operator requires numeric type", "factor")
			return ast.ZeroExpr
		}

		operator := p.this()
		p.advance("factor operator") // consume operator
		right := p.unary(ctx, t)

		expr = ast.NewExpr(ast.KindInfix, t.Kind(), &ast.Infix{
			Operator: operator,
			Left:     expr,
			Right:    right,
		})
	}

	return expr
}

func (p *Parser) unary(ctx context.Context, typeToken types.Type) ast.ExprValue {
	if p.match(tokens.Not, tokens.Minus, tokens.BitAnd) {
		// Previous operator is stored, to disallow double references.
		prevOperator := p.prev()
		if prevOperator.Type == tokens.LParen && p.i >= 2 && p.tokens[p.i-2].Type == tokens.BitAnd {
			prevOperator = p.tokens[p.i-2]
		}

		operator := p.this()
		p.advance("unary operator") // consume operator

		exprType := typeToken

		if operator.Type == tokens.BitAnd {
			// Special reference handling.
			if prevOperator.Type == tokens.BitAnd {
				p.error(p.this(), "double reference is not allowed", "unary")
				return ast.ZeroExpr
			}

			if typeToken != types.None && typeToken.Kind() == types.ReferenceKind {
				// If a type is specified, we need to pass the reference underlying type to the expression parsing.
				refType, ok := typeToken.(*types.Reference)
				if !ok {
					p.error(p.this(), "unable to assert reference type", "unary")
					return ast.ZeroExpr
				}

				exprType = refType.Value
			}
		}

		right := p.unary(ctx, exprType)
		if right == ast.ZeroExpr {
			return ast.ZeroExpr
		}

		rt := right.Type()

		if operator.Type == tokens.Not && !types.IsBool(rt) {
			p.error(p.this(), "operator requires bool type", "unary")
			return ast.ZeroExpr
		} else if operator.Type == tokens.Minus && !types.IsSigned(rt) {
			p.error(p.this(), "operator requires signed numeric type", "unary")
			return ast.ZeroExpr
		}

		return ast.NewExpr(ast.KindPrefix, rt.Kind(), &ast.Prefix{
			Operator: operator,
			Right:    right,
		})
	}

	if (typeToken == nil || typeToken == types.None) && p.this().Type == tokens.Identifier {
		// TODO: get rid of double lookup for identifiers
		symbol, ok := p.symbols.Resolve(p.this().Literal)
		if !ok {
			// If this is an imported package name, skip the type pre-lookup;
			// primary() will handle it via parsePkgSelector.
			if _, isImport := p.symbols.ResolveCogImport(p.this().Literal); !isImport {
				p.error(p.this(), "undefined identifier", "primary")
				return ast.ZeroExpr
			}
		} else {
			typeToken = symbol.Type()
		}
	}

	node := p.primary(ctx, typeToken)
	if node == ast.ZeroExpr {
		return ast.ZeroExpr
	}

	if p.this().Type == tokens.Question {
		token := p.this()
		p.advance("unary ?") // consume ?

		// ? works on both option and result types.
		if typeToken.Kind() != types.OptionKind && typeToken.Kind() != types.ResultKind {
			p.error(token, "? operator requires option or result type", "unary")
			return ast.ZeroExpr
		}

		return ast.NewExpr(ast.KindSuffix, node.Type().Kind(), &ast.Suffix{
			Operator: token,
			Left:     node,
		})
	}

	if p.this().Type == tokens.Not {
		token := p.this()
		p.advance("unary !") // consume !

		if typeToken.Kind() != types.ResultKind {
			p.error(token, "! operator requires result type", "unary")
			return ast.ZeroExpr
		}

		// Must-check: cannot extract error without checking ? first.
		if node.NodeKind == ast.KindIdentifier {
			ident := node.AsIdentifier()
			if !p.symbols.IsErrorChecked(ident.Name) {
				p.error(ident.Token, "must check "+ident.Name+" before accessing error", "unary")
				return ast.ZeroExpr
			}
		}

		return ast.NewExpr(ast.KindSuffix, node.Type().Kind(), &ast.Suffix{
			Operator: token,
			Left:     node,
		})
	}

	// Must-check analysis: bare access to option/result requires prior ? check.
	if node.NodeKind == ast.KindIdentifier {
		ident := node.AsIdentifier()

		if (node.TypeKind == types.OptionKind || node.TypeKind == types.ResultKind) && !p.symbols.IsValueChecked(ident.Name) {
			p.error(ident.Token, "must check "+ident.Name+" before accessing value", "unary")
			return ast.ZeroExpr
		}
	}

	return node
}

func (p *Parser) primary(ctx context.Context, typeToken types.Type) ast.ExprValue {
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
				p.error(p.this(), "unable to assert option type", "primary")
				return ast.ZeroExpr
			}

			// TODO: handle none type

			typeToken = optionType.Value
		case types.EitherKind:
			// Handle either literal.
			eitherType, ok := typeToken.(*types.Either)
			if !ok {
				p.error(p.this(), "unable to assert either type", "primary")
				return ast.ZeroExpr
			}

			token := p.this()

			// Infer type.
			expr := p.primary(ctx, types.None)
			if expr == ast.ZeroExpr {
				return ast.ZeroExpr
			}

			var isRight bool

			if types.Equal(expr.Type(), eitherType.Left) {
				// matched left
			} else if types.Equal(expr.Type(), eitherType.Right) {
				isRight = true
			} else {
				p.error(p.this(), fmt.Sprintf("expression of type %q not in either type %q", expr.Type().String(), eitherType.String()), "primary")
				return ast.ZeroExpr
			}

			return ast.NewExpr(ast.KindEitherLiteral, types.EitherKind, &ast.EitherLiteral{
				Token:      token,
				EitherType: eitherType,
				Value:      expr,
				IsRight:    isRight,
			})
		}
	}

	if p.match(tokens.LBracket, tokens.Map, tokens.Set) {
		// Literal with type annotation.
		literalType := p.parseType(ctx)

		if typeToken != types.None && literalType.String() != typeToken.String() {
			p.error(p.this(), fmt.Sprintf("literal type %q does not match expected type %q", literalType.String(), typeToken.String()), "primary")
			return ast.ZeroExpr
		}

		typeToken = literalType
	}

	switch p.this().Type {
	case tokens.Builtin:
		t := p.this()

		if t.Literal == "go" {
			return p.parseGoCallExpression(ctx)
		}

		p.advance("primary builtin") // consume @

		builtinParser, ok := p.builtins[t.Literal]
		if !ok {
			p.error(t, "unknown builtin function", "primary")
			return ast.ZeroExpr
		}

		node := builtinParser(ctx, t, typeToken)
		if node == nil {
			return ast.ZeroExpr
		}

		// TODO: optimize, return type kind should be known based on builtin type.
		return ast.NewExpr(ast.KindBuiltin, node.Type().Kind(), node)
	case tokens.FloatLiteral,
		tokens.IntLiteral,
		tokens.StringLiteral:
		return p.parseLiteral(typeToken)
	case tokens.False, tokens.True:
		node := &ast.BoolLiteral{
			Token: p.this(),
			Value: p.this().Type == tokens.True,
		}

		p.advance("primary literal") // consume literal

		return ast.NewExpr(ast.KindBoolLiteral, types.Bool, node)
	case tokens.LParen: // Grouped expression
		p.advance("primary (") // consume '('

		expr := p.expression(ctx, typeToken)

		if p.this().Type != tokens.RParen {
			p.error(p.this(), "expected ')' after grouped expression", "primary")
			return ast.ZeroExpr
		}

		p.advance("primary )") // consume ')'

		return expr
	case tokens.Identifier:
		symbol, ok := p.symbols.Resolve(p.this().Literal)
		if !ok {
			// Check if this is an imported cog package name.
			imp, isImport := p.symbols.ResolveCogImport(p.this().Literal)
			if isImport {
				return p.parsePkgSelector(ctx, imp)
			}

			p.error(p.this(), "undefined identifier", "primary")

			return ast.ZeroExpr
		}

		p.advance("primary identifier") // consume identifier

		if symbol.Identifier.Qualifier == ast.QualifierType && p.this().Type == tokens.LBrace {
			// Named struct literal
			literal := p.primary(ctx, symbol.Type())
			if literal == ast.ZeroExpr {
				return ast.ZeroExpr
			}

			literal.AsStructLiteral().StructType = &types.Alias{
				Name:     symbol.Identifier.Name,
				Derived:  literal.Type(),
				Exported: symbol.Identifier.Exported,
				Global:   symbol.Identifier.Global,
			}

			return literal
		}

		switch p.this().Type {
		case tokens.LParen:
			// Function call
			procType, ok := symbol.Identifier.ValueType.(*types.Procedure)
			if !ok {
				p.error(p.this(), "identifier is not callable", "primary")
				return ast.ZeroExpr
			}

			if len(procType.TypeParams) > 0 {
				// Generic call with type inference.
				args := p.parseCallArguments(ctx, procType)

				typeArgs, returnType := p.inferTypeArgs(procType, args)
				if typeArgs == nil {
					return ast.ZeroExpr
				}

				return ast.NewExpr(ast.KindCall, returnType.Kind(), &ast.Call{
					Expr:       ast.NewExpr(ast.KindIdentifier, symbol.Identifier.ValueType.Kind(), symbol.Identifier),
					Arguments:  args,
					ReturnType: returnType,
					TypeArgs:   typeArgs,
				})
			}

			return ast.NewExpr(ast.KindCall, procType.ReturnType.Kind(), &ast.Call{
				Expr:       ast.NewExpr(ast.KindIdentifier, symbol.Identifier.ValueType.Kind(), symbol.Identifier),
				Arguments:  p.parseCallArguments(ctx, procType),
				ReturnType: procType.ReturnType,
			})
		case tokens.LT:
			// Explicit type arguments on generic call: genFunc<utf8>("hello")
			procType, ok := symbol.Identifier.ValueType.(*types.Procedure)
			if !ok || len(procType.TypeParams) == 0 {
				// Not a generic callable — let comparison() handle '<'.
				return ast.NewExpr(ast.KindIdentifier, symbol.Identifier.ValueType.Kind(), symbol.Identifier)
			}

			typeArgs := p.parseTypeArguments(ctx)
			if typeArgs == nil {
				return ast.ZeroExpr
			}

			if p.this().Type != tokens.LParen {
				p.error(p.this(), "expected '(' after type arguments in generic call", "primary")
				return ast.ZeroExpr
			}

			args := p.parseCallArguments(ctx, procType)
			returnType := p.validateExplicitTypeArgs(procType, typeArgs, args)

			// Validation failed (nil) but proc has a return type — error already reported.
			if returnType == nil && procType.ReturnType != nil {
				return ast.ZeroExpr
			}

			return ast.NewExpr(ast.KindCall, returnType.Kind(), &ast.Call{
				Expr:       ast.NewExpr(ast.KindIdentifier, symbol.Identifier.ValueType.Kind(), symbol.Identifier),
				Arguments:  args,
				ReturnType: returnType,
				TypeArgs:   typeArgs,
			})
		case tokens.Dot:
			symbolType := symbol.Type()
			kind := symbolType.Kind()

			if symbol.Identifier.Qualifier == ast.QualifierType &&
				kind != types.EnumKind && kind != types.ErrorKind {
				p.error(p.this(), fmt.Sprintf("%q is a type, not a value: cannot invoke methods on types", symbol.Identifier.Name), "primary")
				return ast.ZeroExpr
			}

			// Selector expression
			selector := p.this()

			var (
				expr     ast.Expr = symbol.Identifier
				selected *ast.Identifier
			)

			for p.this().Type == tokens.Dot && p.this().Type != tokens.EOF {
				p.advance("primary identifier .") // consume .

				if p.this().Type != tokens.Identifier {
					p.error(p.this(), "expected field identifier after . selector", "primary")
					return ast.ZeroExpr
				}

				var typName string

				switch kind {
				case types.EnumKind, types.ErrorKind:
					typName = symbol.Identifier.Name
				default:
					typName = symbolType.String()
				}

				field, ok := p.symbols.ResolveField(typName, p.this().Literal)
				if !ok {
					p.error(p.this(), fmt.Sprintf("undefined field %q for selector %q", p.this().Literal, typName), "primary")
					return ast.ZeroExpr
				}

				field.Identifier.Token = p.this()

				p.advance("primary identifier field") // consume field identifier

				// For enum selectors, wrap the field type in an alias so the enum
				// type can be inferred downstream.  For struct fields, preserve the
				// original field type (e.g. float64) so arithmetic works correctly.
				if field.Scope == EnumScope {
					field.Identifier.ValueType = &types.Alias{
						Name:     symbol.Identifier.Name,
						Derived:  symbol.Type(),
						Exported: symbol.Identifier.Exported,
						Global:   symbol.Identifier.Global,
					}
				}

				if selected != nil {
					// If there is already a selected field, add it to selector expression.
					expr = &ast.Selector{
						Token: selector,
						Expr:  ast.NewExpr(ast.KindSelector, expr.Type().Kind(), expr),
						Field: selected,
					}
				}

				// Change selected to the right most selected field.
				selected = field.Identifier

				// Update symbolType for chained selector expressions.
				symbolType = field.Type()
			}

			expr = &ast.Selector{
				Token: selector,
				Expr:  ast.NewExpr(ast.KindSelector, expr.Type().Kind(), expr),
				Field: selected,
			}

			if p.match(tokens.LParen, tokens.LT) {
				// Method call expression
				if expr.Type().Kind() != types.ProcedureKind {
					p.error(p.prev(), fmt.Sprintf("cannot call expression: expression of type %q is not a function", expr.Type()))
					return ast.ZeroExpr
				}

				procType, ok := expr.Type().(*types.Procedure)
				if !ok {
					panic("unable to cast procedure kind expressions to type in call parsing")
				}

				var typeArgs []types.Type

				if p.this().Type == tokens.LT {
					typeArgs = p.parseTypeArguments(ctx)
					if typeArgs == nil {
						return ast.ZeroExpr
					}
				}

				args := p.parseCallArguments(ctx, procType)
				if args == nil {
					return ast.ZeroExpr
				}

				return ast.NewExpr(ast.KindCall, procType.ReturnType.Kind(), &ast.Call{
					Expr:       ast.NewExpr(ast.KindSelector, expr.Type().Kind(), expr),
					Arguments:  args,
					ReturnType: procType.ReturnType,
					TypeArgs:   typeArgs,
				})
			}

			return ast.NewExpr(ast.KindSelector, selected.ValueType.Kind(), &ast.Selector{
				Token: selector,
				Expr:  ast.NewExpr(ast.KindSelector, expr.Type().Kind(), expr),
				Field: selected,
			})
		default:
			// Variable reference
			if symbol.Identifier == nil {
				p.error(p.this(), "nil identifier in variable reference", "primary")
				return ast.ZeroExpr
			}

			if symbol.Identifier.ValueType != nil &&
				typeToken.Kind() != types.Invalid &&
				symbol.Identifier.ValueType.Kind() != typeToken.Kind() {
				// Allow option-typed identifiers when the inner type matches the expected type.
				optType, isOption := symbol.Identifier.ValueType.(*types.Option)
				if !isOption || optType.Value.Kind() != typeToken.Kind() {
					p.error(p.this(), fmt.Sprintf("type of identifier %q (%s) does not match expected type (%s)", symbol.Identifier.Name, symbol.Identifier.ValueType, typeToken), "primary")
					return ast.ZeroExpr
				}
			}

			return ast.NewExpr(ast.KindIdentifier, symbol.Identifier.ValueType.Kind(), symbol.Identifier)
		}
	case tokens.LBrace:
		switch t := typeToken.(type) {
		case *types.Alias:
			expr := p.primary(ctx, t.Derived)
			if expr == ast.ZeroExpr {
				return ast.ZeroExpr
			}

			// Place back type alias
			switch expr.NodeKind {
			case ast.KindArrayLiteral:
				// TODO: figure out why only array need this special handling.
				expr.AsArrayLiteral().ArrayType = t.Derived.Underlying().(*types.Array)
			case ast.KindEitherLiteral:
				expr.AsEitherLiteral().EitherType = t
			case ast.KindMapLiteral:
				expr.AsMapLiteral().MapType = t
			case ast.KindSetLiteral:
				expr.AsSetLiteral().SetType = t
			case ast.KindStructLiteral:
				expr.AsStructLiteral().StructType = t
			case ast.KindTupleLiteral:
				expr.AsTupleLiteral().TupleType = t
			}

			return expr
		case *types.Array:
			// TODO: see if it's possible to evaluate array length
			arrayLiteral := &ast.ArrayLiteral{
				Token:     p.this(),
				ArrayType: t,
				Values:    []ast.ExprValue{},
			}

			p.advance("primary array {") // consume {

			for !p.match(tokens.RBrace, tokens.EOF) {
				if ctx.Err() != nil {
					return ast.ZeroExpr
				}

				value := p.expression(ctx, t.Element)
				if value != ast.ZeroExpr {
					arrayLiteral.Values = append(arrayLiteral.Values, value)
				}

				if p.this().Type == tokens.Comma {
					p.advance("primary array ,") // consume ','
				}
			}

			if p.this().Type != tokens.RBrace {
				p.error(arrayLiteral.Token, "array literal is missing closing }", "primary")
				return ast.ZeroExpr
			}

			p.advance("primary array }") // consume }

			return ast.NewExpr(ast.KindArrayLiteral, types.ArrayKind, arrayLiteral)
		case *types.Map:
			mapLiteral := &ast.MapLiteral{
				Token:   p.this(),
				MapType: t,
				Pairs:   []*ast.KeyValue{},
			}

			p.advance("primary map {") // consume {

			for !p.match(tokens.RBrace, tokens.EOF) {
				if ctx.Err() != nil {
					return ast.ZeroExpr
				}

				key := p.expression(ctx, t.Key)
				if key != ast.ZeroExpr {
					// TODO: optimize
					for i := range mapLiteral.Pairs {
						if mapLiteral.Pairs[i].Key.String() == key.String() {
							p.error(p.prev(), "duplicate key in map literal", "primary")
							return ast.ZeroExpr
						}
					}
				}

				if p.this().Type != tokens.Colon {
					p.error(p.this(), "expected colon after key in map literal", "primary")
					return ast.ZeroExpr
				}

				p.advance("primary map :") // consume :

				val := p.expression(ctx, t.Value)
				if val == ast.ZeroExpr {
					return ast.ZeroExpr
				}

				mapLiteral.Pairs = append(mapLiteral.Pairs, &ast.KeyValue{
					Key:   key,
					Value: val,
				})

				if p.this().Type == tokens.Comma {
					p.advance("primary map ,") // consume ,
				}
			}

			if p.this().Type != tokens.RBrace {
				p.error(mapLiteral.Token, "map literal is missing closing }", "primary")
				return ast.ZeroExpr
			}

			p.advance("primary map }") // consume }

			return ast.NewExpr(ast.KindMapLiteral, types.MapKind, mapLiteral)
		case *types.Procedure:
			procLiteral := &ast.ProcedureLiteral{
				ProcedureType: t,
			}

			// Re-enter type parameter scope so methods are visible in the body.
			if len(t.TypeParams) > 0 {
				p.symbols = NewEnclosedSymbolTable(p.symbols)

				for _, tp := range t.TypeParams {
					p.symbols.Define(&ast.Identifier{
						Name:      tp.Name,
						ValueType: tp,
						Qualifier: ast.QualifierType,
					})

					// Register interface methods from the constraint.
					iface, ok := tp.Underlying().(*types.Interface)
					if ok {
						for _, method := range iface.Methods {
							p.symbols.DefineMethod(tp.Name, &ast.Identifier{
								Name:      method.Name,
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
					p.symbols.Define(&ast.Identifier{
						Name:      param.Name,
						ValueType: param.Type,
						Qualifier: ast.QualifierImmutable,
					})
				}
			}

			// Track the return type for result-aware return parsing.
			prevReturnType := p.currentReturnType
			p.currentReturnType = t.ReturnType

			body := p.parseBlockStatement(ctx)

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
				return ast.ZeroExpr
			}

			procLiteral.Body = body

			return ast.NewExpr(ast.KindProcedureLiteral, types.ProcedureKind, procLiteral)
		case *types.Set:
			setLiteral := &ast.SetLiteral{
				Token:   p.this(),
				SetType: t,
				Values:  []ast.ExprValue{},
			}

			p.advance("primary set {") // consume {

			for !p.match(tokens.RBrace, tokens.EOF) {
				if ctx.Err() != nil {
					return ast.ZeroExpr
				}

				value := p.expression(ctx, t.Element)
				if value != ast.ZeroExpr {
					for i := range setLiteral.Values {
						if setLiteral.Values[i].String() == value.String() {
							p.error(p.prev(), "duplicate key in set literal", "primary")
							return ast.ZeroExpr
						}
					}

					setLiteral.Values = append(setLiteral.Values, value)
				}

				if p.this().Type == tokens.Comma {
					p.advance("primary set ,") // consume ','
				}
			}

			if p.this().Type != tokens.RBrace {
				p.error(setLiteral.Token, "set literal is missing closing }", "primary")
				return ast.ZeroExpr
			}

			p.advance("primary set }") // consume }

			return ast.NewExpr(ast.KindSetLiteral, types.SetKind, setLiteral)
		case *types.Slice:
			sliceLiteral := &ast.SliceLiteral{
				Token:       p.this(),
				ElementType: t.Element,
				Values:      []ast.ExprValue{},
			}

			p.advance("primary slice {") // consume {

			for !p.match(tokens.RBrace, tokens.EOF) {
				if ctx.Err() != nil {
					return ast.ZeroExpr
				}

				value := p.expression(ctx, t.Element)
				if value != ast.ZeroExpr {
					sliceLiteral.Values = append(sliceLiteral.Values, value)
				}

				if p.this().Type == tokens.Comma {
					p.advance("primary array ,") // consume ','
				}
			}

			if p.this().Type != tokens.RBrace {
				p.error(sliceLiteral.Token, "slice literal is missing closing }", "primary")
				return ast.ZeroExpr
			}

			p.advance("primary slice }") // consume }

			return ast.NewExpr(ast.KindSliceLiteral, types.SliceKind, sliceLiteral)
		case *types.Struct:
			structLiteral := &ast.StructLiteral{
				Token:      p.this(),
				StructType: t,
				Values:     make([]ast.FieldValue, 0, len(t.Fields)),
			}

			p.advance("primary struct {") // consume {

			for !p.match(tokens.RBrace, tokens.EOF) {
				if ctx.Err() != nil {
					return ast.ZeroExpr
				}

				if p.this().Type != tokens.Identifier {
					p.error(p.this(), "expected identifier at in struct literal", "primary")
					return ast.ZeroExpr
				}

				index := slices.IndexFunc(t.Fields, func(f *types.Field) bool {
					return f.Name == p.this().Literal
				})

				if index == -1 {
					p.error(p.this(), "unknown field found in struct literal", "primary")
					return ast.ZeroExpr
				}

				fieldValue := ast.FieldValue{
					Name: p.this().Literal,
				}

				p.advance("primary struct identifier") // consume identifier

				if p.this().Type != tokens.Assign {
					p.error(p.this(), "expected = after identifier in struct literal", "primary")
					return ast.ZeroExpr
				}

				p.advance("primary struct =") // consume =

				startToken := p.this()

				value := p.expression(ctx, t.Fields[index].Type)
				if value == ast.ZeroExpr {
					p.error(startToken, "failed to parse field expression in struct literal", "primary")
					return ast.ZeroExpr
				}

				fieldValue.Value = value

				if p.this().Type == tokens.Comma {
					p.advance("primary struct ,") // consume ','
				}

				structLiteral.Values = append(structLiteral.Values, fieldValue)
			}

			if p.this().Type != tokens.RBrace {
				p.error(structLiteral.Token, "struct literal is missing closing }", "primary")
				return ast.ZeroExpr
			}

			p.advance("primary struct }") // consume }

			return ast.NewExpr(ast.KindStructLiteral, types.StructKind, structLiteral)
		case *types.Tuple:
			tupleLiteral := &ast.TupleLiteral{
				Token:     p.this(),
				TupleType: t,
				Values:    make([]ast.ExprValue, 0, len(t.Types)),
			}

			p.advance("primary tuple {") // consume {

			for i := range t.Types {
				startToken := p.this()

				value := p.expression(ctx, t.Index(i))
				if value == ast.ZeroExpr {
					p.error(startToken, "failed to parse expression in tuple literal", "primary")
					return ast.ZeroExpr
				}

				tupleLiteral.Values = append(tupleLiteral.Values, value)

				if i < len(t.Types)-1 {
					if p.this().Type != tokens.Comma {
						p.error(p.this(), "expected , after expression in tuple literal", "primary")
						return ast.ZeroExpr
					}

					p.advance("primary tuple ,") // consume ','
				}
			}

			if p.this().Type != tokens.RBrace {
				p.error(tupleLiteral.Token, "tuple literal is missing closing }", "primary")
				return ast.ZeroExpr
			}

			p.advance("primary tuple }") // consume }

			return ast.NewExpr(ast.KindTupleLiteral, types.TupleKind, tupleLiteral)
		case *types.Basic:
			if t.Kind() != types.Complex32 {
				p.error(p.this(), fmt.Sprintf("unexpected basic type %q for expression starting with {", t.String()), "primary")
				return ast.ZeroExpr
			}

			token := p.this()
			p.advance("primary complex32 {") // consume {

			realPart := p.expression(ctx, types.Basics[types.Float16])
			if realPart == ast.ZeroExpr {
				return ast.ZeroExpr
			}

			if p.this().Type != tokens.Comma {
				p.error(p.this(), "expected , after real part in complex32 literal", "primary")
				return ast.ZeroExpr
			}

			p.advance("primary complex32 ,") // consume ,

			imagPart := p.expression(ctx, types.Basics[types.Float16])
			if imagPart == ast.ZeroExpr {
				return ast.ZeroExpr
			}

			if p.this().Type != tokens.RBrace {
				p.error(p.this(), "expected } after imaginary part in complex32 literal", "primary")
				return ast.ZeroExpr
			}

			p.advance("primary complex32 }") // consume }

			if realPart.TypeKind != types.Float16 || imagPart.TypeKind != types.Float16 {
				p.error(token, "complex32 literal requires float16 literal values", "primary")
				return ast.ZeroExpr
			}

			return ast.NewExpr(ast.KindComplex32Literal, types.Complex32, &ast.Complex32Literal{
				Token: token,
				Value: [2]f16.Float16{realPart.AsFloat16Literal().Value, imagPart.AsFloat16Literal().Value},
			})
		default:
			if typeToken == nil || typeToken == types.None {
				p.error(p.prev(), "cannot infer type for untyped literal", "primary")
				p.advance("primary {") // consume {

				for !p.match(tokens.RBrace, tokens.EOF) {
					p.advance("primary skip token")
				}

				if p.this().Type == tokens.RBrace {
					p.advance("primary }") // consume }
				}

				return ast.ZeroExpr
			}

			p.error(p.this(), fmt.Sprintf("unexpected type %q for expression starting with {", typeToken.String()), "primary")

			return ast.ZeroExpr
		}
	default:
		p.error(p.this(), "unexpected token encountered while parsing expression", "primary")
		return ast.ZeroExpr
	}
}

func (p *Parser) match(types ...tokens.Type) bool {
	return slices.Contains(types, p.this().Type)
}
