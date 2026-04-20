package parser

import (
	"context"
	"fmt"

	"github.com/samborkent/cog/internal/ast"
	"github.com/samborkent/cog/internal/tokens"
	"github.com/samborkent/cog/internal/types"
)

func (p *Parser) parseMethod(ctx context.Context, receiver *ast.Identifier, typeName string, exported, reference bool) *ast.Method {
	method := &ast.Method{
		Token:    p.this(),
		Export:   exported,
		Receiver: receiver,
	}

	storedReceiver, ok := p.symbols.Resolve(typeName)
	if !ok {
		p.error(p.this(), fmt.Sprintf("undefined receiver type %q", typeName), "parseMethod")
		return nil
	}

	if storedReceiver.Identifier.Qualifier != ast.QualifierType {
		p.error(p.this(), "encountered non-type method receiver "+storedReceiver.Identifier.Name, "parseMethod")
		return nil
	}

	if exported && !storedReceiver.Identifier.Exported {
		p.error(p.this(), "exported method not allowed on unexported type", "parseMethod")
		return nil
	}

	if reference {
		method.Type = &types.Reference{
			Value: &types.Alias{
				Name:     storedReceiver.Identifier.Name,
				Derived:  storedReceiver.Identifier.ValueType,
				Exported: storedReceiver.Identifier.Exported,
				Global:   storedReceiver.Identifier.Global,
			},
		}
	} else {
		method.Type = &types.Alias{
			Name:     storedReceiver.Identifier.Name,
			Derived:  storedReceiver.Identifier.ValueType,
			Exported: storedReceiver.Identifier.Exported,
			Global:   storedReceiver.Identifier.Global,
		}
	}

	if p.this().Type != tokens.Dot {
		p.error(p.this(), "expected . after reciver in method declaration", "parseMethod")
		return nil
	}

	p.advance("parseMethod .") // consume .

	if p.this().Type != tokens.Identifier {
		p.error(p.this(), "expected method in name in type method declaration", "parseMethod")
		return nil
	}

	methodName := p.this().Literal
	methodSymbol, ok := p.symbols.ResolveField(typeName, methodName)
	if !ok {
		p.error(p.this(), "method is undefined", "parseMethod")
		return nil
	}

	// Check for duplicate method definition.
	methodKey := typeName + "." + methodName
	if _, exists := p.definedMethods[methodKey]; exists {
		p.error(p.this(), fmt.Sprintf("duplicate method definition: %s", methodKey), "parseMethod")
		return nil
	}

	p.definedMethods[methodKey] = struct{}{}

	p.advance("parseMethod identifier") // consume identifier
	p.advance("parseMethod :")          // consume :

	if method.Receiver != nil {
		p.symbols = NewEnclosedSymbolTable(p.symbols)
		p.symbols.Define(receiver)

		defer func() { p.symbols = p.symbols.Outer }()
	}

	decl := p.parseTypedDeclaration(ctx, methodSymbol.Identifier)

	if decl == nil {
		return nil
	}

	// Reject variable receiver on func (pure functions cannot mutate).
	if receiver != nil && receiver.Qualifier == ast.QualifierVariable {
		if procType, ok := decl.Assignment.Identifier.ValueType.(*types.Procedure); ok && procType.Function {
			p.error(receiver.Token, "func cannot have a variable receiver; use proc for methods that mutate state", "parseMethod")
			return nil
		}
	}

	method.Declaration = decl

	return method
}
