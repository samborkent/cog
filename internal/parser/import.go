package parser

import (
	"path"
	"strings"

	"github.com/samborkent/cog/internal/ast"
	"github.com/samborkent/cog/internal/tokens"
	"github.com/samborkent/cog/internal/types"
)

func (p *Parser) parseImport() *ast.Import {
	node := &ast.Import{
		Token:   p.this(),
		Imports: make([]*ast.Identifier, 0),
	}

	p.advance("parseImport import") // consume 'import'

	if p.this().Type != tokens.LParen {
		p.error(p.this(), "expected '(' after import", "parseImport")
		return nil
	}

	p.advance("parseImport (") // consume '('

	for ; p.this().Type != tokens.RParen && p.this().Type != tokens.EOF; p.advance("parseImport loop") {
		if p.this().Type != tokens.StringLiteral {
			p.error(p.this(), "found non-string token in import list: "+p.this().Literal, "parseImport")
			return nil
		}

		importPath := p.this().Literal

		// Safety: disallow parent traversal and absolute paths.
		if strings.Contains(importPath, "..") || strings.HasPrefix(importPath, "/") {
			p.error(p.this(), "import path must be a relative subdirectory path (no '..' or leading '/')", "parseImport")
			return nil
		}

		// Package name is the last segment of the path.
		pkgName := path.Base(importPath)

		_, alreadyImported := p.symbols.ResolveCogImport(pkgName)
		if alreadyImported {
			// Already registered (e.g. during FindGlobals); just record in the AST node.
			ident := &ast.Identifier{
				Token:     p.this(),
				Name:      importPath,
				ValueType: types.None,
			}
			node.Imports = append(node.Imports, ident)

			continue
		}

		ident := &ast.Identifier{
			Token:     p.this(),
			Name:      importPath,
			ValueType: types.None,
		}

		node.Imports = append(node.Imports, ident)

		// Register the import in the symbol table.
		// Exports will be populated later by the driver (cmd/main.go)
		// after the imported package has been parsed.
		p.symbols.DefineCogImport(&CogImport{
			Path:    importPath,
			Name:    pkgName,
			Exports: make(map[string]Symbol),
		})
	}

	p.advance("parseImport )") // consume ')'

	return node
}
