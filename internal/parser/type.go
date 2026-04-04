package parser

import (
	"context"
	"fmt"

	"github.com/samborkent/cog/internal/ast"
	"github.com/samborkent/cog/internal/tokens"
	"github.com/samborkent/cog/internal/types"
)

func (p *Parser) parseCombinedType(ctx context.Context, exported, global bool) types.Type {
	switch p.this().Type {
	case tokens.Enum:
		ident := &ast.Identifier{
			Token:    p.tokens[p.i-2],
			Name:     p.tokens[p.i-2].Literal,
			Exported: exported,
			Global:   global,
		}

		return p.parseEnumType(ctx, ident)
	case tokens.Error:
		ident := &ast.Identifier{
			Token:    p.tokens[p.i-2],
			Name:     p.tokens[p.i-2].Literal,
			Exported: exported,
			Global:   global,
		}

		return p.parseErrorType(ctx, ident)
	case tokens.Function, tokens.Procedure:
		return p.parseProcedureType(ctx, exported, global)
	}

	typ := p.parseType(ctx)

	switch p.this().Type {
	case tokens.BitAnd:
		// Tuple
		tuple := &types.Tuple{
			Types:    make([]types.Type, 1, types.TupleMaxTypes),
			Exported: exported,
			Global:   global,
		}

		// Put parsed type as first type.
		tuple.Types[0] = typ

		for p.this().Type == tokens.BitAnd {
			p.advance("parseCombinedType tuple &") // consume &

			next := p.parseType(ctx)
			if next != nil {
				tuple.Types = append(tuple.Types, next)
			}
		}

		return tuple
	case tokens.Pipe:
		// Union
		union := &types.Union{
			Variants: []types.Type{typ},
			Exported: exported,
		}

		for p.this().Type == tokens.Pipe {
			p.advance("parseCombinedType union |") // consume |

			next := p.parseType(ctx)
			if next != nil {
				union.Variants = append(union.Variants, next)
			}
		}

		return union
	case tokens.Not:
		// Result type: T ! E
		p.advance("parseCombinedType result !") // consume !

		errorType := p.parseCombinedType(ctx, exported, global)
		if errorType == nil {
			return nil
		}

		if errorType.Kind() != types.ErrorKind {
			p.error(p.prev(), "result error type must be an error type", "parseCombinedType")
			return nil
		}

		if typ.Kind() == types.ErrorKind {
			p.error(p.prev(), "result value type cannot be an error type", "parseCombinedType")
			return nil
		}

		return &types.Result{
			Value: typ,
			Error: errorType,
		}
	}

	return typ
}

