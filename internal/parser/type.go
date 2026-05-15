package parser

import (
	"context"
	"fmt"

	"github.com/samborkent/cog/internal/ast"
	"github.com/samborkent/cog/internal/tokens"
	"github.com/samborkent/cog/internal/types"
)

func (p *Parser) parseCombinedType(ctx context.Context, exported, global bool) types.Type {
	switch p.lex.This().Type {
	case tokens.Enum:
		ident := &ast.Identifier{
			Token:    p.lex.Peek(-2),
			Exported: exported,
			Global:   global,
		}

		return p.parseEnumType(ctx, ident)
	case tokens.Error:
		ident := &ast.Identifier{
			Token:    p.lex.Peek(-2),
			Exported: exported,
			Global:   global,
		}

		return p.parseErrorType(ctx, ident)
	case tokens.Function, tokens.Procedure:
		return p.parseProcedureType(ctx, exported, global)
	}

	typ := p.parseType(ctx)

	// Propagate exported/global flags to tuple types parsed by parseType.
	if t, ok := typ.(*types.Tuple); ok {
		t.Exported = exported
		t.Global = global
	}

	switch p.lex.This().Type {
	case tokens.BitXor:
		// Either (concrete two-type union with Left/Right)
		p.lex.Step() // consume ^

		right := p.parseType(ctx)
		if right == nil {
			return nil
		}

		return &types.Either{
			Left:     typ,
			Right:    right,
			Exported: exported,
			Global:   global,
		}
	case tokens.Not:
		// Result type: T ! E
		p.lex.Step() // consume !

		errorType := p.parseCombinedType(ctx, exported, global)
		if errorType == nil {
			return nil
		}

		if errorType.Kind() != types.ErrorKind {
			p.error(p.lex.Peek(-1), "result error type must be an error type", "parseCombinedType")
			return nil
		}

		if typ.Kind() == types.ErrorKind {
			p.error(p.lex.Peek(-1), "result value type cannot be an error type", "parseCombinedType")
			return nil
		}

		return &types.Result{
			Value: typ,
			Error: errorType,
		}
	}

	return typ
}

// canStartType reports whether the current token can begin a type expression.
func (p *Parser) canStartType() bool {
	switch p.lex.This().Type {
	case tokens.Interface, tokens.LBracket, tokens.LParen, tokens.Map, tokens.Set,
		tokens.Struct, tokens.BitAnd, tokens.Function, tokens.Procedure:
		return true
	case tokens.Identifier:
		return true
	}
	// Check for built-in type keywords (int64, utf8, etc.).
	_, ok := types.Lookup[p.lex.This().Type]

	return ok
}

