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
		p.error(p.this(), "exported methods on unexported types are not allowed", "parseMethod")
		return nil
	}

	method.Receiver = storedReceiver.Identifier

	p.advance("parseMethod .") // consume .

	if p.this().Type != tokens.Identifier {
		p.error(p.this(), "expected method in name in type method declaration", "parseMethod")
		return nil
	}

	methodIdent := &ast.Identifier{
		Token:     p.this(),
		Name:      p.this().Literal,
		Exported:  receiver.Exported, // TODO: check if correct? you may only export methods on exported types.
		Qualifier: ast.QualifierMethod,
		Global:    true, // TODO: check if methods may be declared in local scope
	}

	p.advance("parseMethod identifier") // consume identifier
	p.advance("parseMethod :")          // consume :

	decl := p.parseTypedDeclaration(ctx, methodIdent)
	if decl == nil {
		return nil
	}

	if err := p.symbols.DefineMethod(receiver.Name, methodIdent); err != nil {
		p.error(p.this(), err.Error(), "parseMethod")
		return nil
	}

	method.Declaration = decl

	return method
}
