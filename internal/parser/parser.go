package parser

import (
	"context"
	"errors"
	"fmt"

	"github.com/samborkent/cog/internal/ast"
	"github.com/samborkent/cog/internal/tokens"
)

type Parser struct {
	tokens   []tokens.Token
	symbols  *SymbolTable
	builtins map[string]BuiltinParser

	Errs  []error
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
		Errs:    make([]error, 0),
		debug:   debug,
	}

	return p, nil
}

func (p *Parser) Parse(ctx context.Context, fileName string) (*ast.File, error) {
	p.findGlobals(ctx)

	// Reset errors, so they're only printed once.
	p.Errs = make([]error, 0, len(p.Errs))

	p.builtins = map[string]BuiltinParser{
		"if":    p.parseBuiltinIf,
		"map":   p.parseBuiltinMap,
		"print": p.parseBuiltinPrint,
		"ptr":   p.parseBuiltinPtr,
		"set":   p.parseBuiltinSet,
		"slice": p.parseBuiltinSlice,
	}

	// Static checks.
	if p.tokens[0].Type != tokens.Package {
		p.error(p.tokens[0], "missing package declaration", "Parse")
	}

	f := &ast.File{
		Name:       fileName,
		Package:    p.parsePackage(),
		Statements: []ast.Statement{},
	}

	// Iterate tokens.
tokenLoop:
	for p.this().Type != tokens.EOF {
		if ctx.Err() != nil {
			return f, fmt.Errorf("parser error:\n%w", errors.Join(p.Errs...))
		}

		prev := p.i

		switch p.this().Type {
		case tokens.Comment:
			f.Statements = append(f.Statements, &ast.Comment{
				Token: p.this(),
				Text:  p.this().Literal,
			})
			p.advance("Parse comment")
		case tokens.Dynamic,
			tokens.Export,
			tokens.Identifier,
			tokens.Variable:
			node := p.parseStatement(ctx)
			if node != nil {
				f.Statements = append(f.Statements, node)
			} else {
				p.synchronize()
			}
		case tokens.GoImport:
			node := p.parseGoImport()
			if node != nil {
				f.Statements = append(f.Statements, node)
			} else {
				p.synchronize()
			}
		case tokens.EOF:
			break tokenLoop
		default:
			p.error(p.this(), "unknown token", "Parse")
			p.synchronize()
		}

		// Guard against infinite loops: if no progress was made, force advance.
		if p.i == prev {
			p.advance("Parse recovery")
		}

		// Check for EOF again, in case it was reached during parsing.
		if p.this().Type == tokens.EOF {
			break tokenLoop
		}
	}

	if err := errors.Join(p.Errs...); err != nil {
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
		return
	}

	if p.debug {
		fmt.Printf("DEBUG: %s: advance from %q to %q\n", scope, p.this(), p.next())
	}

	p.i++
}

// synchronize advances tokens until it finds a token that can begin a new statement.
// This enables error recovery by skipping malformed input.
func (p *Parser) synchronize() {
	for p.this().Type != tokens.EOF {
		switch p.this().Type {
		case tokens.Identifier,
			tokens.Builtin,
			tokens.Comment,
			tokens.If,
			tokens.For,
			tokens.Switch,
			tokens.Return,
			tokens.Export,
			tokens.Dynamic,
			tokens.Variable,
			tokens.GoImport,
			tokens.RBrace,
			tokens.Break,
			tokens.Continue:
			return
		default:
			p.advance("synchronize")
		}
	}
}

func (p *Parser) error(t tokens.Token, msg string, scope ...string) {
	if len(scope) > 0 {
		p.Errs = append(p.Errs, fmt.Errorf("\t%s: %v: %s", t, scope, msg))
	} else {
		p.Errs = append(p.Errs, fmt.Errorf("\t%s: %s", t, msg))
	}
}
