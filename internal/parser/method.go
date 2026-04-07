package parser

import (
	"context"
	"fmt"

	"github.com/samborkent/cog/internal/ast"
	"github.com/samborkent/cog/internal/tokens"
)

func (p *Parser) parseMethod(ctx context.Context, receiver *ast.Identifier) *ast.Method {
	method := &ast.Method{
		Token: p.this(),
	}

	storedReceiver, ok := p.symbols.Resolve(receiver.Name)
	if !ok {
		p.error(p.this(), fmt.Sprintf("undefined receiver %q", receiver.Name), "parseMethod")
		return nil
	}

	if receiver.Exported && !storedReceiver.Identifier.Exported {
		p.error(p.this(), "exported method not allowed on unexported type", "parseMethod")
		return nil
	}

	method.Receiver = storedReceiver.Identifier

	p.advance("parseMethod .") // consume .

	if p.this().Type != tokens.Identifier {
		p.error(p.this(), "expected method in name in type method declaration", "parseMethod")
		return nil
	}

	methodSymbol, ok := p.symbols.ResolveField(receiver.Name, p.this().Literal)
	if !ok {
		p.error(p.this(), fmt.Sprintf("method %q is undefined", p.this().Literal), "parseMethod")
		return nil
	}

	p.advance("parseMethod identifier") // consume identifier
	p.advance("parseMethod :")          // consume :

	decl := p.parseTypedDeclaration(ctx, methodSymbol.Identifier)
	if decl == nil {
		return nil
	}

	method.Declaration = decl

	return method
}
