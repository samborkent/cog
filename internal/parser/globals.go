package parser

import (
	"context"

	"github.com/samborkent/cog/internal/ast"
	"github.com/samborkent/cog/internal/tokens"
	"github.com/samborkent/cog/internal/types"
)

// isGenericTypeDecl checks whether the current position is at the start of a
// generic type declaration (e.g. List<T ~ any> ~ ...). It expects the cursor
// on an Identifier and looks ahead for the pattern: < Identifier ~
// This distinguishes generic type aliases from comparison expressions like
// index < 5.
func (p *Parser) isGenericTypeDecl() bool {
	// Current token is an Identifier. Check: next is <, next+1 is Identifier, next+2 is ~.
	if p.next().Type != tokens.LT {
		return false
	}

	// Peek at i+2 and i+3.
	if p.i+3 >= len(p.tokens) {
		return false
	}

	return p.tokens[p.i+2].Type == tokens.Identifier && p.tokens[p.i+3].Type == tokens.Tilde
}

// FindGlobals scans the token stream to pre-register all top-level names
// (types, declarations, enums) into the parser's symbol table. It can be
// called externally when multiple parsers share one symbol table, so that
// all files' globals are visible before any file is fully parsed.
func (p *Parser) FindGlobals(ctx context.Context) {
	if p.scriptMode {
		// Script files have definition scope (no forward references).
		// Only scan for import statements so that imported packages can be
		// compiled before the file is parsed.
		p.findScriptImports(ctx)
		return
	}

	// Pre-register all type names so forward references can be resolved.
	p.preRegisterTypeNames(ctx)

tokenLoop:
	for p.this().Type != tokens.EOF {
		exported := false

		prev := p.i

		if p.this().Type == tokens.Export {
			if p.scriptMode {
				p.advance("findGlobals export script") // skip export in script mode
				continue
			}

			p.advance("findGlobals export") // consume export

			exported = true
		}

		qualifier := ast.QualifierImmutable

		switch p.this().Type {
		case tokens.BitAnd:
			// Reference receiver method.
			p.advance("findGlobals &") // consume &
		case tokens.Dynamic:
			qualifier = ast.QualifierDynamic

			p.advance("findGlobals dyn") // consume dyn
		case tokens.Variable:
			qualifier = ast.QualifierVariable

			p.advance("findGlobals var") // consume var
		}

		switch p.this().Type {
		case tokens.GoImport:
			p.advance("findGlobals goimport") // consume goimport

			if p.this().Type == tokens.LParen {
				p.skipGrouped(ctx)
			}
		case tokens.Import:
			p.parseImport() // process imports during global scan
		case tokens.Identifier:
			switch p.next().Type {
			case tokens.Colon, tokens.Declaration:
				p.findGlobalDecl(ctx, exported, qualifier)
			case tokens.Dot:
				p.findGlobalMethod(ctx, exported)
			case tokens.Tilde:
				p.findGlobalType(ctx, exported)
			case tokens.LT:
				if p.isGenericTypeDecl() {
					p.findGlobalType(ctx, exported)
				} else {
					p.advance("findGlobals") // not a type decl, skip
				}
			default:
				p.advance("findGlobals") // consume token
			}
		case tokens.Package:
			p.advance("findGlobals package") // consume package

			if p.this().Type == tokens.Identifier {
				p.advance("findGlobals package identifier") // consume package identifier
			}
		case tokens.EOF:
			break tokenLoop
		default:
			p.advance("findGlobals") // consume token
		}

		// Guard against infinite loops: if no progress was made, force advance.
		if p.i == prev {
			p.advance("findGlobals recovery")
		}
	}

	p.i = 0
	p.Errs = p.Errs[:0]
}

func (p *Parser) findScriptImports(ctx context.Context) {
	// Pre-register type names so that type aliases can be resolved during parsing.
	p.preRegisterTypeNames(ctx)

	for p.this().Type != tokens.EOF {
		if ctx.Err() != nil {
			return
		}

		switch p.this().Type {
		case tokens.Import:
			p.parseImport()
		case tokens.GoImport:
			p.advance("findScriptImports goimport") // consume goimport

			if p.this().Type == tokens.LParen {
				p.skipGrouped(ctx)
			}
		default:
			p.advance("findScriptImports")
		}
	}

	p.i = 0
	p.Errs = p.Errs[:0]
}

