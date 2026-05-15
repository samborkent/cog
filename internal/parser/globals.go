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
	if p.lex.Peek(1).Type != tokens.LT {
		return false
	}

	// Peek at i+2 and i+3.
	if p.lex.Peek(3) == (tokens.Token{}) || p.lex.Peek(3).Type == tokens.EOF {
		return false
	}

	return p.lex.Peek(2).Type == tokens.Identifier &&
		p.lex.Peek(3).Type == tokens.Tilde
}

// FindGlobals scans the token stream to pre-register all top-level names
// (types, declarations, enums) into the parser's symbol table. It can be
// called externally when multiple parsers share one symbol table, so that
// all files' globals are visible before any file is fully parsed.
func (p *Parser) FindGlobals(ctx context.Context) {
	p.globalsPass = true

	defer func() {
		p.globalsPass = false
		p.lex.Reset()
		p.Errs = p.Errs[:0]
	}()

	if p.scriptMode {
		// Script files have definition scope (no forward references).
		// Only scan for import statements so that imported packages can be
		// compiled before the file is parsed.
		p.findScriptImports(ctx)
		return
	}

	for p.lex.This().Type != tokens.EOF && ctx.Err() == nil {
		t := p.lex.This()
		exported := false

		if t.Type == tokens.Export {
			if p.scriptMode {
				p.lex.Step()
				continue
			}

			p.lex.Step() // consume export

			exported = true
		}

		qualifier := ast.QualifierImmutable

		switch p.lex.This().Type {
		case tokens.BitAnd:
			// Reference receiver method.
			p.lex.Step() // consume &
		case tokens.Dynamic:
			qualifier = ast.QualifierDynamic

			p.lex.Step() // consume dyn
		case tokens.LParen:
			// Receiver variable: (f : Type) or (var f : &Type)
			p.lex.Step() // consume (

			if p.lex.This().Type == tokens.Variable {
				p.lex.Step() // consume var
			}

			p.lex.Step() // consume receiver identifier

			if p.lex.This().Type == tokens.Colon {
				p.lex.Step() // consume :
			}

			if p.lex.This().Type == tokens.BitAnd {
				p.lex.Step() // consume &
			}
		case tokens.Variable:
			qualifier = ast.QualifierVariable

			p.lex.Step() // consume var
		}

		switch p.lex.This().Type {
		case tokens.GoImport:
			p.lex.Step() // consume goimport

			if p.lex.This().Type == tokens.LParen {
				p.skipGrouped(ctx)
			}
		case tokens.Import:
			p.parseImport(ctx) // process imports during global scan
		case tokens.Identifier:
			switch p.lex.Peek(1).Type {
			case tokens.Colon, tokens.Declaration:
				p.findGlobalDecl(ctx, exported, qualifier)
			case tokens.Dot, tokens.RParen:
				p.findGlobalMethod(ctx, exported)
			case tokens.Tilde:
				p.findGlobalType(ctx, exported)
			case tokens.LT:
				if p.isGenericTypeDecl() {
					p.findGlobalType(ctx, exported)
				} else {
					p.lex.Step() // not a type decl, skip
				}
			default:
				p.lex.Step() // consume token
			}
		case tokens.Package:
			p.lex.Step() // consume package
		default:
			p.lex.Step() // skip unknown token
		}
	}

	// Check for undefined types after the scan.
	for name, sym := range p.symbols.table {
		if sym.Scope == ScanScope && sym.Identifier.Qualifier == ast.QualifierType &&
			types.IsNone(sym.Identifier.ValueType) {
			p.error(sym.Identifier.Token, "undefined type: "+name, "FindGlobals")
		}
	}

	// Check for unresolved forward value references.
	for name, sym := range p.symbols.table {
		if sym.Scope == ScanScope && sym.Identifier.Qualifier != ast.QualifierType &&
			types.IsNone(sym.Identifier.ValueType) {
			p.error(sym.Identifier.Token, "undefined identifier: "+name, "FindGlobals")
		}
	}
}

func (p *Parser) findScriptImports(ctx context.Context) {
	for p.lex.This().Type != tokens.EOF && ctx.Err() == nil {
		switch p.lex.This().Type {
		case tokens.Import:
			p.parseImport(ctx)
		case tokens.GoImport:
			p.lex.Step() // consume goimport

			if p.lex.This().Type == tokens.LParen {
				p.skipGrouped(ctx)
			}
		case tokens.Identifier:
			// Register type aliases so they're visible during parsing.
			if p.lex.Peek(1).Type == tokens.Tilde {
				ident := &ast.Identifier{
					Token:  p.lex.This(),
					Global: true,
				}

				p.symbols.DefineGlobal(ident)
			}

			p.lex.Step()
		default:
			p.lex.Step()
		}
	}
}

