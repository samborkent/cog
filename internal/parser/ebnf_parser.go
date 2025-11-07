package parser

import (
	"context"
	"fmt"
	"slices"

	"github.com/samborkent/cog/internal/ast"
	"github.com/samborkent/cog/internal/tokens"
	"github.com/samborkent/cog/internal/types"
)

func (p *Parser) expression(ctx context.Context, typeToken types.Type) ast.Expression {
	expr := p.equality(ctx, typeToken)

	for p.match(tokens.And, tokens.Or) {
		if ctx.Err() != nil || expr == nil {
			return nil
		}

		if !types.IsBool(expr.Type()) {
			fmt.Println(expr.String())

			p.error(p.this(), "operator requires bool type", "expression")
			return nil
		}

		operator := p.this()
		p.advance("expression operator") // consume operator
		right := p.equality(ctx, types.Basics[types.Bool])

		expr = &ast.Infix{
			Operator: operator,
			Left:     expr,
			Right:    right,
		}
	}

	if expr != nil {
		return expr
	}

	return nil
}

func (p *Parser) equality(ctx context.Context, typeToken types.Type) ast.Expression {
	expr := p.comparison(ctx, typeToken)

	for p.match(tokens.Equal, tokens.NotEqual) {
		if ctx.Err() != nil || expr == nil {
			return nil
		}

		operator := p.this()
		p.advance("equality operator") // consume operator
		right := p.comparison(ctx, types.None)

		infix := &ast.Infix{
			Operator: operator,
			Left:     expr,
			Right:    right,
		}

		if infix.Left.Type().Underlying().Kind() != infix.Right.Type().Underlying().Kind() {
			infix.EqualizeLiteralTypes()
		}

		expr = infix
	}

	if expr != nil {
		return expr
	}

	return nil
}

func (p *Parser) comparison(ctx context.Context, typeToken types.Type) ast.Expression {
	expr := p.term(ctx, typeToken)

	for p.match(tokens.GT, tokens.GTEqual, tokens.LT, tokens.LTEqual) {
		if ctx.Err() != nil || expr == nil {
			return nil
		}

		if !types.IsNumber(expr.Type()) {
			p.error(p.this(), "operator requires numeric type", "comparison")
			return nil
		}

		operator := p.this()
		p.advance("comparison operator") // consume operator
		right := p.term(ctx, types.None)

		infix := &ast.Infix{
			Operator: operator,
			Left:     expr,
			Right:    right,
		}

		if infix.Left.Type().Underlying().Kind() != infix.Right.Type().Underlying().Kind() {
			infix.EqualizeLiteralTypes()
		}

		expr = infix
	}

	if expr != nil {
		return expr
	}

	return nil
}

func (p *Parser) term(ctx context.Context, typeToken types.Type) ast.Expression {
	expr := p.factor(ctx, typeToken)

	for p.match(tokens.Minus, tokens.Plus) {
		if ctx.Err() != nil || expr == nil {
			return nil
		}

		// TODO: this is a hack due to lack of known Go typing at compile time, figure out a better solution.
		if expr.Type() != types.None {
			if p.this().Type == tokens.Plus {
				if !types.IsSummable(expr.Type()) {
					p.error(p.this(), fmt.Sprintf("operator requires numeric or string type, got %q", expr.Type()), "term")
					return nil
				}
			} else {
				// Minus
				if !types.IsNumber(expr.Type()) {
					p.error(p.this(), fmt.Sprintf("operator requires numeric type, got %q", expr.Type()), "term")
					return nil
				}
			}
		}

		operator := p.this()
		p.advance("term operator") // consume operator
		right := p.factor(ctx, expr.Type())

		expr = &ast.Infix{
			Operator: operator,
			Left:     expr,
			Right:    right,
		}
	}

	if expr != nil {
		return expr
	}

	return nil
}

func (p *Parser) factor(ctx context.Context, typeToken types.Type) ast.Expression {
	expr := p.unary(ctx, typeToken)

	for p.match(tokens.Asterisk, tokens.Divide) {
		if ctx.Err() != nil || expr == nil {
			return nil
		}

		if !types.IsNumber(expr.Type()) {
			p.error(p.this(), "operator requires numeric type", "factor")
			return nil
		}

		operator := p.this()
		p.advance("factor operator") // consume operator
		right := p.unary(ctx, expr.Type())

		expr = &ast.Infix{
			Operator: operator,
			Left:     expr,
			Right:    right,
		}
	}

	if expr != nil {
		return expr
	}

	return nil
}