func (p *Parser) preRegisterTypeNames(ctx context.Context) {
	for p.this().Type != tokens.EOF {
		if ctx.Err() != nil {
			return
		}

		exported := false

		if p.this().Type == tokens.Export {
			if p.scriptMode {
				p.advance("preRegister export script") // skip export in script mode
				continue
			}

			p.advance("preRegister export") // consume export

			exported = true
		}

		// Skip qualifiers (types can't have dyn/var but scan past them)
		if p.this().Type == tokens.Dynamic || p.this().Type == tokens.Variable {
			p.advance("preRegister qualifier") // consume qualifier
		}

		if p.this().Type == tokens.Identifier &&
			(p.next().Type == tokens.Tilde || p.isGenericTypeDecl()) {
			ident := &ast.Identifier{
				Token:    p.this(),
				Name:     p.this().Literal,
				Exported: exported,
				// Qualifier defaults to QualifierType (zero value)
				Global: true,
			}

			p.symbols.DefineGlobal(ident)

			p.advance("preRegister identifier") // consume token

			// Skip type parameters in procedure/function declarations during pre-registration.
			if p.this().Type == tokens.LT {
				p.skipTypeParams(ctx)
			}

			continue
		}

		// Skip type parameters in procedure/function declarations during pre-registration.
		if (p.this().Type == tokens.Procedure || p.this().Type == tokens.Function) &&
			p.next().Type == tokens.LT {
			p.advance("preRegister proc") // consume token
			p.skipTypeParams(ctx)

			continue
		}

		p.advance("preRegister") // consume token
	}

	p.i = 0
	p.Errs = p.Errs[:0]
}

func (p *Parser) findGlobalDecl(ctx context.Context, exported bool, qualifier ast.Qualifier) {
	if p.this().Type != tokens.Identifier {
		return
	}

	_, ok := p.symbols.Resolve(p.this().Literal)
	if ok {
		// Report redeclare error and advance past the identifier to avoid an infinite loop
		p.error(p.this(), "cannot redeclare variable", "findGlobalDecl")
		p.advance("findGlobalDecl redeclare") // consume identifier to make progress

		return
	}

	ident := &ast.Identifier{
		Token:     p.this(),
		Name:      p.this().Literal,
		Exported:  exported,
		Qualifier: qualifier,
		Global:    true,
	}

	p.advance("findGlobalDecl identifier") // consume identifier

	switch p.this().Type {
	case tokens.Colon:
		p.advance("findGlobalDecl :") // consume :

		ident.ValueType = p.parseCombinedType(ctx, exported, true)

		if ident.Name == "main" {
			procType, isProc := ident.ValueType.(*types.Procedure)
			if !isProc || procType.Function || len(procType.Parameters) != 0 || procType.ReturnType != nil {
				p.error(ident.Token, `"main" can only be declared as proc()`, "findGlobalDecl")

				// Skip past the function body to avoid stalling.
				if p.this().Type == tokens.Assign {
					p.advance("findGlobalDecl skip =") // consume =

					if p.this().Type == tokens.LBrace {
						p.skipScope(ctx)
					}
				}

				return
			}
		}

		p.symbols.DefineGlobal(ident)

		if p.this().Type == tokens.Assign {
			p.advance("findGlobalDecl =") // consume =

			if p.this().Type == tokens.LBrace {
				p.skipScope(ctx)
			} else {
				_ = p.expression(ctx, ident.ValueType)
			}
		}
	case tokens.Declaration:
		if ident.Name == "main" {
			p.error(ident.Token, `"main" can only be declared as proc()`, "findGlobalDecl")

			p.advance("findGlobalDecl skip :=") // consume :=

			if p.this().Type == tokens.LBrace {
				p.skipScope(ctx)
			}

			return
		}

		p.advance("findGlobalDecl :=") // consume :=
		p.symbols.DefineGlobal(ident)

		if p.this().Type == tokens.LBrace {
			p.skipScope(ctx)
		} else {
			_ = p.expression(ctx, types.None)
		}
	default:
		return
	}
}