func (p *Parser) findGlobalDecl(ctx context.Context, exported bool, qualifier ast.Qualifier) {
	if p.lex.This().Type != tokens.Identifier {
		return
	}

	sym, ok := p.symbols.Resolve(p.lex.This().Literal)
	if ok && !(sym.Scope == ScanScope && types.IsNone(sym.Identifier.ValueType)) {
		// Report redeclare error and advance past the identifier to avoid an infinite loop
		p.error(p.lex.This(), "cannot redeclare variable", "findGlobalDecl")
		p.lex.Step() // consume identifier to make progress

		return
	}

	ident := &ast.Identifier{
		Token:     p.lex.This(),
		Exported:  exported,
		Qualifier: qualifier,
		Global:    true,
	}

	p.lex.Step() // consume identifier

	switch p.lex.This().Type {
	case tokens.Colon:
		p.lex.Step() // consume :

		ident.ValueType = p.parseCombinedType(ctx, exported, true)

		if ident.Token.Literal == "main" {
			procType, isProc := ident.ValueType.(*types.Procedure)
			if !isProc || procType.Function || len(procType.Parameters) != 0 || procType.ReturnType != nil {
				p.error(ident.Token, `"main" can only be declared as proc()`, "findGlobalDecl")

				// Skip past the function body to avoid stalling.
				if p.lex.This().Type == tokens.Assign {
					p.lex.Step() // consume =

					if p.lex.This().Type == tokens.LBrace {
						p.lex.SkipBody()
					}
				}

				return
			}
		}

		p.symbols.DefineGlobal(ident)

		if p.lex.This().Type == tokens.Assign {
			p.lex.Step() // consume =

			if p.lex.This().Type == tokens.LBrace {
				p.lex.SkipBody()
			} else {
				_ = p.expression(ctx, ident.ValueType)
			}
		}
	case tokens.Declaration:
		if ident.Token.Literal == "main" {
			p.error(ident.Token, `"main" can only be declared as proc()`, "findGlobalDecl")

			p.lex.Step() // consume :=

			if p.lex.This().Type == tokens.LBrace {
				p.lex.SkipBody()
			}

			return
		}

		p.lex.Step() // consume :=
		p.symbols.DefineGlobal(ident)

		if p.lex.This().Type == tokens.LBrace {
			p.lex.SkipBody()
		} else {
			_ = p.expression(ctx, types.None)
		}
	default:
		return
	}
}

