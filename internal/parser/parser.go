package parser

import (
	"context"
	"errors"
	"fmt"

	"github.com/samborkent/cog/internal/ast"
	"github.com/samborkent/cog/internal/tokens"
	"github.com/samborkent/cog/internal/types"
)

type Section []tokens.Token

type Parser struct {
	tokens   []tokens.Token
	symbols  *SymbolTable
	builtins map[string]BuiltinParser

	errs  []error
	i     int
	debug bool
}

func NewParser(tokens []tokens.Token, debug bool) (*Parser, error) {
	if len(tokens) == 0 {
		return nil, errors.New("no tokens provided to parser")
	}

	p := &Parser{
		tokens:  tokens,
		symbols: NewSymbolTable(),
		errs:    make([]error, 0),
		debug:   debug,
	}

	return p, nil
}

func (p *Parser) findGlobals(ctx context.Context) {
tokenLoop:
	for p.this().Type != tokens.EOF {
		switch p.this().Type {
		case tokens.Constant:
			p.findGlobalConst(ctx, false)
		case tokens.Export:
			p.advance("findGlobals export") // consume export

			switch p.this().Type {
			case tokens.Constant:
				p.findGlobalConst(ctx, true)
			case tokens.Identifier:
				ident := &ast.Identifier{
					Token:    p.this(),
					Name:     p.this().Literal,
					Exported: true,
				}

				p.advance("findGlobals identifier") // consume identifier

				switch p.this().Type {
				case tokens.Colon:
					p.findGlobalProc(ctx, ident)
				case tokens.Tilde:
					p.findGlobalType(ctx, ident)
				}
			}
		case tokens.Identifier:
			ident := &ast.Identifier{
				Token:    p.this(),
				Name:     p.this().Literal,
				Exported: false,
			}

			p.advance("findGlobals identifier") // consume identifier

			switch p.this().Type {
			case tokens.Colon:
				p.findGlobalProc(ctx, ident)
			case tokens.Tilde:
				p.findGlobalType(ctx, ident)
			}
		case tokens.EOF:
			break tokenLoop
		default:
			p.advance("findGlobals") // consume token
		}
	}

	p.i = 0
}

func (p *Parser) findGlobalConst(ctx context.Context, exported bool) {
	p.advance("findGlobalConst const") // consume const

	if p.this().Type == tokens.Identifier {
		_, ok := p.symbols.Resolve(p.this().Literal)
		if ok {
			p.error(p.this(), "cannot redeclare variable", "findGlobalConst")
			return
		}

		ident := &ast.Identifier{
			Token:    p.this(),
			Name:     p.this().Literal,
			Exported: exported,
		}

		p.advance("findGlobalConst const identifier") // consume identifier

		if p.this().Type == tokens.Colon || p.this().Type == tokens.Declaration {
			p.symbols.DefineGlobal(ident, SymbolKindConstant)

			p.advance("findGlobalConst const :/:=") // consume : or :=

			if p.this().Type == tokens.Enum {
				p.advance("findGlobalConst enum") // consume enum

				if p.this().Type != tokens.LBracket {
					p.error(p.this(), "expected [ in enum declaration", "findGlobalConst")
					return
				}

				p.advance("findGlobalConst enum [") // consume [

				enumValType := p.parseCombinedType(ctx, exported)

				p.symbols.Update(ident.Name, &types.Enum{Value: enumValType})

				p.advance("findGlobalConst enum type") // consume type

				if p.this().Type != tokens.RBracket {
					p.error(p.this(), "expected ] in enum declaration", "findGlobalConst")
					return
				}

				p.advance("findGlobalConst enum ]") // consume ]

				if p.this().Type != tokens.Assign {
					p.error(p.this(), "expected = in enum declaration", "findGlobalConst")
					return
				}

				p.advance("findGlobalConst enum =") // consume =

				if p.this().Type != tokens.LBrace {
					p.error(p.this(), "expected { in enum literal", "findGlobalConst")
					return
				}

				p.advance("findGlobalConst enum literal {") // consume {

				for p.this().Type != tokens.RBrace {
					if ctx.Err() != nil {
						return
					}

					if p.this().Type != tokens.Identifier {
						p.error(p.this(), "expected identifier in enum literal", "findGlobalConst")
						return
					}

					p.symbols.DefineEnumValue(ident.Name, &ast.Identifier{
						Token:     p.this(),
						Name:      p.this().Literal,
						ValueType: enumValType,
						Exported:  exported,
					})

					p.advance("findGlobalConst enum literal identifier") // consume identifier

					if p.this().Type != tokens.Declaration {
						p.error(p.this(), "expected := in enum literal", "findGlobalConst")
						return
					}

					p.advance("findGlobalConst enum literal :=") // consume :=

					_ = p.expression(ctx, enumValType)

					if p.this().Type == tokens.Comma {
						p.advance("findGlobalConst enum literal ,") // consume ,
					}
				}
			}
		}
	}
}

func (p *Parser) findGlobalProc(ctx context.Context, ident *ast.Identifier) {
	p.advance("findGlobals identifier :") // consume :

	if p.this().Type == tokens.Procedure || p.this().Type == tokens.Function {
		_, ok := p.symbols.Resolve(p.this().Literal)
		if ok {
			p.error(p.this(), "cannot redeclare functions", "findGlobals")
		} else {
			// TODO: support constant functions
			procedure := p.parseProcedure(ctx, ident, true)
			if procedure != nil {
				p.symbols.DefineProcdure(procedure, SymbolKindVariable, true)
			}
		}
	}
}