func (p *Parser) parseType(ctx context.Context) types.Type {
	switch p.this().Type {
	case tokens.LBracket:
		p.advance("parseType [") // consume [

		if p.this().Type == tokens.RBracket {
			// Slice type
			if p.this().Type != tokens.RBracket {
				p.error(p.this(), "expected closing ] in slice type", "parseType")
				return nil
			}

			p.advance("parseType ]") // consume ]

			elemType := p.parseType(ctx)
			if elemType == nil {
				return nil
			}

			return &types.Slice{
				Element: elemType,
			}
		}

		// Array type
		switch p.this().Type {
		case tokens.IntLiteral:
		case tokens.Identifier:
			symbol, ok := p.symbols.Resolve(p.this().Literal)
			if ok && types.IsFixed(symbol.Identifier.ValueType) {
				break
			}

			fallthrough
		default:
			p.error(p.this(), "expected fixed-point number type as array length", "parseCombinedType")
			return nil
		}

		lenExpr := p.expression(ctx, types.None)
		if lenExpr == nil {
			return nil
		}

		if p.this().Type != tokens.RBracket {
			p.error(p.this(), "expected closing ] in array type", "parseType")
			return nil
		}

		p.advance("parseType ]") // consume ]

		elemType := p.parseType(ctx)
		if elemType == nil {
			return nil
		}

		return &types.Array{
			Element: elemType,
			Length:  lenExpr,
		}
	case tokens.Map:
		p.advance("parseType map") // consume map

		if p.this().Type != tokens.LT {
			p.error(p.this(), "expected < after map type", "parseType")
			return nil
		}

		p.advance("parseType map <") // consume <

		keyType := p.parseType(ctx)
		if keyType == nil {
			return nil
		}

		if p.this().Type != tokens.Comma {
			p.error(p.this(), "expected , after map key type", "parseType")
			return nil
		}

		p.advance("parseType map ,") // consume ,

		valType := p.parseType(ctx)
		if valType == nil {
			return nil
		}

		if p.this().Type != tokens.GT {
			p.error(p.this(), "expected > after map value type", "parseType")
			return nil
		}

		p.advance("parseType map >") // consume >

		return &types.Map{
			Key:   keyType,
			Value: valType,
		}
	case tokens.Set:
		p.advance("parseType set") // consume set

		if p.this().Type != tokens.LT {
			p.error(p.this(), "expected < after set type", "parseType")
			return nil
		}

		p.advance("parseType set <") // consume <

		elemType := p.parseType(ctx)
		if elemType == nil {
			return nil
		}

		if p.this().Type != tokens.GT {
			p.error(p.this(), "expected > after set element type", "parseType")
			return nil
		}

		p.advance("parseType set >") // consume >

		return &types.Set{Element: elemType}
	case tokens.Struct:
		return p.parseStruct(ctx)
	case tokens.BitAnd:
		// Reference type parsing
		p.advance("parseType &") // consume &

		valType := p.parseType(ctx)
		if valType == nil {
			return nil
		}

		// TODO: check if this is correct.
		if types.IsPointer(valType) {
			p.error(p.this(), fmt.Sprintf("reference of pointer type %q not allowed", valType.Kind()), "parseType")
			return nil
		}

		return &types.Reference{
			Value: valType,
		}
	}

	typ, ok := types.Lookup[p.this().Type]
	if !ok {
		// Check for imported package type: pkg.Type
		if p.this().Type == tokens.Identifier && p.next().Type == tokens.Dot {
			if imp, isImport := p.symbols.ResolveCogImport(p.this().Literal); isImport {
				p.advance("parseType pkg") // consume package name
				p.advance("parseType .")   // consume '.'

				if p.this().Type != tokens.Identifier {
					p.error(p.this(), "expected type name after package selector", "parseType")
					return nil
				}

				sym, found := imp.Exports[p.this().Literal]
				if !found || sym.Identifier.Qualifier != ast.QualifierType {
					p.error(p.this(), fmt.Sprintf("package %q has no exported type %q", imp.Name, p.this().Literal), "parseType")
					return nil
				}

				ident := sym.Identifier
				if types.IsNone(ident.ValueType) {
					typ = types.NewForwardAlias(ident.Name, ident.Exported, ident.Global, func() types.Type {
						return ident.ValueType
					})
				} else {
					typ = &types.Alias{
						Name:     ident.Name,
						Derived:  ident.ValueType,
						Exported: ident.Exported,
						Global:   ident.Global,
					}
				}

				p.advance("parseType pkg type") // consume type name

				if p.this().Type == tokens.Question {
					p.advance("parseType ?") // consume ?

					if typ.Kind() == types.OptionKind {
						p.error(p.this(), "nested optional types are not allowed", "parseType")
						return nil
					}

					return &types.Option{Value: typ}
				}

				return typ
			}
		}

		// Non-basic type, try to find in symbol table.
		typeSymbol, ok := p.symbols.Resolve(p.this().Literal)
		if !ok || typeSymbol.Identifier.Qualifier != ast.QualifierType {
			p.error(p.this(), "unknown type found in type declaration", "parseType")
			return nil
		}

		ident := typeSymbol.Identifier

		// If the symbol is a type parameter (inside a generic alias body),
		// return the TypeParam directly.
		if tp, ok := ident.ValueType.(*types.TypeParam); ok {
			p.advance("parseType typeparam") // consume type param name
			return tp
		}

		if types.IsNone(ident.ValueType) {
			// Forward reference: type name is pre-registered but not yet resolved.
			// Create a lazy alias that resolves when the type is accessed.
			typ = types.NewForwardAlias(ident.Name, ident.Exported, ident.Global, func() types.Type {
				return ident.ValueType
			})
		} else {
			// Copy type parameters from the original type if it's an alias
			var typeParams []*types.TypeParam
			if originalAlias, ok := ident.ValueType.(*types.Alias); ok {
				typeParams = originalAlias.TypeParams
			}

			typ = &types.Alias{
				Name:       ident.Name,
				Derived:    ident.ValueType,
				Exported:   ident.Exported,
				Global:     ident.Global,
				TypeParams: typeParams,
			}
		}
	}

	p.advance("parseType type") // consume type

	// Check for generic instantiation: Alias<int32, utf8>
	if p.this().Type == tokens.LT {
		typ = p.instantiateGenericAlias(ctx, typ)
		if typ == nil {
			return nil
		}
	}

	if p.this().Type == tokens.Question {
		// Optional type
		p.advance("parseType ?") // consume ?

		if typ.Kind() == types.OptionKind {
			p.error(p.this(), "nested optional types are not allowed", "parseType")
			return nil
		}

		return &types.Option{
			Value: typ,
		}
	}

	return typ
}

