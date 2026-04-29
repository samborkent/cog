package parser

import (
	"context"
	"fmt"

	"github.com/samborkent/cog/internal/ast"
	"github.com/samborkent/cog/internal/tokens"
	"github.com/samborkent/cog/internal/types"
)

func (p *Parser) parseMethod(ctx context.Context, receiver *ast.Identifier, typeName string, exported, reference bool) ast.NodeIndex {
	methodToken := p.this()

	storedReceiver, ok := p.symbols.Resolve(typeName)
	if !ok {
		p.error(p.this(), fmt.Sprintf("undefined receiver type %q", typeName), "parseMethod")
		return ast.ZeroNodeIndex
	}

	if storedReceiver.Identifier.Qualifier != ast.QualifierType {
		p.error(p.this(), "encountered non-type method receiver "+storedReceiver.Identifier.Name, "parseMethod")
		return ast.ZeroNodeIndex
	}

	if exported && !storedReceiver.Identifier.Exported {
		p.error(p.this(), "exported method not allowed on unexported type", "parseMethod")
		return ast.ZeroNodeIndex
	}

	var methodType types.Type

	if reference {
		methodType = &types.Reference{
			Value: &types.Alias{
				Name:     storedReceiver.Identifier.Name,
				Derived:  storedReceiver.Identifier.ValueType,
				Exported: storedReceiver.Identifier.Exported,
				Global:   storedReceiver.Identifier.Global,
			},
		}
	} else {
		methodType = &types.Alias{
			Name:     storedReceiver.Identifier.Name,
			Derived:  storedReceiver.Identifier.ValueType,
			Exported: storedReceiver.Identifier.Exported,
			Global:   storedReceiver.Identifier.Global,
		}
	}

	if p.this().Type != tokens.Dot {
		p.error(p.this(), "expected . after reciver in method declaration", "parseMethod")
		return ast.ZeroNodeIndex
	}

	p.advance("parseMethod .") // consume .

	if p.this().Type != tokens.Identifier {
		p.error(p.this(), "expected method in name in type method declaration", "parseMethod")
		return ast.ZeroNodeIndex
	}

	methodName := p.this().Literal
	methodSymbol, ok := p.symbols.ResolveField(typeName, methodName)
	if !ok {
		p.error(p.this(), "method is undefined", "parseMethod")
		return ast.ZeroNodeIndex
	}

	// Check for duplicate method definition.
	methodKey := typeName + "." + methodName
	if _, exists := p.definedMethods[methodKey]; exists {
		p.error(p.this(), fmt.Sprintf("duplicate method definition: %s", methodKey), "parseMethod")
		return ast.ZeroNodeIndex
	}

	p.definedMethods[methodKey] = struct{}{}

	p.advance("parseMethod identifier") // consume identifier
	p.advance("parseMethod :")          // consume :

	if receiver != nil {
		p.symbols = NewEnclosedSymbolTable(p.symbols)
		p.symbols.Define(receiver)

		defer func() { p.symbols = p.symbols.Outer }()
	}

	decl := p.parseTypedDeclaration(ctx, methodSymbol.Identifier)
	if decl == ast.ZeroNodeIndex {
		return ast.ZeroNodeIndex
	}

	// Reject variable receiver on func (pure functions cannot mutate).
	if receiver != nil && receiver.Qualifier == ast.QualifierVariable {
		declExpr := p.ast.Node(decl)

		if procType, ok := declExpr.(*ast.Declaration).Assignment.Identifier.ValueType.(*types.Procedure); ok && procType.Function {
			p.error(receiver.Token, "func cannot have a variable receiver; use proc for methods that mutate state", "parseMethod")
			return ast.ZeroNodeIndex
		}
	}

	return p.ast.NewMethod(methodToken, exported, receiver, methodType, decl)
}
