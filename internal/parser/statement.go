package parser

import (
	"context"

	"github.com/samborkent/cog/internal/ast"
	"github.com/samborkent/cog/internal/tokens"
	"github.com/samborkent/cog/internal/types"
)

func (p *Parser) parseStatement(ctx context.Context) ast.NodeIndex {
	switch p.this().Type {
	case tokens.Comment:
		commentToken := p.this()
		p.advance("parseStatement comment")
		return p.ast.NewComment(commentToken)
	case tokens.BitAnd:
		// Skip, get it with prev in identifier case.
		p.advance("parseStatement ref") // consume &

		if p.this().Type != tokens.Identifier {
			p.error(p.this(), "expected identifier after '&'", "parseStatement")
			return ast.ZeroNodeIndex
		}

		return p.parseStatement(ctx)
	case tokens.Break, tokens.Continue:
		branchToken := p.this()

		p.advance("parseStatement branch") // consume break or continue

		var label *ast.Identifier

		if p.this().Type == tokens.Identifier {
			// TODO: should use symbol table.
			label = &ast.Identifier{
				Token:     p.this(),
				Name:      p.this().Literal,
				ValueType: types.None,
			}

			p.advance("parseStatement branch label") // consume label
		}

		return p.ast.NewBranch(branchToken, label)
	case tokens.Builtin:
		t := p.this()

		p.advance("parseStatement builtin") // consume @

		builtinParser, ok := p.builtins[t.Literal]
		if !ok {
			p.error(t, "unknown builtin function", "parseStatement")
			return ast.ZeroNodeIndex
		}

		expr := builtinParser(ctx, t, types.None)
		if expr == ast.ZeroExprIndex {
			return ast.ZeroNodeIndex
		}

		return p.ast.NewExpressionStatement(t, expr)
	case tokens.Dynamic:
		// Skip, get it with prev in identifier case.
		p.advance("parseStatement dyn") // consume dyn

		if p.symbols.Outer != nil {
			p.error(p.this(), "dynamic scope variable declarations are only allowed in package scope", "parseStatement")
			return ast.ZeroNodeIndex
		}

		return p.parseStatement(ctx)
	case tokens.Export:
		if p.scriptMode {
			p.error(p.this(), "export keyword not allowed in script files", "parseStatement")
			p.advance("parseStatement export script") // consume export

			return ast.ZeroNodeIndex
		}

		if p.symbols.Outer != nil {
			p.error(p.this(), "export statements are only allowed in the global scope", "parseStatement")
			return ast.ZeroNodeIndex
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

				return p.parseTypedDeclaration(ctx, ident)
			case tokens.Declaration:
				return p.parseDeclaration(ctx, p.this(), ident)
			case tokens.Tilde, tokens.LT:
				return p.parseTypeAlias(ctx, ident)
			case tokens.Dot:
				return p.parseMethod(ctx, nil, ident.Name, true, reference)
			default:
				p.error(p.this(), "unexpected token following exported identifier", "parseStatement")
				p.advance("parseStatement export error") // consume unknown token

				return ast.ZeroNodeIndex
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
				return ast.ZeroNodeIndex
			}

			receiverIdent := &ast.Identifier{
				Token:     p.this(),
				Name:      p.this().Literal,
				Qualifier: qualifier,
			}

			p.advance("parseStatement export receiver identifier") // consume identifier

			if p.this().Type != tokens.Colon {
				p.error(p.this(), "expected : after receiver variable in exported method declaration", "parseStatement")
				return ast.ZeroNodeIndex
			}

			p.advance("parseStatement export :") // consume :

			exportRef := false

			if p.this().Type == tokens.BitAnd {
				exportRef = true
				p.advance("parseStatement export &") // consume &
			}

			if p.this().Type != tokens.Identifier {
				p.error(p.this(), "expected type identifier after : in exported method declaration", "parseStatement")
				return ast.ZeroNodeIndex
			}

			typeSymbol, ok := p.symbols.Resolve(p.this().Literal)
			if !ok || typeSymbol.Identifier.Qualifier != ast.QualifierType {
				p.error(p.this(), "unknown type found in type declaration", "parseType")
				return ast.ZeroNodeIndex
			}

			receiverIdent.ValueType = &types.Alias{
				Name:    typeSymbol.Identifier.Name,
				Derived: typeSymbol.Identifier.ValueType,
			}

			p.advance("parseStatement export receiver type") // consume identifier

			if p.this().Type != tokens.RParen {
				p.error(p.this(), "expected ) after receiver in exported method declaration", "parseStatement")
				return ast.ZeroNodeIndex
			}

			p.advance("parseStatement export )") // consume )

			return p.parseMethod(ctx,
				receiverIdent,
				typeSymbol.Identifier.Name,
				true,
				exportRef,
			)
		default:
			p.error(p.this(), "unexpected token found after export", "parseStatement")
			return ast.ZeroNodeIndex
		}
	case tokens.For:
		return p.parseForStatement(ctx)
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
		case tokens.Assign:
			// Assignment
			if !p.scriptMode && p.symbols.Outer == nil {
				p.error(p.this(), "no assignment allowed in package scope, use declaration instead", "parseStatement")
				return ast.ZeroNodeIndex
			}

			return p.parseAssignment(ctx, ident)
		case tokens.Colon:
			// Typed declaration or label
			switch p.this().Type {
			case tokens.For:
				// Labeled for statement
				return p.parseForStatement(ctx)
			case tokens.If:
				// Labeled if statement
				return p.parseIfStatement(ctx)
			case tokens.Match:
				// Labeled match statement
				return p.parseMatch(ctx)
			case tokens.Switch:
				// Labeled switch statement
				return p.parseSwitch(ctx)
			}

			p.advance("parseStatement :") // advance :

			return p.parseTypedDeclaration(ctx, ident)
		case tokens.Declaration:
			// Untyped declaration
			return p.parseDeclaration(ctx, p.this(), ident)
		case tokens.Identifier:
			// Procedure / method call or selector assignment
			identToken := p.this()

			callExpr := p.expression(ctx, types.None)
			if callExpr == ast.ZeroExprIndex {
				return ast.ZeroNodeIndex
			}

			// Selector assignment: f.value = "changed"
			if p.this().Type == tokens.Assign {
				selector, ok := p.ast.Expr(callExpr).(*ast.Selector)
				if !ok {
					p.error(p.this(), "invalid assignment target", "parseStatement")
					return ast.ZeroNodeIndex
				}

				// Resolve the receiver and check mutability.
				symbol, ok := p.symbols.Resolve(ident.Name)
				if !ok {
					p.error(ident.Token, "unknown identifier", "parseStatement")
					return ast.ZeroNodeIndex
				}

				if symbol.Identifier.Qualifier != ast.QualifierVariable {
					p.error(ident.Token, "cannot assign to field of immutable receiver", "parseStatement")

					// Skip the rest of the assignment to continue parsing.
					p.advance("parseStatement error =") // consume =
					_ = p.expression(ctx, types.None)

					return ast.ZeroNodeIndex
				}

				// Build a selector identifier for the assignment.
				selectorIdent := &ast.Identifier{
					Token:     ident.Token,
					Name:      ident.Name + "." + selector.Fields[len(selector.Fields)-1].Name,
					Qualifier: ast.QualifierVariable,
				}

				return p.parseAssignment(ctx, selectorIdent)
			}

			return p.ast.NewExpressionStatement(identToken, callExpr)
		case tokens.Tilde, tokens.LT:
			// Type declaration (possibly generic)
			return p.parseTypeAlias(ctx, ident)
		case tokens.Dot:
			if p.symbols.Outer == nil {
				// Method declaration (only possible in global scope)
				p.parseMethod(ctx, nil, ident.Name, false, reference)
			}

			fallthrough
		default:
			p.error(p.this(), "unexpected token found after identifier", "parseStatement")
			return ast.ZeroNodeIndex
		}
	case tokens.If:
		return p.parseIfStatement(ctx)
	case tokens.LParen:
		p.advance("parseStatement (") // consume (

		qualifier := ast.QualifierImmutable

		if p.this().Type == tokens.Variable {
			qualifier = ast.QualifierVariable
			p.advance("parseStatement var") // consume var
		}

		if p.this().Type != tokens.Identifier {
			p.error(p.this(), "expected identifier after ( in method declaration", "parseStatement")
			return ast.ZeroNodeIndex
		}

		receiverIdent := &ast.Identifier{
			Token:     p.this(),
			Name:      p.this().Literal,
			Qualifier: qualifier,
		}

		p.advance("parseStatement receiver identifier") // consume identifier

		if p.this().Type != tokens.Colon {
			p.error(p.this(), "expected : after receiver variable in method declaration", "parseStatement")
			return ast.ZeroNodeIndex
		}

		p.advance("parseStatement :") // consume :

		reference := false

		if p.this().Type == tokens.BitAnd {
			reference = true
			p.advance("parseStatement &") // consume &
		}

		if p.this().Type != tokens.Identifier {
			p.error(p.this(), "expected type identifier after : in method declaration", "parseStatement")
			return ast.ZeroNodeIndex
		}

		typeSymbol, ok := p.symbols.Resolve(p.this().Literal)
		if !ok || typeSymbol.Identifier.Qualifier != ast.QualifierType {
			p.error(p.this(), "unknown type found in type declaration", "parseType")
			return ast.ZeroNodeIndex
		}

		receiverIdent.ValueType = &types.Alias{
			Name:    typeSymbol.Identifier.Name,
			Derived: typeSymbol.Identifier.ValueType,
		}

		p.advance("parseStatement receiver type") // consume identifier

		if p.this().Type != tokens.RParen {
			p.error(p.this(), "expected ) after receiver in method declaration", "parseStatement")
			return ast.ZeroNodeIndex
		}

		p.advance("parseStatement )") // consume (

		return p.parseMethod(ctx,
			receiverIdent,
			typeSymbol.Identifier.Name,
			false,
			reference,
		)
	case tokens.Match:
		return p.parseMatch(ctx)
	case tokens.Return:
		returnToken := p.this()

		p.advance("parseStatement return") // consume return

		var resultType *types.Result

		if p.currentReturnType != nil {
			resultType, _ = p.currentReturnType.Underlying().(*types.Result)
		}

		exprIndex := p.expression(ctx, types.None)
		if exprIndex == ast.ZeroExprIndex {
			return ast.ZeroNodeIndex
		}

		// If the enclosing procedure returns a Result type, only wrap
		// value/error variants. Returning a full result value must pass
		// through unchanged to preserve its IsError state.
		if resultType != nil {
			expr := p.ast.Expr(exprIndex)
			exprType := expr.Type()

			if _, isVariant := resultExprState(resultType, exprType); isVariant {
				exprIndex = p.ast.NewResultLiteral(returnToken, p.currentReturnType, exprIndex, exprType.Kind() == types.ErrorKind)
			} else if ident, ok := expr.(*ast.Identifier); ok &&
				expr.Type().Kind() == types.ResultKind &&
				p.symbols.IsValueChecked(ident.Name) {
				// A checked result identifier used as a bare expression denotes
				// its success value; wrap it for a result-typed return.
				exprIndex = p.ast.NewResultLiteral(returnToken, p.currentReturnType, exprIndex, exprType.Kind() == types.ErrorKind)
			}
		}

		return p.ast.NewReturn(returnToken, exprIndex)
	case tokens.Switch:
		return p.parseSwitch(ctx)
	case tokens.Variable:
		// Skip, get it with prev in identifier case.
		p.advance("parseStatement var") // consume var

		if !p.scriptMode && p.symbols.Outer == nil {
			p.error(p.this(), "variable declarations are not allowed in package scope", "parseStatement")
			return ast.ZeroNodeIndex
		}

		return p.parseStatement(ctx)
	case tokens.EOF:
		return ast.ZeroNodeIndex
	default:
		p.error(p.this(), "unknown token", "parseStatement")
		p.advance("parseStatement unknown") // consume unknown token

		return ast.ZeroNodeIndex
	}
}
