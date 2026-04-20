package parser

import (
	"context"
	"errors"
	"fmt"
	"slices"

	"github.com/samborkent/cog/internal/ast"
	"github.com/samborkent/cog/internal/tokens"
	"github.com/samborkent/cog/internal/types"
)

type Parser struct {
	tokens   []tokens.Token
	symbols  *SymbolTable
	builtins map[string]BuiltinParser
	filePath string

	Errs              []error
	i                 int
	debug             bool
	scriptMode        bool
	currentReturnType types.Type // return type of the enclosing procedure (for result wrapping)
	definedMethods    map[string]struct{}
}

// NewParserWithSymbols creates a parser that uses the provided symbol table.
// This allows multiple parsers (one per file) to share a single symbol table
// so that global declarations from one file are visible in all others.
func NewParserWithSymbols(tokens []tokens.Token, symbols *SymbolTable, debug bool, fileName string) (*Parser, error) {
	if len(tokens) == 0 {
		return nil, errors.New("no tokens provided to parser")
	}

	p := &Parser{
		tokens:         tokens,
		symbols:        symbols,
		Errs:           make([]error, 0),
		debug:          debug,
		definedMethods: make(map[string]struct{}),
	}

	return p, nil
}

// NewScriptParser creates a parser in script mode for .cogs files.
// Script mode forbids package declarations and export keywords.
func NewScriptParser(tokens []tokens.Token, debug bool) (*Parser, error) {
	return NewScriptParserWithSymbols(tokens, NewSymbolTable(), debug)
}

// NewScriptParserWithSymbols creates a script-mode parser with a shared symbol table.
func NewScriptParserWithSymbols(tokens []tokens.Token, symbols *SymbolTable, debug bool) (*Parser, error) {
	if len(tokens) == 0 {
		return nil, errors.New("no tokens provided to parser")
	}

	p := &Parser{
		tokens:         tokens,
		symbols:        symbols,
		Errs:           make([]error, 0),
		debug:          debug,
		scriptMode:     true,
		definedMethods: make(map[string]struct{}),
	}

	return p, nil
}

func (p *Parser) Parse(ctx context.Context, fileName string) (*ast.File, error) {
	p.FindGlobals(ctx)

	return p.ParseOnly(ctx, fileName)
}

// ParseOnly runs the full parse without calling FindGlobals first.
// Use this when FindGlobals has already been called on a shared symbol table
// across multiple files.
func (p *Parser) ParseOnly(ctx context.Context, fileName string) (*ast.File, error) {
	// Reset position and errors for a clean parse.
	p.i = 0
	p.Errs = make([]error, 0, len(p.Errs))

	p.builtins = map[string]BuiltinParser{
		"cast":  p.parseBuiltinCast,
		"if":    p.parseBuiltinIf,
		"map":   p.parseBuiltinMap,
		"print": p.parseBuiltinPrint,
		"ref":   p.parseBuiltinRef,
		"set":   p.parseBuiltinSet,
		"slice": p.parseBuiltinSlice,
	}

	var pkg *ast.Package

	if p.scriptMode {
		// Script mode: no package declaration allowed.
		if p.tokens[0].Type == tokens.Package {
			p.error(p.tokens[0], "package declaration not allowed in script files", "Parse")
		}

		// Synthesize package main.
		pkg = &ast.Package{
			Token:      tokens.Token{Type: tokens.Package, Literal: "package"},
			Identifier: &ast.Identifier{Name: "main"},
		}
	} else {
		if p.tokens[0].Type != tokens.Package {
			p.error(p.tokens[0], "missing package declaration", "Parse")
		}

		pkg = p.parsePackage()
	}

	f := &ast.File{
		Name:       fileName,
		Package:    pkg,
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
			tokens.Variable,
			tokens.Builtin,
			tokens.If,
			tokens.For,
			tokens.Switch,
			tokens.Return,
			tokens.Break,
			tokens.Continue,
			tokens.BitAnd,
			tokens.LParen:
			ident := p.this().Literal

			node := p.parseStatement(ctx)
			if node != nil {
				if ident == "main" {
					f.ContainsMain = true
				}

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
		case tokens.Import:
			node := p.parseImport()
			if node != nil {
				f.Statements = append(f.Statements, node)
			} else {
				p.synchronize()
			}
		case tokens.EOF:
			break tokenLoop
		default:
			p.error(p.this(), "unexpected token", "Parse")
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

	if p.debug && p.this().Type != tokens.Comment {
		from := p.this().Type.String()
		if slices.Contains([]tokens.Type{
			tokens.Identifier, tokens.StringLiteral, tokens.IntLiteral, tokens.FloatLiteral,
		}, p.this().Type) {
			from = p.this().Literal
		}

		to := p.next().Type.String()
		if slices.Contains([]tokens.Type{
			tokens.Identifier, tokens.StringLiteral, tokens.IntLiteral, tokens.FloatLiteral,
		}, p.next().Type) {
			to = p.next().Literal
		}

		_, _ = fmt.Printf("ADVANCE: ln %d, col %d:\t%s\t\tfrom %q,\tto %q\n",
			p.this().Ln, p.this().Col, scope, from, to)
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
			tokens.Import,
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
		p.Errs = append(p.Errs, fmt.Errorf("\t%s: %v: %s", p.stringToken(t), scope, msg))
	} else {
		p.Errs = append(p.Errs, fmt.Errorf("\t%s: %s", p.stringToken(t), msg))
	}
}

func (p *Parser) stringToken(t tokens.Token) string {
	if t.Literal == "" {
		return fmt.Sprintf("%s:\tln %d, col %d: %s",
			p.filePath, t.Ln, t.Col, t.Type,
		)
	}

	if t.Type == tokens.Builtin {
		return fmt.Sprintf("%s:\tln %d, col %d: @%s",
			p.filePath, t.Ln, t.Col, t.Literal,
		)
	}

	return fmt.Sprintf("%s:\tln %d, col %d: %s: %s",
		p.filePath, t.Ln, t.Col, t.Type, t.Literal,
	)
}