func (p *Parser) findGlobalType(ctx context.Context, exported bool) {
	ident := &ast.Identifier{
		Token:    p.lex.This(),
		Exported: exported,
		Global:   true,
	}

	p.lex.Step() // consume identifier

	existing, ok := p.symbols.Resolve(ident.Token.Literal)
	if ok {
		// Allow forward-declared type stubs to be resolved.
		if existing.Scope == ScanScope &&
			existing.Identifier.Qualifier == ast.QualifierType &&
			types.IsNone(existing.Identifier.ValueType) {
			ident = existing.Identifier
		} else {
			p.error(p.lex.This(), "cannot redeclare type", "findGlobalType")
			return
		}
	} else {
		// Register type name immediately so it's visible for forward references
		// and survives early returns from parsing errors.
		p.symbols.DefineGlobal(ident)
	}

	// Parse optional type parameters: <T ~ any, K ~ comparable>
	var typeParams []*types.Alias

	if p.lex.This().Type == tokens.LT {
		typeParams = p.parseTypeParams(ctx)
		if typeParams == nil {
			return
		}
	}

	if p.lex.This().Type != tokens.Tilde {
		p.error(p.lex.This(), "expected ~ after type name", "findGlobalType")
		return
	}

	p.lex.Step() // consume ~

	if p.lex.This().Type == tokens.Enum {
		p.lex.Step() // consume enum

		if p.lex.This().Type != tokens.LT {
			p.error(p.lex.This(), "expected < in enum declaration", "findGlobalType")
			return
		}

		p.lex.Step() // consume <

		enumValType := p.parseCombinedType(ctx, exported, true)

		enumType := &types.Enum{ValueType: enumValType}

		if p.lex.This().Type != tokens.GT {
			p.error(p.lex.This(), "expected > in enum declaration", "findGlobalType")
			return
		}

		p.lex.Step() // consume >

		if p.lex.This().Type != tokens.LBrace {
			p.error(p.lex.This(), "expected { in enum literal", "findGlobalType")
			return
		}

		p.lex.Step() // consume {

		for p.lex.This().Type != tokens.EOF && ctx.Err() == nil {
			t := p.lex.This()
			if t.Type == tokens.RBrace {
				break
			}

			if t.Type != tokens.Identifier {
				p.error(t, "expected identifier in enum literal", "findGlobalType")
				p.lex.Step()
				continue
			}

			valIdent := &ast.Identifier{
				Token:     t,
				ValueType: enumValType,
				Exported:  exported,
			}

			p.symbols.DefineEnumValue(ident.Token.Literal, valIdent)

			p.lex.Step() // consume identifier

			if p.lex.This().Type != tokens.Declaration {
				p.error(p.lex.This(), "expected := in enum literal", "findGlobalType")
				return
			}

			p.lex.Step() // consume :=

			enumVal := p.expression(ctx, enumValType)
			if enumVal != ast.ZeroExprIndex {
				enumType.Values = append(enumType.Values, &types.EnumValue{
					Name:  valIdent.Token.Literal,
					Value: p.typeExpr(enumVal),
				})
			}

			if p.lex.This().Type == tokens.Comma {
				p.lex.Step() // consume ,
			}
		}

		ident.ValueType = enumType

		return
	}

	if p.lex.This().Type == tokens.Error {
		errorType := p.parseErrorType(ctx, ident)
		if errorType == nil {
			return
		}

		ident.ValueType = errorType

		return
	}

	// If there are type params, push them into an enclosed scope so that
	// type parameter names (e.g. T) are resolvable in the alias body.
	if len(typeParams) > 0 {
		outer := p.symbols
		p.symbols = NewEnclosedSymbolTable(outer)

		for _, tp := range typeParams {
			p.symbols.Define(&ast.Identifier{
				Token: tokens.Token{
					Type:    tokens.Identifier,
					Literal: tp.Name,
				},
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
				Name:       ident.Token.Literal,
				Derived:    alias,
				Exported:   ident.Exported,
				Global:     ident.Global,
				TypeParams: typeParams,
			}
		}
	}

	ident.ValueType = alias

	// For struct types, register fields in the symbol table.
	if alias != nil && alias.Kind() == types.StructKind {
		p.symbols.Define(ident)
	}
}

func (p *Parser) skipGrouped(ctx context.Context) {
	parenIndex := 0

	for p.lex.This().Type != tokens.EOF && ctx.Err() == nil {
		switch p.lex.This().Type {
		case tokens.LParen:
			parenIndex++
		case tokens.RParen:
			parenIndex--
		}

		p.lex.Step()

		if parenIndex == 0 {
			return
		}
	}
}

func (p *Parser) findGlobalMethod(ctx context.Context, exported bool) {
	// Parse method declaration: Type.Method : proc() = ...
	// Current token is the receiver type name.
	receiverName := p.lex.This().Literal
	p.lex.Step() // consume receiver type name

	if p.lex.This().Type == tokens.RParen {
		p.lex.Step() // consume )
	}

	if p.lex.This().Type != tokens.Dot {
		p.error(p.lex.This(), "expected . after receiver type name", "findGlobalMethod")
		return
	}

	p.lex.Step() // consume .

	if p.lex.This().Type != tokens.Identifier {
		p.error(p.lex.This(), "expected method name after .", "findGlobalMethod")
		return
	}

	// Create a placeholder method identifier to register in the symbol table.
	methodIdent := &ast.Identifier{
		Token:     p.lex.This(),
		Exported:  exported,
		Qualifier: ast.QualifierMethod,
		Global:    true,
	}

	p.lex.Step() // consume identifier

	if p.lex.This().Type != tokens.Colon {
		p.error(p.lex.This(), "expected function type definition after method declaration", "findGlobalMethod")
		return
	}

	p.lex.Step() // consume :

	procType := p.parseProcedureType(ctx, exported, true)
	if procType == nil {
		return
	}

	methodIdent.ValueType = procType

	if p.lex.This().Type != tokens.Assign {
		p.error(p.lex.This(), "expected function body assignment after method type definition", "findGlobalMethod")
		return
	}

	p.lex.Step() // consume =

	p.lex.SkipBody()

	// Register the method in the symbol table so it's available for forward references.
	if err := p.symbols.DefineMethod(receiverName, methodIdent); err != nil {
		p.error(p.lex.This(), err.Error(), "findGlobalMethod")
		return
	}

	// Attach the method to the receiver's underlying struct so that
	// interface satisfaction checks can find it.
	if sym, ok := p.symbols.Resolve(receiverName); ok && sym.Identifier.ValueType != nil {
		method := &types.Method{
			Name:      methodIdent.Token.Literal,
			Procedure: procType,
		}

		switch v := sym.Identifier.ValueType.(type) {
		case *types.Struct:
			v.Methods = append(v.Methods, method)
		case *types.Alias:
			if s, ok := v.Underlying().(*types.Struct); ok {
				s.Methods = append(s.Methods, method)
			}
		}
	}
}