func (p *Parser) unary(ctx context.Context, typeToken types.Type) ast.Expression {
	if p.match(tokens.Not, tokens.Minus) {
		operator := p.this()
		p.advance("unary operator") // consume operator
		right := p.unary(ctx, typeToken)

		if right == nil {
			return nil
		}

		if operator.Type == tokens.Not && !types.IsBool(right.Type()) {
			p.error(p.this(), "operator requires bool type", "unary")
			return nil
		} else if operator.Type == tokens.Minus && !types.IsNumber(right.Type()) {
			p.error(p.this(), "operator requires numeric type", "unary")
			return nil
		}

		return &ast.Prefix{
			Operator: operator,
			Right:    right,
		}
	}

	if (typeToken == nil || typeToken == types.None) && p.this().Type == tokens.Identifier {
		// TODO: get rid of double lookup for identifiers
		symbol, ok := p.symbols.Resolve(p.this().Literal)
		if !ok {
			p.error(p.this(), "undefined identifier", "primary")
			return nil
		}

		typeToken = symbol.Type()
	}

	node := p.primary(ctx, typeToken)
	if node == nil {
		return nil
	}

	if p.this().Type == tokens.Question {
		token := p.this()
		p.advance("unary ?") // consume ?

		if typeToken.Kind() != types.OptionKind {
			p.error(token, "option operator requires option type", "unary")
			return nil
		}

		return &ast.Suffix{
			Operator: token,
			Left:     node,
		}
	}

	return node
}

