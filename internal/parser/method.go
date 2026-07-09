package parser

import (
	"context"
	"fmt"

	"github.com/samborkent/cog/internal/ast"
	"github.com/samborkent/cog/internal/tokens"
	"github.com/samborkent/cog/internal/types"
)

func (p *Parser) parseMethod(ctx context.Context, receiverVar *ast.Identifier, typeName string, exported, reference bool) ast.NodeIndex {
	if p.symbols.Outer != nil {
		p.error(p.lex.This(), "method declaration is only allowed in global scope", "parseMethod")
		return ast.ZeroNodeIndex
	}

	receiverType, ok := p.symbols.ResolveType(typeName)
	if !ok && p.globalsPass {
		// Forward type reference for receiver — register stub.
		receiverType = &ast.Type{
			Token: p.lex.Peek(-1), // Previous token is type token
			Alias: &types.Alias{
				Name: typeName,
			},
		}

		p.symbols.DefineType(receiverType)
	} else if !ok {
		p.error(p.lex.This(), fmt.Sprintf("undefined receiver type %q", typeName), "parseMethod")
		return ast.ZeroNodeIndex
	}

	if exported && !receiverType.Alias.Exported && !types.IsNone(receiverType.Alias.Derived) {
		p.error(p.lex.This(), "exported method not allowed on unexported type", "parseMethod")
		return ast.ZeroNodeIndex
	}

	var receiverVarType types.Type

	if reference {
		receiverVarType = &types.Reference{
			Value: receiverType.Alias,
		}
	} else {
		receiverVarType = receiverType.Alias
	}

	receiverVar.ValueType = receiverVarType

	if p.lex.This().Type != tokens.Dot {
		p.error(p.lex.This(), "expected . after receiver in method declaration", "parseMethod")
		return ast.ZeroNodeIndex
	}

	p.lex.Step() // consume .

	if p.lex.This().Type != tokens.Identifier {
		p.error(p.lex.This(), "expected method in name in type method declaration", "parseMethod")
		return ast.ZeroNodeIndex
	}

	methodToken := p.lex.This()

	existingMethod := receiverType.Alias.Method(methodToken.Literal)
	if existingMethod == nil {
		if p.globalsPass {
			// Register method during globals pass if it's not registered yet.
			receiverType.Alias.RegisterMethods(&types.Method{
				Name: methodToken.Literal,
			})
		} else {
			p.error(p.lex.This(), "method is undefined", "parseMethod")
			return ast.ZeroNodeIndex
		}
	} else {
		p.error(p.lex.This(), fmt.Sprintf("duplicate method %q definition for type %q", methodToken.Literal, typeName), "parseMethod")
		return ast.ZeroNodeIndex
	}

	p.lex.Step() // consume identifier

	if p.lex.This().Type != tokens.Colon {
		p.error(p.lex.This(), "expected : after method name in method declaration", "parseMethod")
		return ast.ZeroNodeIndex
	}

	p.lex.Step() // consume :

	procType := p.parseProcedureType(ctx, exported, reference)
	if procType == nil {
		return ast.ZeroNodeIndex
	}

	if p.lex.This().Type != tokens.Assign {
		p.error(p.lex.This(), "expected = after type in method declaration", "parseMethod")
		return ast.ZeroNodeIndex
	}

	p.lex.Step() // consume =

	if receiverVar != nil {
		p.symbols = NewEnclosedSymbolTable(p.symbols)
		p.symbols.DefineIdent(receiverVar)
		p.symbols.MarkUsed(receiverVar.String())

		prevReceiver := p.currentReceiver
		p.currentReceiver = receiverVar

		defer func() {
			p.checkUnused()
			p.symbols = p.symbols.Outer
			p.currentReceiver = prevReceiver
		}()
	}

	procLiteral := p.primary(ctx, procType)
	if procLiteral == ast.ZeroExprIndex {
		return ast.ZeroNodeIndex
	}

	// Reject variable receiver on func (pure functions cannot mutate).
	if receiverVar != nil && receiverVar.Qualifier == ast.QualifierVariable {
		if procType.Function {
			p.error(receiverVar.Token, "func cannot have a variable receiver; use proc for methods that mutate state", "parseMethod")
			return ast.ZeroNodeIndex
		}
	}

	return p.ast.NewMethod(methodToken, exported, receiverVar, receiverVarType, procType, procLiteral)
}