// instantiateGenericAlias parses type arguments after a generic alias reference
// and produces the instantiated concrete type. The current token must be '<'.
func (p *Parser) instantiateGenericAlias(ctx context.Context, typ types.Type) types.Type {
	alias, ok := typ.(*types.Alias)
	if !ok {
		p.error(p.this(), "type arguments on non-alias type", "instantiateGenericAlias")
		return nil
	}

	// Resolve the alias to find the generic definition with TypeParams.
	var genAlias *types.Alias

	if alias.Derived != nil && !types.IsNone(alias.Derived) {
		if a, ok := alias.Derived.(*types.Alias); ok && len(a.TypeParams) > 0 {
			genAlias = a
		}
	}

	if genAlias == nil {
		// The alias itself may carry TypeParams (for direct resolutions).
		if len(alias.TypeParams) > 0 {
			genAlias = alias
		}
	}

	if genAlias == nil {
		// Try resolving the underlying value type (set during findGlobalType).
		switch v := alias.Derived.(type) {
		case *types.Alias:
			if len(v.TypeParams) > 0 {
				genAlias = v
			}
		}
	}

	if genAlias == nil {
		p.error(p.this(), fmt.Sprintf("type %q is not generic", alias.Name), "instantiateGenericAlias")
		return nil
	}

	typeArgs := p.parseTypeArguments(ctx)
	if typeArgs == nil {
		return nil
	}

	if len(typeArgs) != len(genAlias.TypeParams) {
		p.error(p.this(), fmt.Sprintf("wrong number of type arguments for %q: expected %d, got %d",
			alias.Name, len(genAlias.TypeParams), len(typeArgs)), "instantiateGenericAlias")

		return nil
	}

	// Check constraint satisfaction.
	for i, arg := range typeArgs {
		tp := genAlias.TypeParams[i]
		if !tp.SatisfiedBy(arg) {
			p.error(p.this(), fmt.Sprintf("type argument %q does not satisfy constraint %q for parameter %q",
				arg.String(), tp.ConstraintString(), tp.Name), "instantiateGenericAlias")

			return nil
		}
	}

	// Build substitution map and instantiate.
	argMap := make(map[string]types.Type, len(typeArgs))
	for i, tp := range genAlias.TypeParams {
		argMap[tp.Name] = typeArgs[i]
	}

	return genAlias.Instantiate(argMap)
}

func (p *Parser) parseStruct(ctx context.Context) types.Type {
	p.advance("parseStruct struct") // consume struct

	if p.this().Type != tokens.LBrace {
		p.error(p.this(), "expected { after struct declaration", "parseStruct")
		return nil
	}

	p.advance("parseStruct {") // consume {

	fields := []*types.Field{}

	for p.this().Type != tokens.RBrace {
		if ctx.Err() != nil {
			return nil
		}

		switch p.this().Type {
		case tokens.Export:
			p.advance("parseStruct export") // consume export

			if p.this().Type == tokens.LParen {
				p.advance("parseStruct export (") // consume (

				for p.this().Type != tokens.RParen {
					field := p.parseField(ctx, true)
					if field == nil {
						return nil
					}

					fields = append(fields, field)
				}

				p.advance("parseStruct export )") // consume )

				continue
			}

			field := p.parseField(ctx, true)
			if field == nil {
				return nil
			}

			fields = append(fields, field)
		case tokens.Identifier:
			field := p.parseField(ctx, false)
			if field == nil {
				return nil
			}

			fields = append(fields, field)
		default:
			p.error(p.this(), "unexpected token found in struct declaration", "parseStruct")
			return nil
		}
	}

	p.advance("parseStruct }") // consume }

	return &types.Struct{
		Fields: fields,
	}
}

func (p *Parser) parseField(ctx context.Context, exported bool) *types.Field {
	field := &types.Field{
		Name:     p.this().Literal,
		Exported: exported,
	}

	p.advance("parseField identifier") // consume identifier

	if p.this().Type != tokens.Colon {
		p.error(p.this(), "expected : after field name in struct declaration", "parseStruct")
		return nil
	}

	p.advance("parseField :") // consume :

	field.Type = p.parseCombinedType(ctx, exported, false)

	return field
}