func (p *Parser) parseType(ctx context.Context) types.Type {
	switch p.lex.This().Type {
	case tokens.Interface:
		return p.parseInterface(ctx)
	case tokens.LParen:
		// Tuple type: (T1, T2, ...)
		p.lex.Step() // consume (

		first := p.parseType(ctx)
		if first == nil {
			return nil
		}

		if p.lex.This().Type != tokens.Comma {
			p.error(p.lex.This(), "expected ',' after first tuple element type", "parseType")
			return nil
		}

		tuple := &types.Tuple{
			Types: make([]types.Type, 1, types.TupleMaxTypes),
		}

		tuple.Types[0] = first

		for p.lex.This().Type != tokens.EOF && ctx.Err() == nil {
			if p.lex.This().Type != tokens.Comma {
				break
			}

			p.lex.Step() // consume ,

			next := p.parseType(ctx)
			if next != nil {
				tuple.Types = append(tuple.Types, next)
			}
		}

		if p.lex.This().Type != tokens.RParen {
			p.error(p.lex.This(), "expected ')' to close tuple type", "parseType")
			return nil
		}

		p.lex.Step() // consume )

		return tuple
	case tokens.LBracket:
		p.lex.Step() // consume [

		if p.lex.This().Type == tokens.RBracket {
			// Slice type
			if p.lex.This().Type != tokens.RBracket {
				p.error(p.lex.This(), "expected closing ] in slice type", "parseType")
				return nil
			}

			p.lex.Step() // consume ]

			elemType := p.parseType(ctx)
			if elemType == nil {
				return nil
			}

			return &types.Slice{
				Element: elemType,
			}
		}

		// Array type
		switch p.lex.This().Type {
		case tokens.IntLiteral:
		case tokens.Identifier:
			symbol, ok := p.symbols.Resolve(p.lex.This().Literal)
			if ok && types.IsFixed(symbol.Identifier.ValueType) {
				break
			}

			fallthrough
		default:
			p.error(p.lex.This(), "expected fixed-point number type as array length", "parseCombinedType")
			return nil
		}

		lenExpr := p.expression(ctx, types.None)
		if lenExpr == ast.ZeroExprIndex {
			return nil
		}

		if p.lex.This().Type != tokens.RBracket {
			p.error(p.lex.This(), "expected closing ] in array type", "parseType")
			return nil
		}

		p.lex.Step() // consume ]

		elemType := p.parseType(ctx)
		if elemType == nil {
			return nil
		}

		return &types.Array{
			Element: elemType,
			Length:  p.typeExpr(lenExpr),
		}
	case tokens.Map:
		p.lex.Step() // consume map

		if p.lex.This().Type != tokens.LT {
			p.error(p.lex.This(), "expected < after map type", "parseType")
			return nil
		}

		p.lex.Step() // consume <

		keyType := p.parseType(ctx)
		if keyType == nil {
			return nil
		}

		if p.lex.This().Type != tokens.Comma {
			p.error(p.lex.This(), "expected , after map key type", "parseType")
			return nil
		}

		p.lex.Step() // consume ,

		valType := p.parseType(ctx)
		if valType == nil {
			return nil
		}

		if p.lex.This().Type != tokens.GT {
			p.error(p.lex.This(), "expected > after map value type", "parseType")
			return nil
		}

		p.lex.Step() // consume >

		return &types.Map{
			Key:   keyType,
			Value: valType,
		}
	case tokens.Set:
		p.lex.Step() // consume set

		if p.lex.This().Type != tokens.LT {
			p.error(p.lex.This(), "expected < after set type", "parseType")
			return nil
		}

		p.lex.Step() // consume <

		elemType := p.parseType(ctx)
		if elemType == nil {
			return nil
		}

		if p.lex.This().Type != tokens.GT {
			p.error(p.lex.This(), "expected > after set element type", "parseType")
			return nil
		}

		p.lex.Step() // consume >

		return &types.Set{Element: elemType}
	case tokens.Struct:
		return p.parseStruct(ctx)
	case tokens.BitAnd:
		// Reference type parsing
		p.lex.Step() // consume &

		valType := p.parseType(ctx)
		if valType == nil {
			return nil
		}

		// TODO: check if this is correct.
		if types.IsPointer(valType) {
			p.error(p.lex.This(), fmt.Sprintf("reference of pointer type %q not allowed", valType.Kind()), "parseType")
			return nil
		}

		return &types.Reference{
			Value: valType,
		}
	}

	typ, ok := types.Lookup[p.lex.This().Type]
	if !ok {
		// Reject constraint-only keywords in type position.
		if _, isConstraint := types.ConstraintLookup[p.lex.This().Type]; isConstraint {
			p.error(p.lex.This(), fmt.Sprintf("%q is a constraint, not a concrete type", p.lex.This().Type), "parseType")
			return nil
		}

		// Check for imported package type: pkg.Type
		if p.lex.This().Type == tokens.Identifier && p.lex.Peek(1).Type == tokens.Dot {
			if imp, isImport := p.symbols.ResolveCogImport(p.lex.This().Literal); isImport {
				p.lex.Step() // consume package name
				p.lex.Step() // consume '.'

				if p.lex.This().Type != tokens.Identifier {
					p.error(p.lex.This(), "expected type name after package selector", "parseType")
					return nil
				}

				sym, found := imp.Exports[p.lex.This().Literal]
				if !found || sym.Identifier.Qualifier != ast.QualifierType {
					p.error(p.lex.This(), fmt.Sprintf("package %q has no exported type %q", imp.Name, p.lex.This().Literal), "parseType")
					return nil
				}

				ident := sym.Identifier
				if types.IsNone(ident.ValueType) {
					typ = types.NewForwardAlias(ident.Token.Literal, ident.Exported, ident.Global, func() types.Type {
						return ident.ValueType
					})
				} else {
					typ = &types.Alias{
						Name:     ident.Token.Literal,
						Derived:  ident.ValueType,
						Exported: ident.Exported,
						Global:   ident.Global,
					}
				}

				p.lex.Step() // consume type name

				if p.lex.This().Type == tokens.Question {
					p.lex.Step() // consume ?

					if typ.Kind() == types.OptionKind {
						p.error(p.lex.This(), "nested optional types are not allowed", "parseType")
						return nil
					}

					return &types.Option{Value: typ}
				}

				return typ
			}
		}

		// Non-basic type, try to find in symbol table.
		typeSymbol, ok := p.symbols.Resolve(p.lex.This().Literal)
		if !ok && p.globalsPass && p.lex.This().Type == tokens.Identifier {
			// Forward type reference. Register stub, resolve lazily.
			ident := &ast.Identifier{
				Token:     p.lex.This(),
				Qualifier: ast.QualifierType,
				Global:    true,
			}

			p.symbols.DefineGlobal(ident) // registers with ScanScope + ValueType=None
			p.lex.Step()

			return types.NewForwardAlias(ident.Token.Literal, ident.Exported, ident.Global, func() types.Type {
				return ident.ValueType
			})
		}

		if !ok || typeSymbol.Identifier.Qualifier != ast.QualifierType {
			p.error(p.lex.This(), "unknown type found in type declaration", "parseType")
			return nil
		}

		ident := typeSymbol.Identifier

		// If the symbol is a type parameter (inside a generic alias body),
		// return the type parameter alias directly.
		if alias, ok := ident.ValueType.(*types.Alias); ok && alias.IsTypeParam() {
			p.lex.Step() // consume type param name
			return alias
		}

		if types.IsNone(ident.ValueType) {
			// Forward reference: type name is pre-registered but not yet resolved.
			// Create a lazy alias that resolves when the type is accessed.
			typ = types.NewForwardAlias(ident.Token.Literal, ident.Exported, ident.Global, func() types.Type {
				return ident.ValueType
			})
		} else {
			// Copy type parameters from the original type if it's an alias
			var typeParams []*types.Alias

			if originalAlias, ok := ident.ValueType.(*types.Alias); ok {
				typeParams = originalAlias.TypeParams
			}

			typ = &types.Alias{
				Name:       ident.Token.Literal,
				Derived:    ident.ValueType,
				Exported:   ident.Exported,
				Global:     ident.Global,
				TypeParams: typeParams,
			}
		}
	}

	p.lex.Step() // consume type

	// Check for generic instantiation: Alias<int32, utf8>
	if p.lex.This().Type == tokens.LT {
		typ = p.instantiateGenericAlias(ctx, typ)
		if typ == nil {
			return nil
		}
	}

	if p.lex.This().Type == tokens.Question {
		// Optional type
		p.lex.Step() // consume ?

		if typ.Kind() == types.OptionKind {
			p.error(p.lex.This(), "nested optional types are not allowed", "parseType")
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
		p.error(p.lex.This(), "type arguments on non-alias type", "instantiateGenericAlias")
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
		p.error(p.lex.This(), fmt.Sprintf("type %q is not generic", alias.Name), "instantiateGenericAlias")
		return nil
	}

	typeArgs := p.parseTypeArguments(ctx)
	if typeArgs == nil {
		return nil
	}

	if len(typeArgs) != len(genAlias.TypeParams) {
		p.error(p.lex.This(), fmt.Sprintf("wrong number of type arguments for %q: expected %d, got %d",
			alias.Name, len(genAlias.TypeParams), len(typeArgs)), "instantiateGenericAlias")

		return nil
	}

	// Check constraint satisfaction.
	for i, arg := range typeArgs {
		tp := genAlias.TypeParams[i]
		if !tp.SatisfiedBy(arg) {
			p.error(p.lex.This(), fmt.Sprintf("type argument %q does not satisfy constraint %q for parameter %q",
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

func (p *Parser) parseInterface(ctx context.Context) types.Type {
	p.lex.Step() // consume interface

	if p.lex.This().Type != tokens.LBrace {
		p.error(p.lex.This(), "expected { after interface declaration", "parseInterface")
		return nil
	}

	p.lex.Step() // consume {

	methods := []*types.Method{}

	for p.lex.This().Type != tokens.EOF && ctx.Err() == nil {
		tok := p.lex.This()
		if tok.Type == tokens.RBrace {
			break
		}

		if tok.Type != tokens.Identifier {
			p.error(tok, "unexpected token found in interface declaration", "parseInterface")
			return nil
		}

		method := &types.Method{
			Name: tok.Literal,
		}

		p.lex.Step() // consume identifier

		if p.lex.This().Type != tokens.Colon {
			p.error(p.lex.This(), "expected : after method name in interface method", "parseInterface")
			return nil
		}

		p.lex.Step() // consume :

		methodType := p.parseProcedureType(ctx, true, false)
		if methodType == nil {
			return nil
		}

		method.Procedure = methodType

		methods = append(methods, method)
	}

	p.lex.Step() // consume }

	return &types.Interface{
		Methods: methods,
	}
}

func (p *Parser) parseStruct(ctx context.Context) types.Type {
	p.lex.Step() // consume struct

	if p.lex.This().Type != tokens.LBrace {
		p.error(p.lex.This(), "expected { after struct declaration", "parseStruct")
		return nil
	}

	p.lex.Step() // consume {

	fields := []*types.Field{}

	isComplex := false

	for p.lex.This().Type != tokens.EOF && ctx.Err() == nil {
		tok := p.lex.This()
		if tok.Type == tokens.RBrace {
			break
		}

		switch tok.Type {
		case tokens.Export:
			p.lex.Step() // consume export

			if p.lex.This().Type == tokens.LParen {
				p.lex.Step() // consume (

				for p.lex.This().Type != tokens.EOF && ctx.Err() == nil {
					if p.lex.This().Type == tokens.RParen {
						break
					}

					field := p.parseField(ctx, true)
					if field == nil {
						return nil
					}

					if field.PointerLike {
						isComplex = true
					}

					fields = append(fields, field)
				}

				p.lex.Step() // consume )

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
			p.error(p.lex.This(), "unexpected token found in struct declaration", "parseStruct")
			return nil
		}
	}

	p.lex.Step() // consume }

	return &types.Struct{
		Fields:    fields,
		IsComplex: isComplex,
	}
}

func (p *Parser) parseField(ctx context.Context, exported bool) *types.Field {
	field := &types.Field{
		Name:     p.lex.This().Literal,
		Exported: exported,
	}

	p.lex.Step() // consume identifier

	if p.lex.This().Type != tokens.Colon {
		p.error(p.lex.This(), "expected : after field name in struct declaration", "parseStruct")
		return nil
	}

	p.lex.Step() // consume :

	fieldType := p.parseCombinedType(ctx, exported, false)
	if fieldType == nil {
		return nil
	}

	field.Type = fieldType

	if field.Type.Kind() == types.StructKind {
		field.PointerLike = field.Type.Underlying().(*types.Struct).IsComplex
	} else {
		field.PointerLike = types.IsPointer(field.Type)
	}

	return field
}

func (p *Parser) parseProcedureType(ctx context.Context, exported, global bool) *types.Procedure {
	procType := &types.Procedure{
		Function:   p.lex.This().Type == tokens.Function,
		Parameters: make([]*types.Parameter, 0),
	}

	p.lex.Step() // consume proc/func

	if p.lex.This().Type == tokens.LT {
		procType.TypeParams = p.parseTypeParams(ctx)
		if procType.TypeParams == nil {
			return nil
		}

		// Enter scope for type parameters.
		p.symbols = NewEnclosedSymbolTable(p.symbols)

		// Pre-register type parameters in symbol table for recursive references.
		for _, tp := range procType.TypeParams {
			p.symbols.Define(&ast.Identifier{
				Token: tokens.Token{
					Type:    tokens.Identifier,
					Literal: tp.Name,
				},
				ValueType: tp,
				Qualifier: ast.QualifierType,
			})
		}

		defer func() {
			// Exit type parameter scope.
			p.symbols = p.symbols.Outer
		}()
	}

	if p.lex.This().Type != tokens.LParen {
		p.error(p.lex.This(), fmt.Sprintf("expected '(' after %q in type", p.lex.Peek(-1).Type), "parseProcedureType")
		return nil
	}

	p.lex.Step() // consume (

	// Flag to keep track of if any of the parameters is optional.
	// When a parameter is marked as optional, all following parameters must also be optional.
	haveOptional := false

	for p.lex.This().Type != tokens.EOF && ctx.Err() == nil {
		tok := p.lex.This()
		if tok.Type == tokens.RParen {
			break
		}

		if tok.Type != tokens.Identifier {
			p.error(tok, "expected parameter identifier", "parseParameters")
			return nil
		}

		param := &types.Parameter{
			Name: tok.Literal,
		}

		p.lex.Step() // consume identifier

		if p.lex.This().Type == tokens.Question {
			param.Optional = true
			haveOptional = true

			p.lex.Step() // consume ?
		} else if haveOptional {
			// This parameter is not optional, but a previous parameter was, this is not allowed.
			p.error(p.lex.Peek(-1), "all input parameters following an optional parameter must also be optional", "parseParameters")
			return nil
		}

		if p.lex.This().Type != tokens.Colon {
			p.error(p.lex.This(), "expected ':' after input parameter identifier", "parseParameters")
			return nil
		}

		p.lex.Step() // consume :

		paramType := p.parseCombinedType(ctx, false, false)
		if paramType == nil {
			p.error(p.lex.This(), "unknown parameter type", "parseParameters")
			return nil
		}

		param.Type = paramType

		if p.lex.This().Type == tokens.Assign {
			if !param.Optional {
				p.error(p.lex.This(), "default values are only allowed for optional input parameters", "parseParameters")
				return nil
			}

			// Default parameter value assignment
			p.lex.Step() // consume '='

			expr := p.expression(ctx, paramType)
			if expr != ast.ZeroExprIndex {
				param.Default = new(p.typeExpr(expr))
			}
		}

		procType.Parameters = append(procType.Parameters, param)

		if p.lex.This().Type == tokens.Comma {
			p.lex.Step() // consume ','
		}
	}

	p.lex.Step() // consume )

	if p.lex.This().Type == tokens.Assign {
		// No return type.
		return procType
	}

	// Only attempt to parse a return type when the current token can
	// actually begin a type expression. Without this check, contexts where
	// a procedure type has no return type and no '=' (interface methods,
	// struct proc fields, etc.) would incorrectly try to parse the next
	// token (e.g. '}') as a return type.
	//
	// This mirrors the Go specification grammar where Result is optional:
	//   Signature = Parameters [ Result ]
	if !p.canStartType() {
		return procType
	}

	// TODO: this should only allow a limited set of types.
	returnType := p.parseCombinedType(ctx, exported, global)
	if returnType == nil {
		return nil
	}

	// Result type: T ! E
	if p.lex.This().Type == tokens.Not {
		p.lex.Step() // consume !

		errorType := p.parseCombinedType(ctx, exported, global)
		if errorType == nil {
			return nil
		}

		if errorType.Kind() != types.ErrorKind {
			p.error(p.lex.Peek(-1), "result error type must be an error type", "parseProcedureType")
			return nil
		}

		if returnType.Kind() == types.ErrorKind {
			p.error(p.lex.Peek(-1), "result value type cannot be an error type", "parseProcedureType")
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
	p.lex.Step() // consume enum

	if p.lex.This().Type != tokens.LT {
		p.error(p.lex.This(), "expected < after enum type", "parseEnumType")
		return nil
	}

	p.lex.Step() // consume <

	valType := p.parseType(ctx)
	if valType == nil {
		return nil
	}

	if p.lex.This().Type != tokens.GT {
		p.error(p.lex.This(), "expected > after enum value type", "parseEnumType")
		return nil
	}

	p.lex.Step() // consume >

	if p.lex.This().Type != tokens.LBrace {
		p.error(p.lex.This(), "expected { after enum type", "parseEnumType")
		return nil
	}

	p.lex.Step() // consume {

	typ := &types.Enum{
		ValueType: valType,
		Values:    make([]*types.EnumValue, 0),
	}

	for p.lex.This().Type != tokens.EOF && ctx.Err() == nil {
		tok := p.lex.This()
		if tok.Type == tokens.RBrace {
			break
		}

		if tok.Type != tokens.Identifier {
			p.error(tok, "expected identifier in enum declaration", "parseEnumType")
			return nil
		}

		valIdent := &ast.Identifier{
			Token:     tok,
			ValueType: valType,
			Exported:  ident.Exported,
		}

		p.symbols.DefineEnumValue(ident.Token.Literal, valIdent)

		p.lex.Step() // consume identifier

		if p.lex.This().Type != tokens.Declaration {
			p.error(p.lex.This(), "expected := in enum literal", "parseEnumType")
			return nil
		}

		p.lex.Step() // consume :=

		enumExpr := p.expression(ctx, valType)
		if enumExpr != ast.ZeroExprIndex {
			typ.Values = append(typ.Values, &types.EnumValue{
				Name:  valIdent.Token.Literal,
				Value: p.typeExpr(enumExpr),
			})
		}

		if p.lex.This().Type == tokens.Comma {
			p.lex.Step() // consume ,
		}
	}

	p.lex.Step() // consume }

	return typ
}

func (p *Parser) parseErrorType(ctx context.Context, ident *ast.Identifier) types.Type {
	p.lex.Step() // consume error

	typ := &types.Error{
		Values: make([]*types.EnumValue, 0),
	}

	if p.lex.This().Type == tokens.LT {
		// Typed error: error<ascii> or error<utf8>
		p.lex.Step() // consume <

		valType := p.parseType(ctx)
		if valType == nil {
			return nil
		}

		if valType.Kind() != types.ASCII && valType.Kind() != types.UTF8 {
			p.error(p.lex.This(), "error type parameter must be ascii or utf8", "parseErrorType")
			return nil
		}

		typ.ValueType = valType

		if p.lex.This().Type != tokens.GT {
			p.error(p.lex.This(), "expected > after error value type", "parseErrorType")
			return nil
		}

		p.lex.Step() // consume >
	}

	if p.lex.This().Type != tokens.LBrace {
		p.error(p.lex.This(), "expected { after error type", "parseErrorType")
		return nil
	}

	p.lex.Step() // consume {

	for p.lex.This().Type != tokens.EOF && ctx.Err() == nil {
		tok := p.lex.This()
		if tok.Type == tokens.RBrace {
			break
		}

		if tok.Type != tokens.Identifier {
			p.error(tok, "expected identifier in error declaration", "parseErrorType")
			return nil
		}

		valName := tok.Literal

		// For typeless errors, the value type is utf8 (printed as the variant name).
		valType := typ.ValueType
		if valType == nil {
			valType = types.Basics[types.UTF8]
		}

		valIdent := &ast.Identifier{
			Token:     tok,
			ValueType: valType,
			Exported:  ident.Exported,
		}

		p.symbols.DefineEnumValue(ident.Token.Literal, valIdent)

		p.lex.Step() // consume identifier

		if typ.ValueType != nil {
			// Typed error: require := value
			if p.lex.This().Type != tokens.Declaration {
				p.error(p.lex.This(), "expected := in typed error literal", "parseErrorType")
				return nil
			}

			p.lex.Step() // consume :=

			enumExpr := p.expression(ctx, typ.ValueType)
			if enumExpr != ast.ZeroExprIndex {
				typ.Values = append(typ.Values, &types.EnumValue{
					Name:  valName,
					Value: p.typeExpr(enumExpr),
				})
			}
		} else {
			literal := &ast.UTF8Literal{
				Token: tokens.Token{Type: tokens.StringLiteral, Literal: valName},
				Value: valName,
			}

			// Typeless error: value is the variant name as a string literal.
			typ.Values = append(typ.Values, &types.EnumValue{
				Name: valName,
				Value: types.Expression{
					Expr:   literal,
					String: literal.String(),
				},
			})
		}

		if p.lex.This().Type == tokens.Comma {
			p.lex.Step() // consume ,
		}
	}

	p.lex.Step() // consume }

	return typ
}
