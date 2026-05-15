package parser

import (
	"context"
	"path"
	"strings"

	"github.com/samborkent/cog/internal/ast"
	"github.com/samborkent/cog/internal/tokens"
	"github.com/samborkent/cog/internal/types"
)

func (p *Parser) parseImport(ctx context.Context) ast.NodeIndex {
	importToken := p.lex.This()
	imports := make([]*ast.Identifier, 0, importPreallocationSize)

	p.lex.Step() // consume 'import'

	if p.lex.This().Type != tokens.LParen {
		p.error(p.lex.This(), "expected '(' after import", "parseImport")
		return ast.ZeroNodeIndex
	}

	p.lex.Step() // consume '('

	for p.lex.This().Type != tokens.EOF && ctx.Err() == nil {
		t := p.lex.This()
		if t.Type == tokens.RParen {
			break
		}

		if t.Type != tokens.StringLiteral {
			p.error(t, "found non-string token in import list: "+t.Literal, "parseImport")
			return ast.ZeroNodeIndex
		}

		importPath := t.Literal

		// Safety: disallow parent traversal and absolute paths.
		if strings.Contains(importPath, "..") || strings.HasPrefix(importPath, "/") {
			p.error(t, "import path must be a relative subdirectory path (no '..' or leading '/')", "parseImport")
			return ast.ZeroNodeIndex
		}

		// Package name is the last segment of the path.
		pkgName := path.Base(importPath)

		_, alreadyImported := p.symbols.ResolveCogImport(pkgName)
		if alreadyImported {
			// Already registered (e.g. during FindGlobals); just record in the AST node.
			ident := &ast.Identifier{
				Token:     t,
				ValueType: types.None,
			}
			imports = append(imports, ident)

			p.lex.Step()
			continue
		}

		ident := &ast.Identifier{
			Token:     t,
			ValueType: types.None,
		}

		imports = append(imports, ident)

		// Register the import in the symbol table.
		// Exports will be populated later by the driver (cmd/main.go)
		// after the imported package has been parsed.
		p.symbols.DefineCogImport(&CogImport{
			Path:    importPath,
			Name:    pkgName,
			Exports: make(map[string]Symbol),
		})

		p.lex.Step()
	}

	p.lex.Step() // consume ')'

	return p.ast.NewImport(importToken, imports)
}
