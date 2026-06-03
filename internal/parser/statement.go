package parser

import (
	"context"
	"fmt"

	"github.com/samborkent/cog/internal/ast"
	"github.com/samborkent/cog/internal/tokens"
	"github.com/samborkent/cog/internal/types"
)

func (p *Parser) parseStatement(ctx context.Context) ast.NodeIndex {
	switch p.lex.This().Type {
	case tokens.Comment:
		commentToken := p.lex.This()
		p.lex.Step() // consume comment
		return p.ast.NewComment(commentToken)
	case tokens.BitAnd:
		// Skip, get it with prev in identifier case.
		p.lex.Step() // consume &

		if p.lex.This().Type != tokens.Identifier {
			p.error(p.lex.This(), "expected identifier after '&'", "parseStatement")
			return ast.ZeroNodeIndex
		}

		return p.parseStatement(ctx)
	case tokens.Break, tokens.Continue:
		branchToken := p.lex.This()

		p.lex.Step() // consume break or continue

		var label *ast.Identifier

		if p.lex.This().Type == tokens.Identifier {
			// TODO: should use symbol table.
			label = &ast.Identifier{
				Token:     p.lex.This(),
				ValueType: types.None,
			}

			p.lex.Step() // consume label
		}

		return p.ast.NewBranch(branchToken, label)
	case tokens.Builtin:
		t := p.lex.This()

		p.lex.Step() // consume @

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
	case tokens.Export:
		if p.scriptMode {
			p.error(p.lex.This(), "export keyword not allowed in script files", "parseStatement")
			p.lex.Step() // consume export

			return ast.ZeroNodeIndex
		}

		if p.symbols.Outer != nil {
			p.error(p.lex.This(), "export statements are only allowed in the global scope", "parseStatement")
			return ast.ZeroNodeIndex
		}

		p.lex.Step() // consume export

		var reference bool

		switch p.lex.This().Type {
		case tokens.BitAnd:
			// Reference method receiver.
			reference = true

			p.lex.Step() // consume &

			fallthrough
		case tokens.Identifier:
			ident := &ast.Identifier{
				Token:     p.lex.This(),
				Exported:  true,
				Qualifier: ast.QualifierImmutable,
				Global:    true,
			}

			p.lex.Step() // consume identifier

			switch p.lex.This().Type {
			case tokens.Colon:
				p.lex.Step() // consume :

				return p.parseTypedDeclaration(ctx, ident)
			case tokens.Declaration:
				return p.parseDeclaration(ctx, p.lex.This(), ident)
			case tokens.Tilde, tokens.LT:
				return p.parseTypeAlias(ctx, ident)
			case tokens.Dot:
				return p.parseMethod(ctx, nil, ident.Token.Literal, true, reference)
			default:
				p.error(p.lex.This(), "unexpected token following exported identifier", "parseStatement")
				p.lex.Step() // consume unknown token

				return ast.ZeroNodeIndex
			}
		case tokens.LParen:
			// Exported method with explicit receiver: export (f : Type).Method
			p.lex.Step() // consume (

			qualifier := ast.QualifierImmutable

			if p.lex.This().Type == tokens.Variable {
				qualifier = ast.QualifierVariable
				p.lex.Step() // consume var
			}

			if p.lex.This().Type != tokens.Identifier {
				p.error(p.lex.This(), "expected identifier after ( in exported method declaration", "parseStatement")
				return ast.ZeroNodeIndex
			}

			receiverIdent := &ast.Identifier{
				Token:     p.lex.This(),
				Qualifier: qualifier,
			}

			p.lex.Step() // consume identifier

			if p.lex.This().Type != tokens.Colon {
				p.error(p.lex.This(), "expected : after receiver variable in exported method declaration", "parseStatement")
				return ast.ZeroNodeIndex
			}

			p.lex.Step() // consume :

			exportRef := false

			if p.lex.This().Type == tokens.BitAnd {
				exportRef = true
				p.lex.Step() // consume &
			}

			if p.lex.This().Type != tokens.Identifier {
				p.error(p.lex.This(), "expected type identifier after : in exported method declaration", "parseStatement")
				return ast.ZeroNodeIndex
			}

			typeSymbol, ok := p.symbols.Resolve(p.lex.This().Literal)
			if !ok || typeSymbol.Identifier.Qualifier != ast.QualifierType {
				p.error(p.lex.This(), "unknown type found in type declaration", "parseType")
				return ast.ZeroNodeIndex
			}

			receiverIdent.ValueType = &types.Alias{
				Name:    typeSymbol.Identifier.Token.Literal,
				Derived: typeSymbol.Identifier.ValueType,
			}

			p.lex.Step() // consume identifier

			if p.lex.This().Type != tokens.RParen {
				p.error(p.lex.This(), "expected ) after receiver in exported method declaration", "parseStatement")
				return ast.ZeroNodeIndex
			}

			p.lex.Step() // consume )

			return p.parseMethod(ctx,
				receiverIdent,
				typeSymbol.Identifier.Token.Literal,
				true,
				exportRef,
			)
		default:
			p.error(p.lex.This(), "unexpected token found after export", "parseStatement")
			return ast.ZeroNodeIndex
		}
	case tokens.For:
		return p.parseForStatement(ctx)
	case tokens.Identifier:
		reference := p.lex.Peek(-1).Type == tokens.BitAnd

		ident := &ast.Identifier{
			Token:     p.lex.This(),
			Exported:  false,
			Qualifier: ast.QualifierImmutable,
			Global:    p.symbols.Outer == nil,
		}

		// Do not skip identifier for function call or imported package selector;
		// parse as expression instead.
		if p.lex.Peek(1).Type == tokens.LParen {
			// Direct function call: e.g. someFunc(...)
		} else if p.lex.Peek(1).Type == tokens.LT {
			// Could be generic call (genFunc<utf8>(...)) or type alias.
			// If the symbol is a generic callable, don't consume the identifier
			// so expression parsing handles it.
			if sym, ok := p.symbols.Resolve(p.lex.This().Literal); ok {
				if proc, ok := sym.Identifier.ValueType.(*types.Procedure); ok && len(proc.TypeParams) > 0 {
					// Generic function call — let expression handle it.
				} else {
					p.lex.Step() // consume identifier
				}
			} else {
				p.lex.Step() // consume identifier
			}
		} else if p.lex.Peek(1).Type == tokens.Dot {
			if _, isImport := p.symbols.ResolveCogImport(p.lex.This().Literal); isImport {
				// Imported package selector: e.g. pkg.Func(...)
			} else if p.symbols.Outer == nil {
				// Only consume in global scope, for method declarations.
				p.lex.Step() // consume identifier
			}
		} else {
			p.lex.Step() // consume identifier
		}

		switch p.lex.This().Type {
		case tokens.Assign:
			// Assignment
			if !p.scriptMode && p.symbols.Outer == nil {
				p.error(p.lex.This(), "no assignment allowed in package scope, use declaration instead", "parseStatement")
				return ast.ZeroNodeIndex
			}

			return p.parseAssignment(ctx, ident)
		case tokens.Colon:
			// Typed declaration or label
			// Check if next token is a control flow keyword for labeled statements
			switch p.lex.Peek(1).Type {
			case tokens.For:
				// Labeled for statement: label: for
				return p.parseForStatement(ctx)
			case tokens.If:
				// Labeled if statement: label: if
				return p.parseIfStatement(ctx)
			case tokens.Match:
				// Labeled match statement: label: match
				return p.parseMatch(ctx)
			case tokens.Switch:
				// Labeled switch statement: label: switch
				return p.parseSwitch(ctx)
			}

			p.lex.Step() // advance :

			return p.parseTypedDeclaration(ctx, ident)
		case tokens.Declaration:
			// Untyped declaration
			return p.parseDeclaration(ctx, p.lex.This(), ident)
		case tokens.Identifier:
			// Procedure / method call or selector assignment
			identToken := p.lex.This()

			callExpr := p.expression(ctx, types.None)
			if callExpr == ast.ZeroExprIndex {
				return ast.ZeroNodeIndex
			}

			// Selector assignment: f.value = "changed"
			if p.lex.This().Type == tokens.Assign {
				selector, ok := p.ast.Expr(callExpr).(*ast.Selector)
				if !ok {
					p.error(p.lex.This(), "invalid assignment target", "parseStatement")
					return ast.ZeroNodeIndex
				}

				// Resolve the receiver and check mutability.
				symbol, ok := p.symbols.Resolve(ident.Token.Literal)
				if !ok {
					p.error(ident.Token, "unknown identifier", "parseStatement")
					return ast.ZeroNodeIndex
				}

				if symbol.Identifier.Qualifier != ast.QualifierVariable {
					p.error(ident.Token, "cannot assign to field of immutable receiver", "parseStatement")

					// Skip the rest of the assignment to continue parsing.
					p.lex.Step() // consume =
					_ = p.expression(ctx, types.None)

					return ast.ZeroNodeIndex
				}

				selectorToken := identToken
				selectorToken.Literal += "." + selector.Fields[len(selector.Fields)-1].Token.Literal

				// Rule 13: Check if the field is reserved by a defer.
				if p.symbols.IsFieldReserved(ident.Token.Literal, selector.Fields[len(selector.Fields)-1].Token.Literal) {
					p.error(ident.Token, fmt.Sprintf("cannot reassign %s: field '%s' reserved by defer", ident.Token.Literal, selector.Fields[len(selector.Fields)-1].Token.Literal), "parseStatement")
					// Skip the rest of the assignment to continue parsing.
					p.lex.Step() // consume =
					_ = p.expression(ctx, types.None)
					return ast.ZeroNodeIndex
				}
				// Rule 19: Re-assigning a moved-out field revives it.
				p.symbols.ReviveField(ident.Token.Literal, selector.Fields[len(selector.Fields)-1].Token.Literal)

				// Determine the field's type and export status for type-checking and transpilation.
				fieldName := selector.Fields[len(selector.Fields)-1].Token.Literal

				var fieldType types.Type
				var fieldExported bool

				if len(selector.Fields) > 1 {
					lastField := selector.Fields[len(selector.Fields)-1]
					fieldType = lastField.ValueType
				}

				// Look up the struct field to get the actual exported status.
				var structType *types.Struct
				switch st := symbol.Type().Underlying().(type) {
				case *types.Struct:
					structType = st
				case *types.Alias:
					if underlying, ok := st.Underlying().(*types.Struct); ok {
						structType = underlying
					}
				}

				if structType != nil {
					if f := structType.Field(fieldName); f != nil {
						fieldExported = f.Exported
						if fieldType == nil {
							fieldType = f.Type
						}
					}
				}

				assignmentToken := p.lex.This()
				p.lex.Step() // consume =

				expr := p.expression(ctx, fieldType)
				if expr == ast.ZeroExprIndex {
					return ast.ZeroNodeIndex
				}

				exprType := p.ast.Expr(expr).Type()

				// Rule 19 (RHS): var to var field consumption.
				if sourceIdent, ok := p.ast.Expr(expr).(*ast.Identifier); ok &&
					sourceIdent.Qualifier == ast.QualifierVariable &&
					sourceIdent.Token.Literal != "_" {
					if !p.symbols.IsAlive(sourceIdent.Token.Literal) {
						p.error(assignmentToken, fmt.Sprintf("cannot assign %s: variable is consumed", sourceIdent.Token.Literal), "parseStatement")
						return ast.ZeroNodeIndex
					}
					p.symbols.MarkConsumed(sourceIdent.Token.Literal)
				}

				if !types.Equal(fieldType, exprType) &&
					!types.AssignableTo(exprType, fieldType) {
					p.error(assignmentToken, fmt.Sprintf("type mismatch: cannot assign %q to field %q of type %q", exprType, fieldName, fieldType), "parseStatement")
					return ast.ZeroNodeIndex
				}

				return p.ast.NewAssignment(assignmentToken, symbol.Identifier, expr,
					ast.WithField(fieldName, fieldExported),
				)
			}

			return p.ast.NewExpressionStatement(identToken, callExpr)
		case tokens.Tilde, tokens.LT:
			// Type declaration (possibly generic)
			return p.parseTypeAlias(ctx, ident)
		case tokens.Dot:
			if p.symbols.Outer == nil {
				// Method declaration (only possible in global scope)
				return p.parseMethod(ctx, nil, ident.Token.Literal, false, reference)
			}

			fallthrough
		default:
			p.error(p.lex.This(), "unexpected token found after identifier", "parseStatement")
			return ast.ZeroNodeIndex
		}
	case tokens.If:
		return p.parseIfStatement(ctx)
	case tokens.LParen:
		p.lex.Step() // consume (

		qualifier := ast.QualifierImmutable

		if p.lex.This().Type == tokens.Variable {
			qualifier = ast.QualifierVariable
			p.lex.Step() // consume var
		}

		if p.lex.This().Type != tokens.Identifier {
			p.error(p.lex.This(), "expected identifier after ( in method declaration", "parseStatement")
			return ast.ZeroNodeIndex
		}

		receiverIdent := &ast.Identifier{
			Token:     p.lex.This(),
			Qualifier: qualifier,
		}

		p.lex.Step() // consume identifier

		if p.lex.This().Type != tokens.Colon {
			p.error(p.lex.This(), "expected : after receiver variable in method declaration", "parseStatement")
			return ast.ZeroNodeIndex
		}

		p.lex.Step() // consume :

		reference := false

		if p.lex.This().Type == tokens.BitAnd {
			reference = true
			p.lex.Step() // consume &
		}

		if p.lex.This().Type != tokens.Identifier {
			p.error(p.lex.This(), "expected type identifier after : in method declaration", "parseStatement")
			return ast.ZeroNodeIndex
		}

		typeSymbol, ok := p.symbols.Resolve(p.lex.This().Literal)
		if !ok || typeSymbol.Identifier.Qualifier != ast.QualifierType {
			p.error(p.lex.This(), "unknown type found in type declaration", "parseType")
			return ast.ZeroNodeIndex
		}

		receiverIdent.ValueType = &types.Alias{
			Name:    typeSymbol.Identifier.Token.Literal,
			Derived: typeSymbol.Identifier.ValueType,
		}

		p.lex.Step() // consume identifier

		if p.lex.This().Type != tokens.RParen {
			p.error(p.lex.This(), "expected ) after receiver in method declaration", "parseStatement")
			return ast.ZeroNodeIndex
		}

		p.lex.Step() // consume )

		return p.parseMethod(ctx,
			receiverIdent,
			typeSymbol.Identifier.Token.Literal,
			false,
			reference,
		)
	case tokens.Defer:
		deferToken := p.lex.This()
		p.lex.Step()

		// Save ownership state before expression parsing so we can detect
		// double-defer even when the call expression overwrites stateReserved.
		preOwn, _ := p.symbols.snapshotOwnership()

		exprIdx := p.expression(ctx, types.None)
		if exprIdx == ast.ZeroExprIndex {
			return ast.ZeroNodeIndex
		}

		switch p.ast.Expr(exprIdx).(type) {
		case *ast.Call, *ast.ProcedureLiteral:
		default:
			p.error(deferToken, "defer requires a procedure call or closure", "parseStatement")
			return ast.ZeroNodeIndex
		}

		// Rule 13: Defer reserves captured var variables.
		captured := p.collectCapturedVars(exprIdx)
		capturedFields := p.collectCapturedStructFields(exprIdx)
		for name := range captured {
			// Check reservation constraint using the pre-call snapshot.
			if prevState, ok := preOwn[name]; ok && prevState == stateReserved {
				p.error(deferToken, fmt.Sprintf("cannot defer %s: already reserved by a defer", name), "parseStatement")
				return ast.ZeroNodeIndex
			}

			// If the variable was already consumed before the defer expression,
			// reject — the deferred call would find the variable dead at scope exit.
			if prevState, ok := preOwn[name]; ok && prevState == stateConsumed {
				p.error(deferToken, fmt.Sprintf("cannot defer %s: variable is already consumed", name), "parseStatement")
				return ast.ZeroNodeIndex
			}

			// Undo any consumption from the call expression so MarkReserved can apply.
			owning := p.symbols.owningScope(name)
			if owning != nil {
				if state, ok := owning.ownership.Load(name); ok && state == stateConsumed {
					owning.ownership.Delete(name)
				}
			}
			if err := p.symbols.MarkReserved(name); err != nil {
				p.error(deferToken, err.Error(), "parseStatement")
				return ast.ZeroNodeIndex
			}
		}
		for sv, fields := range capturedFields {
			for f := range fields {
				if err := p.symbols.MarkFieldReserved(sv, f); err != nil {
					p.error(deferToken, err.Error(), "parseStatement")
					return ast.ZeroNodeIndex
				}
			}
		}

		return p.ast.NewDefer(deferToken, exprIdx)
	case tokens.Match:
		return p.parseMatch(ctx)
	case tokens.Return:
		returnToken := p.lex.This()

		p.lex.Step() // consume return

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
				p.symbols.IsValueChecked(ident.Token.Literal) {
				// A checked result identifier used as a bare expression denotes
				// its success value; wrap it for a result-typed return.
				exprIndex = p.ast.NewResultLiteral(returnToken, p.currentReturnType, exprIndex, exprType.Kind() == types.ErrorKind)
			}
		}

		// Rule 10: Return transfers ownership.
		{
			e := p.ast.Expr(exprIndex)
			if ident, ok := e.(*ast.Identifier); ok && ident.Qualifier == ast.QualifierVariable {
				// Check struct fields before consuming the whole variable.
				if ident.ValueType.Kind() == types.StructKind && !p.symbols.IsFullyAlive(ident.Token.Literal) {
					var deadField string
					if sym, ok := p.symbols.Resolve(ident.Token.Literal); ok && sym.Identifier.ValueType.Kind() == types.StructKind {
						if st, ok := sym.Identifier.ValueType.Underlying().(*types.Struct); ok {
							for _, f := range st.Fields {
								if !p.symbols.IsFieldAlive(ident.Token.Literal, f.Name) {
									deadField = f.Name
									break
								}
							}
						}
					}
					p.error(returnToken, fmt.Sprintf("cannot move %s: field '%s' is already moved out", ident.Token.Literal, deadField), "parseReturn")
					return ast.ZeroNodeIndex
				}
				if !p.symbols.IsAlive(ident.Token.Literal) {
					p.error(returnToken, fmt.Sprintf("cannot return %s: variable is consumed", ident.Token.Literal), "parseReturn")
					return ast.ZeroNodeIndex
				}
				p.symbols.MarkConsumed(ident.Token.Literal)
			}
		}

		return p.ast.NewReturn(returnToken, exprIndex)
	case tokens.Switch:
		return p.parseSwitch(ctx)
	case tokens.EOF:
		return ast.ZeroNodeIndex
	default:
		p.error(p.lex.This(), "unknown token", "parseStatement")
		p.lex.Step() // consume unknown token

		return ast.ZeroNodeIndex
	}
}
