package parser

import (
	"context"

	"github.com/samborkent/cog/internal/ast"
	"github.com/samborkent/cog/internal/tokens"
	"github.com/samborkent/cog/internal/types"
)

func (p *Parser) parseStatement(ctx context.Context) ast.Statement {
	switch p.this().Type {
	case tokens.Comment:
		node := &ast.Comment{
			Token: p.this(),
			Text:  p.this().Literal,
		}
		p.advance("parseStatement comment")

		return node
	case tokens.BitAnd:
		// Skip, get it with prev in identifier case.
		p.advance("parseStatement ref") // consume &

		if p.this().Type != tokens.Identifier {
			p.error(p.this(), "expected identifier after '&'", "parseStatement")
			return nil
		}

		return p.parseStatement(ctx)
	case tokens.Break, tokens.Continue:
		node := &ast.Branch{
			Token: p.this(),
		}

		p.advance("parseStatement branch") // consume break or continue

		if p.this().Type == tokens.Identifier {
			node.Label = &ast.Identifier{
				Token:     p.this(),
				Name:      p.this().Literal,
				ValueType: types.None,
			}

			p.advance("parseStatement branch label") // consume label
		}

		return node
	case tokens.Builtin:
		t := p.this()

		p.advance("parseStatement builtin") // consume @

		builtinParser, ok := p.builtins[t.Literal]
		if !ok {
			p.error(t, "unknown builtin function", "parseStatement")
			return nil
		}

		node := builtinParser(ctx, t, types.None)
		if node == nil {
			return nil
		}

		return &ast.ExpressionStatement{
			Token:      t,
			Expression: node,
		}
	case tokens.Dynamic:
		// Skip, get it with prev in identifier case.
		p.advance("parseStatement dyn") // consume dyn

		if p.symbols.Outer != nil {
			p.error(p.this(), "dynamic scope variable declarations are only allowed in package scope", "parseStatement")
			return nil
		}

		return p.parseStatement(ctx)
	case tokens.Export:
		if p.scriptMode {
			p.error(p.this(), "export keyword not allowed in script files", "parseStatement")
			p.advance("parseStatement export script") // consume export

			return nil
		}

		if p.symbols.Outer != nil {
			p.error(p.this(), "export statements are only allowed in the global scope", "parseStatement")
			return nil
		}

		p.advance("parseStatement export") // consume export

		var reference bool

		switch p.this().Type {
		case tokens.BitAnd:
			// Reference method receiver.
			reference = true

			p.advance("parseStatement export ref") // consume &

			fallthrough
		case tokens.Identifier:
			ident := &ast.Identifier{
				Token:     p.this(),
				Name:      p.this().Literal,
				Exported:  true,
				Qualifier: ast.QualifierImmutable,
				Global:    true,
			}

			p.advance("parseStatement export ident") // consume identifier

			switch p.this().Type {
			case tokens.Colon:
				p.advance("parseStatement export ident :") // consume :

				decl := p.parseTypedDeclaration(ctx, ident)
				if decl != nil {
					return decl
				}

				return nil
			case tokens.Declaration:
				decl := p.parseDeclaration(ctx, ident)
				if decl != nil {
					return decl
				}

				return nil
			case tokens.Tilde, tokens.LT:
				typeDecl := p.parseTypeAlias(ctx, ident)
				if typeDecl != nil {
					return typeDecl
				}

				return nil
			case tokens.Dot:
				// Method declaration
				if node := p.parseMethod(ctx, nil, ident.Name, true, reference); node != nil {
					return node
				}

				return nil
			default:
				p.error(p.this(), "unexpected token following exported identifier", "parseStatement")
				p.advance("parseStatement export error") // consume unknown token

				return nil
			}
		case tokens.LParen:
			// Exported method with explicit receiver: export (f : Type).Method
			p.advance("parseStatement export (") // consume (

			qualifier := ast.QualifierImmutable

			if p.this().Type == tokens.Variable {
				qualifier = ast.QualifierVariable
				p.advance("parseStatement export var") // consume var
			}

			if p.this().Type != tokens.Identifier {
				p.error(p.this(), "expected identifier after ( in exported method declaration", "parseStatement")
				return nil
			}

			receiverIdent := &ast.Identifier{
				Token:     p.this(),
				Name:      p.this().Literal,
				Qualifier: qualifier,
			}

			p.advance("parseStatement export receiver identifier") // consume identifier

			if p.this().Type != tokens.Colon {
				p.error(p.this(), "expected : after receiver variable in exported method declaration", "parseStatement")
				return nil
			}

			p.advance("parseStatement export :") // consume :

			exportRef := false

			if p.this().Type == tokens.BitAnd {
				exportRef = true
				p.advance("parseStatement export &") // consume &
			}

			if p.this().Type != tokens.Identifier {
				p.error(p.this(), "expected type identifier after : in exported method declaration", "parseStatement")
				return nil
			}

			typeSymbol, ok := p.symbols.Resolve(p.this().Literal)
			if !ok || typeSymbol.Identifier.Qualifier != ast.QualifierType {
				p.error(p.this(), "unknown type found in type declaration", "parseType")
				return nil
			}

			receiverIdent.ValueType = &types.Alias{
				Name:    typeSymbol.Identifier.Name,
				Derived: typeSymbol.Identifier.ValueType,
			}

			p.advance("parseStatement export receiver type") // consume identifier

			if p.this().Type != tokens.RParen {
				p.error(p.this(), "expected ) after receiver in exported method declaration", "parseStatement")
				return nil
			}

			p.advance("parseStatement export )") // consume )

			method := p.parseMethod(ctx,
				receiverIdent,
				typeSymbol.Identifier.Name,
				true,
				exportRef,
			)
			if method == nil {
				return nil
			}

			return method
		default:
			p.error(p.this(), "unexpected token found after export", "parseStatement")
			return nil
		}
	case tokens.For:
		if node := p.parseForStatement(ctx); node != nil {
			return node
		}

		return nil
	case tokens.Identifier:
		qualifier := ast.QualifierImmutable

		switch p.prev().Type {
		case tokens.Variable:
			qualifier = ast.QualifierVariable
		case tokens.Dynamic:
			qualifier = ast.QualifierDynamic
		}

		// Check if previous token was &, for reference method receiver.
		reference := p.prev().Type == tokens.BitAnd

		ident := &ast.Identifier{
			Token:     p.this(),
			Name:      p.this().Literal,
			Exported:  false,
			Qualifier: qualifier,
			Global:    p.symbols.Outer == nil,
		}

		// Do not skip identifier for function call or imported package selector;
		// parse as expression instead.
		if p.next().Type == tokens.LParen {
			// Direct function call: e.g. someFunc(...)
		} else if p.next().Type == tokens.LT {
			// Could be generic call (genFunc<utf8>(...)) or type alias.
			// If the symbol is a generic callable, don't consume the identifier
			// so expression parsing handles it.
			if sym, ok := p.symbols.Resolve(p.this().Literal); ok {
				if proc, ok := sym.Identifier.ValueType.(*types.Procedure); ok && len(proc.TypeParams) > 0 {
					// Generic function call — let expression handle it.
				} else {
					p.advance("parseStatement ident") // consume identifier
				}
			} else {
				p.advance("parseStatement ident") // consume identifier
			}
		} else if p.next().Type == tokens.Dot {
			if _, isImport := p.symbols.ResolveCogImport(p.this().Literal); isImport {
				// Imported package selector: e.g. pkg.Func(...)
			} else if p.symbols.Outer == nil {
				// Only consume in global scope, for method declarations.
				p.advance("parseStatement ident") // consume identifier
			}
		} else {
			p.advance("parseStatement ident") // consume identifier
		}

		switch p.this().Type {
		case tokens.Assign: // Assignment
			if !p.scriptMode && p.symbols.Outer == nil {
				p.error(p.this(), "no assignment allowed in package scope, use declaration instead", "parseStatement")
				return nil
			}

			if a := p.parseAssignment(ctx, ident); a != nil {
				return a
			}

			return nil
		case tokens.Colon: // Typed declaration or label
			p.advance("parseStatement ident :") // consume ':'

			switch p.this().Type {
			case tokens.For:
				// Labeled for statement
				forStatement := p.parseForStatement(ctx)
				if forStatement == nil {
					return nil
				}

				ident.ValueType = types.None
				forStatement.Label = &ast.Label{
					Token: ident.Token,
					Label: ident,
				}

				return forStatement
			case tokens.If:
				// Labeled if statement
				ifStatement := p.parseIfStatement(ctx)
				if ifStatement == nil {
					return nil
				}

				ident.ValueType = types.None
				ifStatement.Label = &ast.Label{
					Token: ident.Token,
					Label: ident,
				}

				return ifStatement
			case tokens.Switch:
				// Labeled switch statement
				switchStatement := p.parseSwitch(ctx)
				if switchStatement == nil {
					return nil
				}

				ident.ValueType = types.None
				switchStatement.Label = &ast.Label{
					Token: ident.Token,
					Label: ident,
				}

				return switchStatement
			}

			if d := p.parseTypedDeclaration(ctx, ident); d != nil {
				return d
			}

			return nil
		case tokens.Declaration: // Untyped declaration
			if d := p.parseDeclaration(ctx, ident); d != nil {
				return d
			}

			return nil
		case tokens.Identifier: // Procedure / method call or selector assignment
			identToken := p.this()

			callExpr := p.expression(ctx, types.None)
			if callExpr == nil {
				return nil
			}

			// Selector assignment: f.value = "changed"
			if p.this().Type == tokens.Assign {
				selector, ok := callExpr.(*ast.Selector)
				if !ok {
					p.error(p.this(), "invalid assignment target", "parseStatement")
					return nil
				}

				// Resolve the receiver and check mutability.
				symbol, ok := p.symbols.Resolve(ident.Name)
				if !ok {
					p.error(ident.Token, "unknown identifier", "parseStatement")
					return nil
				}

				if symbol.Identifier.Qualifier != ast.QualifierVariable {
					p.error(ident.Token, "cannot assign to field of immutable receiver", "parseStatement")

					// Skip the rest of the assignment to continue parsing.
					p.advance("parseStatement error =") // consume =
					_ = p.expression(ctx, types.None)

					return nil
				}

				// Build a selector identifier for the assignment.
				selectorIdent := &ast.Identifier{
					Token:     ident.Token,
					Name:      ident.Name + "." + selector.Field.Name,
					Qualifier: ast.QualifierVariable,
				}

				return p.parseAssignment(ctx, selectorIdent)
			}

			return &ast.ExpressionStatement{
				Token:      identToken,
				Expression: callExpr,
			}
		case tokens.Tilde, tokens.LT: // Type declaration (possibly generic)
			typeDecl := p.parseTypeAlias(ctx, ident)
			if typeDecl != nil {
				return typeDecl
			}

			return nil
		case tokens.Dot:
			if p.symbols.Outer == nil {
				// Method declaration (only possible in global scope)
				if node := p.parseMethod(ctx, nil, ident.Name, false, reference); node != nil {
					return node
				}

				return nil
			}

			fallthrough
		default:
			p.error(p.this(), "unexpected token found after identifier", "parseStatement")
			return nil
		}
	case tokens.If:
		if node := p.parseIfStatement(ctx); node != nil {
			return node
		}

		return nil
	case tokens.LParen:
		p.advance("parseStatement (") // consume (

		qualifier := ast.QualifierImmutable

		if p.this().Type == tokens.Variable {
			qualifier = ast.QualifierVariable
			p.advance("parseStatement var") // consume var
		}

		if p.this().Type != tokens.Identifier {
			p.error(p.this(), "expected identifier after ( in method declaration", "parseStatement")
			return nil
		}

		receiverIdent := &ast.Identifier{
			Token:     p.this(),
			Name:      p.this().Literal,
			Qualifier: qualifier,
		}

		p.advance("parseStatement receiver identifier") // consume identifier

		if p.this().Type != tokens.Colon {
			p.error(p.this(), "expected : after receiver variable in method declaration", "parseStatement")
			return nil
		}

		p.advance("parseStatement :") // consume :

		reference := false

		if p.this().Type == tokens.BitAnd {
			reference = true
			p.advance("parseStatement &") // consume &
		}

		if p.this().Type != tokens.Identifier {
			p.error(p.this(), "expected type identifier after : in method declaration", "parseStatement")
			return nil
		}

		typeSymbol, ok := p.symbols.Resolve(p.this().Literal)
		if !ok || typeSymbol.Identifier.Qualifier != ast.QualifierType {
			p.error(p.this(), "unknown type found in type declaration", "parseType")
			return nil
		}

		receiverIdent.ValueType = &types.Alias{
			Name:    typeSymbol.Identifier.Name,
			Derived: typeSymbol.Identifier.ValueType,
		}

		p.advance("parseStatement receiver type") // consume identifier

		if p.this().Type != tokens.RParen {
			p.error(p.this(), "expected ) after receiver in method declaration", "parseStatement")
			return nil
		}

		p.advance("parseStatement )") // consume (

		method := p.parseMethod(ctx,
			receiverIdent,
			typeSymbol.Identifier.Name,
			false,
			reference,
		)
		if method == nil {
			return nil
		}

		return method
	case tokens.Match:
		if node := p.parseMatch(ctx); node != nil {
			return node
		}

		return nil
	case tokens.Return:
		node := &ast.Return{
			Token: p.this(),
		}

		p.advance("parseStatement return") // consume return

		node.Values = make([]ast.Expression, 0)

		var resultType *types.Result
		if p.currentReturnType != nil {
			resultType, _ = p.currentReturnType.Underlying().(*types.Result)
		}

		for p.this().Type != tokens.EOF {
			expr := p.expression(ctx, types.None)
			if expr != nil {
				// If the enclosing procedure returns a Result type, only wrap
				// value/error variants. Returning a full result value must pass
				// through unchanged to preserve its IsError state.
				if resultType != nil {
					if _, isVariant := resultExprState(resultType, expr); isVariant {
						expr = wrapResultLiteral(node.Token, p.currentReturnType, expr)
					} else if ident, ok := expr.(*ast.Identifier); ok && expr.Type().Kind() == types.ResultKind && p.symbols.IsValueChecked(ident.Name) {
						// A checked result identifier used as a bare expression denotes
						// its success value; wrap it for a result-typed return.
						expr = wrapResultLiteral(node.Token, p.currentReturnType, expr)
					}
				}

				node.Values = append(node.Values, expr)
			}

			if p.this().Type != tokens.Comma {
				break
			}

			p.advance("parseStatement return ,") // consume comma
		}

		return node
	case tokens.Switch:
		if node := p.parseSwitch(ctx); node != nil {
			return node
		}

		return nil
	case tokens.Variable:
		// Skip, get it with prev in identifier case.
		p.advance("parseStatement var") // consume var

		if !p.scriptMode && p.symbols.Outer == nil {
			p.error(p.this(), "variable declarations are not allowed in package scope", "parseStatement")
			return nil
		}

		return p.parseStatement(ctx)
	case tokens.EOF:
		return nil
	default:
		p.error(p.this(), "unknown token", "parseStatement")
		p.advance("parseStatement unknown") // consume unknown token

		return nil
	}
}
