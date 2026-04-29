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
				return ast.ZeroExprIndex
			}

			// TODO: handle none type
			typeToken = optionType.Value
		case types.EitherKind:
			// Handle either literal.
			eitherType, ok := typeToken.(*types.Either)
			if !ok {
				p.error(p.this(), "unable to assert either type", "primary")
				return ast.ZeroExprIndex
			}

			token := p.this()

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
				p.error(p.this(), fmt.Sprintf("expression of type %q not in either type %q", exprType, eitherType), "primary")
				return ast.ZeroExprIndex
			}

			return p.ast.NewEitherLiteral(token, eitherType, expr, isRight)
		}
	}

	if p.match(tokens.LBracket, tokens.Map, tokens.Set) {
		// Literal with type annotation.
		literalType := p.parseType(ctx)

		// TODO: should probably use [types.Equal]? This could just compare name for [*types.Alias].
		if typeToken != types.None && literalType.String() != typeToken.String() {
			p.error(p.this(), fmt.Sprintf("literal type %q does not match expected type %q", literalType, typeToken), "primary")
			return ast.ZeroExprIndex
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
		p.advance("primary literal") // consume literal
		return p.ast.NewBoolLiteral(p.prev())
	case tokens.LParen: // Grouped expression
		p.advance("primary (") // consume '('

		expr := p.expression(ctx, typeToken)

		if p.this().Type != tokens.RParen {
			p.error(p.this(), "expected ')' after grouped expression", "primary")
			return ast.ZeroExprIndex
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

			return ast.ZeroExprIndex
		}

		p.advance("primary identifier") // consume identifier

		if symbol.Identifier.Qualifier == ast.QualifierType && p.this().Type == tokens.LBrace {
			// Named struct literal
			literal := p.primary(ctx, symbol.Type())
			if literal == ast.ZeroExprIndex {
				return ast.ZeroExprIndex
			}

			literalExpr := p.ast.Expr(literal)

			literalExpr.(*ast.StructLiteral).StructType = &types.Alias{
				Name:     symbol.Identifier.Name,
				Derived:  literalExpr.Type(),
				Exported: symbol.Identifier.Exported,
				Global:   symbol.Identifier.Global,
			}

			return literal
		}

		switch p.this().Type {
		case tokens.LParen:
			callToken := p.this()

			// Function call
			procType, ok := symbol.Identifier.ValueType.(*types.Procedure)
			if !ok {
				p.error(p.this(), "identifier is not callable", "primary")
				return ast.ZeroExprIndex
			}

			identExpr := p.ast.AddExpr(symbol.Identifier)

			args := p.parseCallArguments(ctx, procType)
			if args == nil {
				return ast.ZeroExprIndex
			}

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

			callToken := p.this()

			if p.this().Type != tokens.LParen {
				p.error(p.this(), "expected '(' after type arguments in generic call", "primary")
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
				p.error(p.this(), fmt.Sprintf("%q is a type, not a value: cannot invoke methods on types", symbol.Identifier.Name), "primary")
				return ast.ZeroExprIndex
			}

			// Selector expression
			selector := p.this()

			expr := p.ast.AddExpr(symbol.Identifier)

			var selExpr *ast.Selector

			for p.this().Type == tokens.Dot && p.this().Type != tokens.EOF {
				p.advance("primary identifier .") // consume .

				if p.this().Type != tokens.Identifier {
					p.error(p.this(), "expected field identifier after . selector", "primary")
					return ast.ZeroExprIndex
				}

				if selExpr == nil {
					selExpr = ast.New[ast.Selector](p.ast)
					selExpr.Token = selector
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
					return ast.ZeroExprIndex
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

				// Add selected field to selector expression.
				selExpr.Fields = append(selExpr.Fields, field.Identifier)

				// Update symbolType for chained selector expressions.
				symbolType = field.Type()
			}

			if selExpr != nil {
				expr = p.ast.AddExpr(selExpr)
			}

			if p.match(tokens.LParen, tokens.LT) {
				exprType := p.ast.Expr(expr).Type()

				// Method call expression
				if exprType.Kind() != types.ProcedureKind {
					p.error(p.prev(), fmt.Sprintf("cannot call expression: expression of type %q is not a function", exprType))
					return ast.ZeroExprIndex
				}

				procType, ok := exprType.(*types.Procedure)
				if !ok {
					panic("unable to cast procedure kind expressions to type in call parsing")
				}

				var typeArgs []types.Type

				if p.this().Type == tokens.LT {
					typeArgs = p.parseTypeArguments(ctx)
					if typeArgs == nil {
						return ast.ZeroExprIndex
					}
				}

				callToken := p.this()

				args := p.parseCallArguments(ctx, procType)
				if args == nil {
					return ast.ZeroExprIndex
				}

				return p.ast.NewCall(callToken, expr, args, procType.ReturnType, typeArgs...)
			}

			return expr
		default:
			// Variable reference
			if symbol.Identifier == nil {
				p.error(p.this(), "nil identifier in variable reference", "primary")
				return ast.ZeroExprIndex
			}

			if symbol.Identifier.ValueType != nil &&
				typeToken.Kind() != types.Invalid &&
				symbol.Identifier.ValueType.Kind() != typeToken.Kind() {
				// Allow option-typed identifiers when the inner type matches the expected type.
				optType, isOption := symbol.Identifier.ValueType.(*types.Option)
				if !isOption || optType.Value.Kind() != typeToken.Kind() {
					p.error(p.this(), fmt.Sprintf("type of identifier %q (%s) does not match expected type (%s)", symbol.Identifier.Name, symbol.Identifier.ValueType, typeToken), "primary")
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
			arrayToken := p.this()
			// TODO: see if it's possible to evaluate array length
			values := make([]ast.ExprIndex, 0, arrayLiteralPreallocationSize)

			p.advance("primary array {") // consume {

			for !p.match(tokens.RBrace, tokens.EOF) {
				if ctx.Err() != nil {
					return ast.ZeroExprIndex
				}

				value := p.expression(ctx, t.Element)
				if value != ast.ZeroExprIndex {
					values = append(values, value)
				}

				if p.this().Type == tokens.Comma {
					p.advance("primary array ,") // consume ','
				}
			}

			if p.this().Type != tokens.RBrace {
				p.error(arrayToken, "array literal is missing closing }", "primary")
				return ast.ZeroExprIndex
			}

			p.advance("primary array }") // consume }

			return p.ast.NewArrayLiteral(arrayToken, t, values)
		case *types.Map:
			mapToken := p.this()
			pairs := make([]ast.KeyValue, 0, mapLiteralPreallocationSize)

			p.advance("primary map {") // consume {

			for !p.match(tokens.RBrace, tokens.EOF) {
				if ctx.Err() != nil {
					return ast.ZeroExprIndex
				}

				key := p.expression(ctx, t.Key)
				if key != ast.ZeroExprIndex {
					// TODO: optimize
					for i := range pairs {
						if p.ExprString(pairs[i].Key) == p.ExprString(key) {
							p.error(p.prev(), "duplicate key in map literal", "primary")
							return ast.ZeroExprIndex
						}
					}
				}

				if p.this().Type != tokens.Colon {
					p.error(p.this(), "expected colon after key in map literal", "primary")
					return ast.ZeroExprIndex
				}

				p.advance("primary map :") // consume :

				val := p.expression(ctx, t.Value)
				if val == ast.ZeroExprIndex {
					return ast.ZeroExprIndex
				}

				pairs = append(pairs, ast.KeyValue{
					Key:   key,
					Value: val,
				})

				if p.this().Type == tokens.Comma {
					p.advance("primary map ,") // consume ,
				}
			}

			if p.this().Type != tokens.RBrace {
				p.error(mapToken, "map literal is missing closing }", "primary")
				return ast.ZeroExprIndex
			}

			p.advance("primary map }") // consume }

			return p.ast.NewMapLiteral(mapToken, t, pairs)
		case *types.Procedure:
			procToken := p.this()

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
				return ast.ZeroExprIndex
			}

			return p.ast.NewProcedureLiteral(procToken, t, body)
		case *types.Set:
			setToken := p.this()
			values := make([]ast.ExprIndex, 0, setLiteralPreallocationSize)

			p.advance("primary set {") // consume {

			for !p.match(tokens.RBrace, tokens.EOF) {
				if ctx.Err() != nil {
					return ast.ZeroExprIndex
				}

				value := p.expression(ctx, t.Element)
				if value != ast.ZeroExprIndex {
					for i := range values {
						// TODO: optimize
						if p.ExprString(values[i]) == p.ExprString(value) {
							p.error(p.prev(), "duplicate key in set literal", "primary")
							return ast.ZeroExprIndex
						}
					}

					values = append(values, value)
				}

				if p.this().Type == tokens.Comma {
					p.advance("primary set ,") // consume ','
				}
			}

			if p.this().Type != tokens.RBrace {
				p.error(setToken, "set literal is missing closing }", "primary")
				return ast.ZeroExprIndex
			}

			p.advance("primary set }") // consume }

			return p.ast.NewSetLiteral(setToken, t, values)
		case *types.Slice:
			sliceToken := p.this()
			values := make([]ast.ExprIndex, 0, sliceLiteralPreallocationSize)

			p.advance("primary slice {") // consume {

			for !p.match(tokens.RBrace, tokens.EOF) {
				if ctx.Err() != nil {
					return ast.ZeroExprIndex
				}

				value := p.expression(ctx, t.Element)
				if value != ast.ZeroExprIndex {
					values = append(values, value)
				}

				if p.this().Type == tokens.Comma {
					p.advance("primary array ,") // consume ','
				}
			}

			if p.this().Type != tokens.RBrace {
				p.error(sliceToken, "slice literal is missing closing }", "primary")
				return ast.ZeroExprIndex
			}

			p.advance("primary slice }") // consume }

			return p.ast.NewSliceLiteral(sliceToken, t, values)
		case *types.Struct:
			structToken := p.this()
			values := make([]ast.FieldValue, 0, len(t.Fields))

			p.advance("primary struct {") // consume {

			for !p.match(tokens.RBrace, tokens.EOF) {
				if ctx.Err() != nil {
					return ast.ZeroExprIndex
				}

				if p.this().Type != tokens.Identifier {
					p.error(p.this(), "expected identifier at in struct literal", "primary")
					return ast.ZeroExprIndex
				}

				index := slices.IndexFunc(t.Fields, func(f *types.Field) bool {
					return f.Name == p.this().Literal
				})

				if index == -1 {
					p.error(p.this(), "unknown field found in struct literal", "primary")
					return ast.ZeroExprIndex
				}

				fieldValue := ast.FieldValue{
					Name: p.this().Literal,
				}

				p.advance("primary struct identifier") // consume identifier

				if p.this().Type != tokens.Assign {
					p.error(p.this(), "expected = after identifier in struct literal", "primary")
					return ast.ZeroExprIndex
				}

				p.advance("primary struct =") // consume =

				startToken := p.this()

				value := p.expression(ctx, t.Fields[index].Type)
				if value == ast.ZeroExprIndex {
					p.error(startToken, "failed to parse field expression in struct literal", "primary")
					return ast.ZeroExprIndex
				}

				fieldValue.Value = value

				if p.this().Type == tokens.Comma {
					p.advance("primary struct ,") // consume ','
				}

				values = append(values, fieldValue)
			}

			if p.this().Type != tokens.RBrace {
				p.error(structToken, "struct literal is missing closing }", "primary")
				return ast.ZeroExprIndex
			}

			p.advance("primary struct }") // consume }

			return p.ast.NewStructLiteral(structToken, t, values)
		case *types.Tuple:
			tupleToken := p.this()
			values := make([]ast.ExprIndex, 0, len(t.Types))

			p.advance("primary tuple {") // consume {

			for i := range t.Types {
				startToken := p.this()

				value := p.expression(ctx, t.Index(i))
				if value == ast.ZeroExprIndex {
					p.error(startToken, "failed to parse expression in tuple literal", "primary")
					return ast.ZeroExprIndex
				}

				values = append(values, value)

				if i < len(t.Types)-1 {
					if p.this().Type != tokens.Comma {
						p.error(p.this(), "expected , after expression in tuple literal", "primary")
						return ast.ZeroExprIndex
					}

					p.advance("primary tuple ,") // consume ','
				}
			}

			if p.this().Type != tokens.RBrace {
				p.error(tupleToken, "tuple literal is missing closing }", "primary")
				return ast.ZeroExprIndex
			}

			p.advance("primary tuple }") // consume }

			return p.ast.NewTupleLiteral(tupleToken, t, values)
		case *types.Basic:
			if t.Kind() != types.Complex32 {
				p.error(p.this(), fmt.Sprintf("unexpected basic type %q for expression starting with {", t.String()), "primary")
				return ast.ZeroExprIndex
			}

			token := p.this()
			p.advance("primary complex32 {") // consume {

			realPart := p.expression(ctx, types.Basics[types.Float16])
			if realPart == ast.ZeroExprIndex {
				return ast.ZeroExprIndex
			}

			if p.this().Type != tokens.Comma {
				p.error(p.this(), "expected , after real part in complex32 literal", "primary")
				return ast.ZeroExprIndex
			}

			p.advance("primary complex32 ,") // consume ,

			imagPart := p.expression(ctx, types.Basics[types.Float16])
			if imagPart == ast.ZeroExprIndex {
				return ast.ZeroExprIndex
			}

			if p.this().Type != tokens.RBrace {
				p.error(p.this(), "expected } after imaginary part in complex32 literal", "primary")
				return ast.ZeroExprIndex
			}

			p.advance("primary complex32 }") // consume }

			realLit, realOk := p.ast.Expr(realPart).(*ast.Float16Literal)
			imagLit, imagOk := p.ast.Expr(imagPart).(*ast.Float16Literal)

			if !realOk || !imagOk {
				p.error(token, "complex32 literal requires float16 literal values", "primary")
				return ast.ZeroExprIndex
			}

			return p.ast.NewComplex32Literal(token, ast.Complex32{realLit.Value, imagLit.Value})
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

				return ast.ZeroExprIndex
			}

			p.error(p.this(), fmt.Sprintf("unexpected type %q for expression starting with {", typeToken.String()), "primary")

			return ast.ZeroExprIndex
		}
	default:
		p.error(p.this(), "unexpected token encountered while parsing expression", "primary")
		return ast.ZeroExprIndex
	}
}

func (p *Parser) match(types ...tokens.Type) bool {
	return slices.Contains(types, p.this().Type)
}