func (p *Parser) primary(ctx context.Context, typeToken types.Type) ast.Expression {
	if typeToken != nil {
		aliasType, ok := typeToken.(*types.Alias)
		if ok {
			typeToken = aliasType.Underlying()
		}

		if typeToken.Kind() == types.OptionKind {
			// Handle option literal.
			optionType, ok := typeToken.(*types.Option)
			if !ok {
				panic("unable to assert option type")
			}

			// TODO: handle none type

			typeToken = optionType.Value
		}

		if typeToken.Kind() == types.UnionKind {
			// Handle union literal.
			unionType, ok := typeToken.(*types.Union)
			if !ok {
				panic("unable to assert union type")
			}

			token := p.this()

			// Infer type.
			expr := p.primary(ctx, types.None)
			if expr == nil {
				return nil
			}

			isEither := expr.Type().Kind() == unionType.Either.Kind()
			isOr := expr.Type().Kind() == unionType.Or.Kind()

			if !isEither && !isOr {
				p.error(p.this(), fmt.Sprintf("expression of type %q not in union type %q", expr.Type().String(), unionType.String()), "primary")
				return nil
			}

			return &ast.UnionLiteral{
				Token:     token,
				UnionType: unionType,
				Value:     expr,
				Tag:       isOr,
			}
		}
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
			return nil
		}

		return builtinParser(ctx, t, typeToken)
	case tokens.FloatLiteral,
		tokens.IntLiteral,
		tokens.StringLiteral:
		return p.parseLiteral(typeToken)
	case tokens.False:
		node := &ast.BoolLiteral{
			Token: p.this(),
		}

		p.advance("primary literal") // consume literal

		return node
	case tokens.True:
		node := &ast.BoolLiteral{
			Token: p.this(),
			Value: true,
		}

		p.advance("primary literal") // consume literal

		return node
	case tokens.LParen: // Grouped expression
		p.advance("primary (") // consume '('

		expr := p.expression(ctx, typeToken)

		if p.this().Type != tokens.RParen {
			p.error(p.this(), "expected ')' after grouped expression", "primary")
			return nil
		}

		p.advance("primary )") // consume ')'

		if expr != nil {
			return expr
		}

		return nil
	case tokens.Identifier:
		symbol, ok := p.symbols.Resolve(p.this().Literal)
		if !ok {
			p.error(p.this(), "undefined identifier", "primary")
			return nil
		}

		p.advance("primary identifier") // consume identifier

		switch p.this().Type {
		case tokens.LParen:
			// Function call
			procType, ok := symbol.Identifier.ValueType.(*types.Procedure)
			if !ok {
				panic("unable to cast to procedure type")
			}

			return &ast.Call{
				Identifier: symbol.Identifier,
				Arguments:  p.parseCallArguments(ctx, procType),
				ReturnType: procType.ReturnType,
			}
		case tokens.Dot:
			// TODO: recursive selector expression

			// Selector expression
			selector := p.this()

			p.advance("primary identifier .") // consume .

			if p.this().Type != tokens.Identifier {
				p.error(p.this(), "expected field identifier after . selector", "primary")
				return nil
			}

			field, ok := p.symbols.ResolveField(symbol.Identifier.Name, p.this().Literal)
			if !ok {
				p.error(p.this(), "undefined field", "primary")
				return nil
			}

			field.Identifier.Token = p.this()

			p.advance("primary identifier field") // consume field identifier

			// Make field type equal to selector type and wrap in alias, so we can infer enum type.
			field.Identifier.ValueType = &types.Alias{
				Name:     symbol.Identifier.Name,
				Derived:  symbol.Type(),
				Exported: symbol.Identifier.Exported,
			}

			return &ast.Selector{
				Token:      selector,
				Identifier: symbol.Identifier,
				Field:      field.Identifier,
			}
		default:
			// Variable reference
			if symbol.Identifier == nil {
				panic("nil identifier in variable reference")
			}

			return symbol.Identifier
		}
	case tokens.LBrace:
		switch t := typeToken.(type) {
		case *types.Alias:
			expr := p.primary(ctx, t.Derived)
			if expr == nil {
				return nil
			}

			// Place back type alias
			switch literal := expr.(type) {
			case *ast.ArrayLiteral:
				literal.ArrayType = t.Derived.Underlying().(*types.Array)
				return literal
			case *ast.MapLiteral:
				literal.ValueType = t
				return literal
			case *ast.SetLiteral:
				literal.ValueType = t
				return literal
			case *ast.StructLiteral:
				literal.StructType = t
				return literal
			case *ast.TupleLiteral:
				literal.TupleType = t
				return literal
			case *ast.UnionLiteral:
				literal.UnionType = t
				return literal
			}

			return expr
		case *types.Array:
			// TODO: see if it's possible to evaluate array length
			arrayLiteral := &ast.ArrayLiteral{
				Token:     p.this(),
				ArrayType: t,
				Values:    []ast.Expression{},
			}

			p.advance("primary array {") // consume {

			for !p.match(tokens.RBrace, tokens.EOF) {
				if ctx.Err() != nil {
					return nil
				}

				value := p.expression(ctx, t.Element)
				if value != nil {
					arrayLiteral.Values = append(arrayLiteral.Values, value)
				}

				if p.this().Type == tokens.Comma {
					p.advance("primary array ,") // consume ','
				}
			}

			if p.this().Type != tokens.RBrace {
				p.error(arrayLiteral.Token, "array literal is missing closing }", "primary")
				return nil
			}

			p.advance("primary array }") // consume }

			return arrayLiteral
		case *types.Map:
			mapLiteral := &ast.MapLiteral{
				Token:     p.this(),
				KeyType:   t.Key,
				ValueType: t.Value,
				Pairs:     []*ast.KeyValue{},
			}

			p.advance("primary map {") // consume {

			for !p.match(tokens.RBrace, tokens.EOF) {
				if ctx.Err() != nil {
					return nil
				}

				key := p.expression(ctx, t.Key)
				if key != nil {
					// TODO: optimize
					for i := range mapLiteral.Pairs {
						if mapLiteral.Pairs[i].Key.String() == key.String() {
							p.error(p.prev(), "duplicate key in map literal", "primary")
							return nil
						}
					}
				}

				if p.this().Type != tokens.Colon {
					p.error(p.this(), "expected colon after key in map literal", "primary")
					return nil
				}

				p.advance("primary map :") // consume :

				val := p.expression(ctx, t.Value)
				if val == nil {
					return nil
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
				return nil
			}

			p.advance("primary map }") // consume }

			return mapLiteral
		case *types.Procedure:
			procLiteral := &ast.ProcedureLiteral{
				ProcedureType: t,
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

			body := p.parseBlockStatement(ctx)

			if len(t.Parameters) > 0 {
				// Leave parameter scope
				p.symbols = p.symbols.Outer
			}

			if body == nil {
				return nil
			}

			procLiteral.Body = body

			return procLiteral
		case *types.Set:
			setLiteral := &ast.SetLiteral{
				Token:     p.this(),
				ValueType: t.Element,
				Values:    []ast.Expression{},
			}

			p.advance("primary set {") // consume {

			for !p.match(tokens.RBrace, tokens.EOF) {
				if ctx.Err() != nil {
					return nil
				}

				value := p.expression(ctx, t.Element)
				if value != nil {
					for i := range setLiteral.Values {
						if setLiteral.Values[i].String() == value.String() {
							p.error(p.prev(), "duplicate key in set literal", "primary")
							return nil
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
				return nil
			}

			p.advance("primary set }") // consume }

			return setLiteral
		case *types.Slice:
			sliceLiteral := &ast.SliceLiteral{
				Token:       p.this(),
				ElementType: t.Element,
				Values:      []ast.Expression{},
			}

			p.advance("primary slice {") // consume {

			for !p.match(tokens.RBrace, tokens.EOF) {
				if ctx.Err() != nil {
					return nil
				}

				value := p.expression(ctx, t.Element)
				if value != nil {
					sliceLiteral.Values = append(sliceLiteral.Values, value)
				}

				if p.this().Type == tokens.Comma {
					p.advance("primary array ,") // consume ','
				}
			}

			if p.this().Type != tokens.RBrace {
				p.error(sliceLiteral.Token, "slice literal is missing closing }", "primary")
				return nil
			}

			p.advance("primary slice }") // consume }

			return sliceLiteral
		case *types.Struct:
			structLiteral := &ast.StructLiteral{
				Token:      p.this(),
				StructType: t,
				Values:     make([]*ast.FieldValue, 0, len(t.Fields)),
			}

			p.advance("primary struct {") // consume {

			for !p.match(tokens.RBrace, tokens.EOF) {
				if ctx.Err() != nil {
					return nil
				}

				if p.this().Type != tokens.Identifier {
					p.error(p.this(), "expected identifier at in struct literal", "primary")
					return nil
				}

				index := slices.IndexFunc(t.Fields, func(f *types.Field) bool {
					return f.Name == p.this().Literal
				})

				if index == -1 {
					p.error(p.this(), "unknown field found in struct literal", "primary")
					return nil
				}

				fieldValue := &ast.FieldValue{
					Name: p.this().Literal,
				}

				p.advance("primary struct identifier") // consume identifier

				if p.this().Type != tokens.Assign {
					p.error(p.this(), "expected = after identifier in struct literal", "primary")
					return nil
				}

				p.advance("primary struct =") // consume =

				startToken := p.this()

				value := p.expression(ctx, t.Fields[index].Type)
				if value == nil {
					p.error(startToken, "failed to parse field expression in struct literal", "primary")
					return nil
				}

				fieldValue.Value = value

				if p.this().Type == tokens.Comma {
					p.advance("primary struct ,") // consume ','
				}

				structLiteral.Values = append(structLiteral.Values, fieldValue)
			}

			if p.this().Type != tokens.RBrace {
				p.error(structLiteral.Token, "struct literal is missing closing }", "primary")
				return nil
			}

			p.advance("primary struct }") // consume }

			return structLiteral
		case *types.Tuple:
			tupleLiteral := &ast.TupleLiteral{
				Token:     p.this(),
				TupleType: t,
				Values:    make([]ast.Expression, 0, len(t.Types)),
			}

			p.advance("primary tuple {") // consume {

			for i := range t.Types {
				startToken := p.this()

				value := p.expression(ctx, t.Index(i))
				if value == nil {
					p.error(startToken, "failed to parse expression in tuple literal", "primary")
					return nil
				}

				tupleLiteral.Values = append(tupleLiteral.Values, value)

				if i < len(t.Types)-1 {
					if p.this().Type != tokens.Comma {
						p.error(p.this(), "expected , after expression in tuple literal", "primary")
						return nil
					}

					p.advance("primary tuple ,") // consume ','
				}
			}

			if p.this().Type != tokens.RBrace {
				p.error(tupleLiteral.Token, "tuple literal is missing closing }", "primary")
				return nil
			}

			p.advance("primary tuple }") // consume }

			return tupleLiteral
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

				return nil
			}

			p.error(p.this(), fmt.Sprintf("unexpected type %q for expression starting with {", typeToken.String()), "primary")
			return nil
		}
	default:
		p.error(p.this(), "unexpected token encountered while parsing expression", "primary")
		return nil
	}
}

func (p *Parser) match(types ...tokens.Type) bool {
	return slices.Contains(types, p.this().Type)
}
