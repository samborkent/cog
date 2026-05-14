package parser

import (
	"arena"
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/samborkent/cog/internal/ast"
	"github.com/samborkent/cog/internal/lexer"
	"github.com/samborkent/cog/internal/tokens"
	"github.com/samborkent/cog/internal/types"
)

const (
	// TODO: base this on heuristics
	errorPreallocationSize     = 16
	statementPreallocationSize = 16
)

type Parser struct {
	lex      *lexer.Lexer
	symbols  *SymbolTable
	builtins map[string]BuiltinParser
	filePath string

	ast               *ast.AST
	Errs              []error
	scriptMode        bool
	currentReturnType types.Type // return type of the enclosing procedure (for result wrapping)
	definedMethods    map[string]struct{}
}

// NewParserWithSymbols creates a parser that uses the provided symbol table.
// This allows multiple parsers (one per file) to share a single symbol table
// so that global declarations from one file are visible in all others.
// If a is non-nil, it is used for AST node allocations.
func NewParserWithSymbols(lex *lexer.Lexer, symbols *SymbolTable, fileName string, fileID uint16, a *arena.Arena) (*Parser, error) {
	if lex == nil {
		return nil, errors.New("no lexer provided to parser")
	}

	p := &Parser{
		lex:            lex,
		symbols:        symbols,
		ast:            ast.NewAST(a, fileID, lex.Len),
		Errs:           make([]error, 0, errorPreallocationSize),
		definedMethods: make(map[string]struct{}),
	}

	return p, nil
}

// NewScriptParser creates a parser in script mode for .cogs files.
// Script mode forbids package declarations and export keywords.
func NewScriptParser(lexer *lexer.Lexer) (*Parser, error) {
	return NewScriptParserWithSymbols(lexer, NewSymbolTable(), nil)
}

// NewScriptParserWithSymbols creates a script-mode parser with a shared symbol table.
// If a is non-nil, it is used for AST node allocations.
func NewScriptParserWithSymbols(lex *lexer.Lexer, symbols *SymbolTable, a *arena.Arena) (*Parser, error) {
	if lex == nil {
		return nil, errors.New("no lexer provided to parser")
	}

	p := &Parser{
		lex:     lex,
		symbols: symbols,
		// TODO: allow multi-file scripts?
		ast:            ast.NewAST(a, 0, lex.Len),
		Errs:           make([]error, 0, errorPreallocationSize),
		scriptMode:     true,
		definedMethods: make(map[string]struct{}),
	}

	return p, nil
}

func (p *Parser) Parse(ctx context.Context, fileName string) (*ast.AST, error) {
	p.FindGlobals(ctx)

	return p.ParseOnly(ctx, fileName)
}

// ParseOnly runs the full parse without calling FindGlobals first.
// Use this when FindGlobals has already been called on a shared symbol table
// across multiple files.
func (p *Parser) ParseOnly(ctx context.Context, fileName string) (*ast.AST, error) {
	// Reset position and errors for a clean parse.
	p.Errs = p.Errs[:0]
	p.lex.Reset()

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
		if p.lex.This().Type == tokens.Package {
			p.error(p.lex.This(), "package declaration not allowed in script files", "Parse")
		}

		// Synthesize package main.
		pkg = ast.New[ast.Package](p.ast)
		pkg.Token = tokens.Token{Type: tokens.Package, Literal: "package"}
		pkg.Identifier = &ast.Identifier{
			Token: tokens.Token{
				Type:    tokens.Identifier,
				Literal: "main",
			},
		}
	} else {
		if p.lex.This().Type != tokens.Package {
			p.error(p.lex.This(), "missing package declaration", "Parse")
		}

		pkg = p.parsePackage()
	}

	stmts := make([]ast.NodeIndex, 0, statementPreallocationSize)
	file := p.ast.NewFile(fileName, pkg, stmts, false)

	// Iterate tokens.
	for p.lex.This().Type != tokens.EOF && ctx.Err() == nil {
		t := p.lex.This()
		switch t.Type {
		case tokens.Comment:
			stmts = append(stmts, p.ast.NewComment(t))
			p.lex.Step() // consume comment
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
			ident := t.Literal

			node := p.parseStatement(ctx)
			if node != ast.ZeroNodeIndex {
				if ident == "main" {
					p.ast.Node(file).(*ast.File).ContainsMain = true
				}

				stmts = append(stmts, node)
			} else {
				p.synchronize(ctx)
			}
		case tokens.GoImport:
			node := p.parseGoImport(ctx)
			if node != ast.ZeroNodeIndex {
				stmts = append(stmts, node)
			} else {
				p.synchronize(ctx)
			}
		case tokens.Import:
			node := p.parseImport(ctx)
			if node != ast.ZeroNodeIndex {
				stmts = append(stmts, node)
			} else {
				p.synchronize(ctx)
			}
		default:
			p.error(t, "unexpected token", "Parse")
			p.synchronize(ctx)
		}

		// // Guard against infinite loops: if no progress was made, force advance.
		// if p.lex.This() == t {
		// 	p.lex.Step()
		// }

		// // Check for EOF again, in case it was reached during parsing.
		// if t.Type == tokens.EOF {
		// 	break tokenLoop
		// }
	}

	if ctx.Err() != nil {
		return p.ast, fmt.Errorf("parser error: %w", ctx.Err())
	}

	if err := errors.Join(p.Errs...); err != nil {
		return p.ast, fmt.Errorf("parser error:\n%w", errors.Join(p.Errs...))
	}

	p.ast.Node(file).(*ast.File).Statements = stmts

	return p.ast, nil
}

// synchronize advances tokens until it finds a token that can begin a new statement.
// This enables error recovery by skipping malformed input.
func (p *Parser) synchronize(ctx context.Context) {
	for p.lex.This().Type != tokens.EOF && ctx.Err() == nil {
		// Check next token, if it can start a statement, return to the main parse loop.
		switch p.lex.Peek(1).Type {
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
			p.lex.Step()
			return
		}

		p.lex.Step()
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

func (p *Parser) NodeString(i ast.NodeIndex) string {
	var out strings.Builder
	p.ast.Node(i).StringTo(&out, p.ast)
	return out.String()
}

func (p *Parser) ExprString(i ast.ExprIndex) string {
	var out strings.Builder
	p.ast.Expr(i).StringTo(&out, p.ast)
	return out.String()
}

func (p *Parser) typeExpr(i ast.ExprIndex) types.Expression {
	expr := p.ast.Expr(i)

	var out strings.Builder
	expr.StringTo(&out, p.ast)

	return types.Expression{
		Expr:   expr,
		String: out.String(),
	}
}