func (p *Parser) findGlobalType(ctx context.Context, exported bool) {
	ident := &ast.Identifier{
		Token:    p.this(),
		Name:     p.this().Literal,
		Exported: exported,
		Global:   true,
	}

	p.advance("findGlobalType identifier") // consume identifier

	preRegistered := false

	existing, ok := p.symbols.Resolve(ident.Name)
	if ok {
		// Allow pre-registered forward-declared types to be resolved.
		if existing.Scope == ScanScope &&
			existing.Identifier.Qualifier == ast.QualifierType &&
			types.IsNone(existing.Identifier.ValueType) {
			preRegistered = true
			ident = existing.Identifier
		} else {
			p.error(p.this(), "cannot redeclare type", "findGlobalType")
			return
		}
	}

	// Parse optional type parameters: <T ~ any, K ~ comparable>
	var typeParams []*types.TypeParam

	if p.this().Type == tokens.LT {
		typeParams = p.parseTypeParams(ctx)
		if typeParams == nil {
			return
		}
	}

	if p.this().Type != tokens.Tilde {
		p.error(p.this(), "expected ~ after type name", "findGlobalType")
		return
	}

	p.advance("findGlobalType ~") // consume ~

	if p.this().Type == tokens.Enum {
		p.advance("findGlobalType enum") // consume enum

		if p.this().Type != tokens.LT {
			p.error(p.this(), "expected < in enum declaration", "findGlobalType")
			return
		}

		p.advance("findGlobalType enum <") // consume <

		enumValType := p.parseCombinedType(ctx, exported, true)

		enumType := &types.Enum{ValueType: enumValType}

		if p.this().Type != tokens.GT {
			p.error(p.this(), "expected > in enum declaration", "findGlobalType")
			return
		}

		p.advance("findGlobalType enum >") // consume >

		if p.this().Type != tokens.LBrace {
			p.error(p.this(), "expected { in enum literal", "findGlobalType")
			return
		}

		p.advance("findGlobalType enum literal {") // consume {

		for p.this().Type != tokens.RBrace && p.this().Type != tokens.EOF {
			if ctx.Err() != nil {
				return
			}

			if p.this().Type != tokens.Identifier {
				p.error(p.this(), "expected identifier in enum literal", "findGlobalType")
				p.advance("findGlobalType enum recovery") // skip bad token

				continue
			}

			valIdent := &ast.Identifier{
				Token:     p.this(),
				Name:      p.this().Literal,
				ValueType: enumValType,
				Exported:  exported,
			}

			p.symbols.DefineEnumValue(ident.Name, valIdent)

			p.advance("findGlobalType enum literal identifier") // consume identifier

			if p.this().Type != tokens.Declaration {
				p.error(p.this(), "expected := in enum literal", "findGlobalType")
				return
			}

			p.advance("findGlobalType enum literal :=") // consume :=

			enumVal := p.expression(ctx, enumValType)
			if enumVal != nil {
				enumType.Values = append(enumType.Values, &types.EnumValue{
					Name:  valIdent.Name,
					Value: enumVal,
				})
			}

			if p.this().Type == tokens.Comma {
				p.advance("findGlobalType enum literal ,") // consume ,
			}
		}

		ident.ValueType = enumType

		if !preRegistered {
			p.symbols.DefineGlobal(ident)
		}

		return
	}

	if p.this().Type == tokens.Error {
		errorType := p.parseErrorType(ctx, ident)
		if errorType == nil {
			return
		}

		ident.ValueType = errorType

		if !preRegistered {
			p.symbols.DefineGlobal(ident)
		}

		return
	}

	// If there are type params, push them into an enclosed scope so that
	// type parameter names (e.g. T) are resolvable in the alias body.
	if len(typeParams) > 0 {
		outer := p.symbols
		p.symbols = NewEnclosedSymbolTable(outer)

		for _, tp := range typeParams {
			p.symbols.Define(&ast.Identifier{
				Name:      tp.Name,
				ValueType: tp,
				Qualifier: ast.QualifierType,
			})
		}

		defer func() { p.symbols = outer }()
	}

	alias := p.parseCombinedType(ctx, ident.Exported, ident.Global)
	if alias == nil {
		return
	}

	// Store type params on the alias.
	if len(typeParams) > 0 {
		if a, ok := alias.(*types.Alias); ok {
			a.TypeParams = typeParams
		} else {
			// Wrap the derived type in an alias to preserve type parameters.
			alias = &types.Alias{
				Name:       ident.Name,
				Derived:    alias,
				Exported:   ident.Exported,
				Global:     ident.Global,
				TypeParams: typeParams,
			}
		}
	}

	ident.ValueType = alias

	if !preRegistered {
		p.symbols.DefineGlobal(ident)
	} else {
		// For pre-registered types, we still need to register struct fields
		// since they weren't available during pre-registration.
		if alias.Kind() == types.StructKind {
			p.symbols.Define(ident)
		}
	}
}