func (p *Parser) parseProcedureType(ctx context.Context, exported, global bool) *types.Procedure {
	procType := &types.Procedure{
		Function:   p.this().Type == tokens.Function,
		Parameters: make([]*types.Parameter, 0),
	}

	p.advance("parseProcedureType proc/func")

	if p.this().Type == tokens.LT {
		procType.TypeParams = p.parseTypeParams(ctx)
		if procType.TypeParams == nil {
			return nil
		}

		// Enter scope for type parameters.
		p.symbols = NewEnclosedSymbolTable(p.symbols)

		// Pre-register type parameters in symbol table for recursive references.
		for _, tp := range procType.TypeParams {
			p.symbols.Define(&ast.Identifier{
				Name:      tp.Name,
				ValueType: tp,
				Qualifier: ast.QualifierType,
			})
		}

		defer func() {
			// Exit type parameter scope.
			p.symbols = p.symbols.Outer
		}()
	}

	if p.this().Type != tokens.LParen {
		p.error(p.this(), fmt.Sprintf("expected '(' after %q in type", p.prev().Type), "parseProcedureType")
		return nil
	}

	p.advance("parseProcedureType (") // consume (

	// Flag to keep track of if any of the parameters is optional.
	// When a parameter is marked as optional, all following parameters must also be optional.
	haveOptional := false

	for i := 0; !p.match(tokens.RParen, tokens.EOF); i++ {
		if ctx.Err() != nil {
			return nil
		}

		if p.this().Type != tokens.Identifier {
			p.error(p.this(), "expected parameter identifier", "parseParameters")
			return nil
		}

		param := &types.Parameter{
			Name: p.this().Literal,
		}

		p.advance("parseParameters loop identifier") // consume identifier

		if p.this().Type == tokens.Question {
			param.Optional = true
			haveOptional = true

			p.advance("parseParameters loop ?") // consume ?
		} else if haveOptional {
			// This parameter is not optional, but a previous parameter was, this is not allowed.
			p.error(p.prev(), "all input parameters following an optional parameter must also be optional", "parseParameters")
			return nil
		}

		if p.this().Type != tokens.Colon {
			p.error(p.this(), "expected ':' after input parameter identifier", "parseParameters")
			return nil
		}

		p.advance("parseParameters loop :") // consume :

		paramType := p.parseCombinedType(ctx, false, false)
		if paramType == nil {
			p.error(p.this(), "unknown parameter type", "parseParameters")
			return nil
		}

		param.Type = paramType

		if p.this().Type == tokens.Assign {
			if !param.Optional {
				p.error(p.this(), "default values are only allowed for optional input parameters", "parseParameters")
				return nil
			}

			// Default parameter value assignment
			p.advance("parseParameters loop =") // consume '='

			expr := p.expression(ctx, paramType)
			if expr != nil {
				param.Default = expr
			}
		}

		procType.Parameters = append(procType.Parameters, param)

		if p.this().Type == tokens.Comma {
			p.advance("parseParameters loop ,") // consume ','
		}
	}

	p.advance("parseProcedureType )") // consume )

	if p.this().Type == tokens.Assign {
		// No return type.
		return procType
	}

	// TODO: this should only allow a limited set of types.
	returnType := p.parseCombinedType(ctx, exported, global)
	if returnType == nil {
		return nil
	}

	// Result type: T ! E
	if p.this().Type == tokens.Not {
		p.advance("parseProcedureType !") // consume !

		errorType := p.parseCombinedType(ctx, exported, global)
		if errorType == nil {
			return nil
		}

		if errorType.Kind() != types.ErrorKind {
			p.error(p.prev(), "result error type must be an error type", "parseProcedureType")
			return nil
		}

		if returnType.Kind() == types.ErrorKind {
			p.error(p.prev(), "result value type cannot be an error type", "parseProcedureType")
			return nil
		}

		returnType = &types.Result{
			Value: returnType,
			Error: errorType,
		}
	}

	procType.ReturnType = returnType

	return procType
}