func (p *Parser) findGlobalType(ctx context.Context, ident *ast.Identifier) {
	_, ok := p.symbols.Resolve(ident.Name)
	if ok {
		p.error(p.this(), "cannot redeclare type", "findGlobals")
		return
	}

	p.advance("findGlobalType ~") // consume ~

	if p.this().Type == tokens.Enum {
		// TODO: avoid code duplication
		p.advance("findGlobalType enum") // consume enum

		if p.this().Type != tokens.LBracket {
			p.error(p.this(), "expected [ in enum declaration", "findGlobalType")
			return
		}

		p.advance("findGlobalType enum [") // consume [

		enumValType := p.parseCombinedType(ctx, ident.Exported)

		if p.this().Type != tokens.RBracket {
			p.error(p.this(), "expected ] in enum declaration", "findGlobalType")
			return
		}

		p.advance("findGlobalType enum ]") // consume ]

		if p.this().Type != tokens.LBrace {
			p.error(p.this(), "expected { after enum declaration", "findGlobalType")
			return
		}

		p.advance("findGlobalType {") // consume {

		for p.this().Type != tokens.RBrace {
			if ctx.Err() != nil {
				return
			}

			if p.this().Type != tokens.Identifier {
				p.error(p.this(), "expected identifier in enum literal", "findGlobalType")
				return
			}

			enumIdent := &ast.Identifier{
				Token:     p.this(),
				Name:      p.this().Literal,
				ValueType: enumValType,
				Exported:  ident.Exported,
			}

			p.symbols.DefineEnumValue(ident.Name, enumIdent)

			p.advance("parseEnum enum literal identifier") // consume identifier

			if p.this().Type != tokens.Declaration {
				p.error(p.this(), "expected := in enum literal", "parseEnum")
				return
			}

			p.advance("parseEnum enum literal :=") // consume :=

			_ = p.expression(ctx, enumValType)

			if p.this().Type == tokens.Comma {
				p.advance("findGlobalConst enum literal ,") // consume ,
			}
		}

		ident.ValueType = &types.Enum{
			Value: enumValType,
		}
	} else {
		alias := p.parseCombinedType(ctx, ident.Exported)
		if alias == nil {
			return
		}

		ident.ValueType = alias
	}

	p.symbols.DefineGlobal(ident, SymbolKindType)
}

func (p *Parser) skipScope(ctx context.Context) {
	braceIndex := 0

	for {
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

// TODO: create separate parse methods, this requires keeping index state in parser
func (p *Parser) Parse(ctx context.Context) (*ast.File, error) {
	p.findGlobals(ctx)

	// Reset errors, so they're only printed once.
	p.errs = make([]error, 0, len(p.errs))

	p.builtins = map[string]BuiltinParser{
		"if":    p.parseBuiltinIf,
		"print": p.parseBuiltinPrint,
	}

	// Static checks.
	if p.tokens[0].Type != tokens.Package {
		p.error(p.tokens[0], "missing package declaration", "Parse")
	}

	f := &ast.File{
		Package:    p.parsePackage(),
		Statements: []ast.Statement{},
	}

	// Iterate tokens.
tokenLoop:
	for p.this().Type != tokens.EOF {
		if ctx.Err() != nil {
			return f, fmt.Errorf("parser error:\n%w", errors.Join(p.errs...))
		}

		switch p.this().Type {
		case tokens.Constant, tokens.Export, tokens.Identifier:
			node := p.parseStatement(ctx)
			if node != nil {
				f.Statements = append(f.Statements, node)
			}
		case tokens.GoImport:
			node := p.parseGoImport()
			if node != nil {
				f.Statements = append(f.Statements, node)
			}

			// TODO: remove
			continue
		case tokens.EOF:
			break tokenLoop
		default:
			p.error(p.this(), "unknown token", "Parse")
			return f, fmt.Errorf("parser error:\n%w", errors.Join(p.errs...))
		}

		// Check for EOF again, in case it was reached during parsing.
		if p.this().Type == tokens.EOF {
			break tokenLoop
		}
	}

	if err := errors.Join(p.errs...); err != nil {
		return f, fmt.Errorf("parser error:\n%w", err)
	}

	return f, nil
}

func (p *Parser) prev() tokens.Token {
	if p.i == 0 {
		return tokens.Token{}
	}

	return p.tokens[p.i-1]
}

func (p *Parser) this() tokens.Token {
	return p.tokens[p.i]
}

func (p *Parser) next() tokens.Token {
	if p.i >= len(p.tokens)-1 {
		return tokens.Token{}
	}

	return p.tokens[p.i+1]
}

func (p *Parser) advance(scope string) {
	if p.i >= len(p.tokens)-1 {
		panic("reached end of token stream")
	}

	if p.debug {
		fmt.Printf("DEBUG: %s: advance from %q to %q\n", scope, p.this(), p.next())
	}

	p.i++
}

func (p *Parser) error(t tokens.Token, msg string, scope ...string) {
	if len(scope) > 0 {
		p.errs = append(p.errs, fmt.Errorf("\t%s: %v: %s", t, scope, msg))
	} else {
		p.errs = append(p.errs, fmt.Errorf("\t%s: %s", t, msg))
	}
}