func (p *Parser) skipExpression(ctx context.Context) {
	parenIndex := 0
	braceIndex := 0
	bracketIndex := 0

	for p.this().Type != tokens.EOF {
		if ctx.Err() != nil {
			return
		}

		switch p.this().Type {
		case tokens.LParen:
			parenIndex++
		case tokens.RParen:
			parenIndex--
		case tokens.LBrace:
			braceIndex++
		case tokens.RBrace:
			braceIndex--
		case tokens.LBracket:
			bracketIndex++
		case tokens.RBracket:
			bracketIndex--
		}

		p.advance("skipExpression")

		if parenIndex == 0 && braceIndex == 0 && bracketIndex == 0 {
			return
		}
	}
}

func (p *Parser) skipScope(ctx context.Context) {
	braceIndex := 0

	for p.this().Type != tokens.EOF {
		if ctx.Err() != nil {
			return
		}

		switch p.this().Type {
		case tokens.LBrace:
			braceIndex++
		case tokens.RBrace:
			braceIndex--
		}

		p.advance("skipScope " + p.this().Literal)

		if braceIndex == 0 {
			return
		}
	}
}

func (p *Parser) skipGrouped(ctx context.Context) {
	parenIndex := 0

	for p.this().Type != tokens.EOF {
		if ctx.Err() != nil {
			return
		}

		switch p.this().Type {
		case tokens.LParen:
			parenIndex++
		case tokens.RParen:
			parenIndex--
		}

		p.advance("skipGrouped " + p.this().Literal)

		if parenIndex == 0 {
			return
		}
	}
}

func (p *Parser) findGlobalMethod(ctx context.Context, exported bool) {
	// Parse method declaration: Type.Method : proc() = ...
	// Current token is the receiver type name.
	receiverName := p.this().Literal
	p.advance("findGlobalMethod receiver") // consume receiver type name

	if p.this().Type != tokens.Dot {
		p.error(p.this(), "expected . after receiver type name", "findGlobalMethod")
		return
	}

	p.advance("findGlobalMethod .") // consume .

	if p.this().Type != tokens.Identifier {
		p.error(p.this(), "expected method name after .", "findGlobalMethod")
		return
	}

	methodName := p.this().Literal

	// Create a placeholder method identifier to register in the symbol table.
	methodIdent := &ast.Identifier{
		Token:     p.this(),
		Name:      methodName,
		Exported:  exported,
		Qualifier: ast.QualifierMethod,
		Global:    true,
	}

	p.advance("findGlobalMethod method") // consume identifier

	if p.this().Type != tokens.Colon {
		p.error(p.this(), "expected function type definition after method declaration", "findGlobalMethod")
		return
	}

	p.advance("findGlobalMethod :") // consume :

	procType := p.parseProcedureType(ctx, exported, true)
	if procType == nil {
		return
	}

	methodIdent.ValueType = procType

	if p.this().Type != tokens.Assign {
		p.error(p.this(), "expected function body assignment after method type definition", "findGlobalMethod")
		return
	}

	p.advance("findGlobalMethod =") // consume =

	p.skipScope(ctx)

	// Register the method in the symbol table so it's available for forward references.
	if err := p.symbols.DefineMethod(receiverName, methodIdent); err != nil {
		p.error(p.this(), err.Error(), "findGlobalMethod")
		return
	}
}

func (p *Parser) skipTypeParams(ctx context.Context) {
	parenIndex := 0

	for p.this().Type != tokens.EOF {
		if ctx.Err() != nil {
			return
		}

		switch p.this().Type {
		case tokens.LT:
			parenIndex++
		case tokens.GT:
			parenIndex--
		}

		p.advance("skipTypeParams " + p.this().Literal)

		if parenIndex == 0 {
			return
		}
	}
}