func (p *Parser) parseEnumType(ctx context.Context, ident *ast.Identifier) types.Type {
	p.advance("parseEnumType enum") // consume enum

	if p.this().Type != tokens.LT {
		p.error(p.this(), "expected < after enum type", "parseEnumType")
		return nil
	}

	p.advance("parseEnumType <") // consume <

	valType := p.parseType(ctx)
	if valType == nil {
		return nil
	}

	if p.this().Type != tokens.GT {
		p.error(p.this(), "expected > after enum value type", "parseEnumType")
		return nil
	}

	p.advance("parseEnumType >") // consume >

	if p.this().Type != tokens.LBrace {
		p.error(p.this(), "expected { after enum type", "parseEnumType")
		return nil
	}

	p.advance("parseEnumType {") // consume {

	typ := &types.Enum{
		ValueType: valType,
		Values:    make([]*types.EnumValue, 0),
	}

	for !p.match(tokens.RBrace, tokens.EOF) {
		if ctx.Err() != nil {
			return nil
		}

		if p.this().Type != tokens.Identifier {
			p.error(p.this(), "expected identifier in enum declaration", "parseEnumType")
			return nil
		}

		valIdent := &ast.Identifier{
			Token:     p.this(),
			Name:      p.this().Literal,
			ValueType: valType,
			Exported:  ident.Exported,
		}

		p.symbols.DefineEnumValue(ident.Name, valIdent)

		p.advance("parseEnumType identifier") // consume identifier

		if p.this().Type != tokens.Declaration {
			p.error(p.this(), "expected := in enum literal", "parseEnumType")
			return nil
		}

		p.advance("parseEnumType :=") // consume :=

		enumExpr := p.expression(ctx, valType)
		if enumExpr != nil {
			typ.Values = append(typ.Values, &types.EnumValue{
				Name:  valIdent.Name,
				Value: enumExpr,
			})
		}

		if p.this().Type == tokens.Comma {
			p.advance("parseEnumType ,") // consume ,
		}
	}

	p.advance("parseEnumType }") // consume }

	return typ
}

func (p *Parser) parseErrorType(ctx context.Context, ident *ast.Identifier) types.Type {
	p.advance("parseErrorType error") // consume error

	typ := &types.Error{
		Values: make([]*types.EnumValue, 0),
	}

	if p.this().Type == tokens.LT {
		// Typed error: error<ascii> or error<utf8>
		p.advance("parseErrorType <") // consume <

		valType := p.parseType(ctx)
		if valType == nil {
			return nil
		}

		if valType.Kind() != types.ASCII && valType.Kind() != types.UTF8 {
			p.error(p.this(), "error type parameter must be ascii or utf8", "parseErrorType")
			return nil
		}

		typ.ValueType = valType

		if p.this().Type != tokens.GT {
			p.error(p.this(), "expected > after error value type", "parseErrorType")
			return nil
		}

		p.advance("parseErrorType >") // consume >
	}

	if p.this().Type != tokens.LBrace {
		p.error(p.this(), "expected { after error type", "parseErrorType")
		return nil
	}

	p.advance("parseErrorType {") // consume {

	for !p.match(tokens.RBrace, tokens.EOF) {
		if ctx.Err() != nil {
			return nil
		}

		if p.this().Type != tokens.Identifier {
			p.error(p.this(), "expected identifier in error declaration", "parseErrorType")
			return nil
		}

		valName := p.this().Literal

		// For typeless errors, the value type is utf8 (printed as the variant name).
		valType := typ.ValueType
		if valType == nil {
			valType = types.Basics[types.UTF8]
		}

		valIdent := &ast.Identifier{
			Token:     p.this(),
			Name:      valName,
			ValueType: valType,
			Exported:  ident.Exported,
		}

		p.symbols.DefineEnumValue(ident.Name, valIdent)

		p.advance("parseErrorType identifier") // consume identifier

		if typ.ValueType != nil {
			// Typed error: require := value
			if p.this().Type != tokens.Declaration {
				p.error(p.this(), "expected := in typed error literal", "parseErrorType")
				return nil
			}

			p.advance("parseErrorType :=") // consume :=

			enumExpr := p.expression(ctx, typ.ValueType)
			if enumExpr != nil {
				typ.Values = append(typ.Values, &types.EnumValue{
					Name:  valName,
					Value: enumExpr,
				})
			}
		} else {
			// Typeless error: value is the variant name as a string literal.
			typ.Values = append(typ.Values, &types.EnumValue{
				Name: valName,
				Value: &ast.UTF8Literal{
					Token: tokens.Token{Type: tokens.StringLiteral, Literal: valName},
					Value: valName,
				},
			})
		}

		if p.this().Type == tokens.Comma {
			p.advance("parseErrorType ,") // consume ,
		}
	}

	p.advance("parseErrorType }") // consume }

	return typ
}
