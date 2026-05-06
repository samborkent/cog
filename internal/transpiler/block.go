package transpiler

import (
	"fmt"
	goast "go/ast"
	gotoken "go/token"
	"strconv"

	"github.com/samborkent/cog/internal/ast"
	"github.com/samborkent/cog/internal/tokens"
	"github.com/samborkent/cog/internal/transpiler/component"
)

var noOp = &goast.EmptyStmt{}

func (t *Transpiler) convertIfBlock(node *ast.Block) (*goast.BlockStmt, *goast.LabeledStmt, error) {
	block := &goast.BlockStmt{
		List: make([]goast.Stmt, 0, len(node.Statements)),
	}

	var label *goast.LabeledStmt

	if len(node.Statements) > 0 {
		// Enter if block scope.
		t.symbols = NewEnclosedSymbolTable(t.symbols)
	}

	for i, stmt := range node.Statements {
		breakExpr, ok := t.Node(stmt).(*ast.Branch)
		if ok && breakExpr.Token.Type == tokens.Break {
			if breakExpr.Label != nil {
				block.List = append(block.List, &goast.BranchStmt{
					Tok:   gotoken.GOTO,
					Label: component.Ident(breakExpr.Label),
				})

				continue
			} else {
				label = &goast.LabeledStmt{
					Label: &goast.Ident{Name: "BreakIf" + strconv.FormatUint(uint64(t.ifLabelCounter), 10)},
					Stmt:  noOp,
				}

				t.ifLabelCounter++
			}

			block.List = append(block.List, &goast.BranchStmt{
				Tok:   gotoken.GOTO,
				Label: label.Label,
			})

			continue
		}

		goStmts, err := t.convertStmt(t.Node(stmt))
		if err != nil {
			return nil, nil, fmt.Errorf("converting statement %d in block: %w", i, err)
		}

		block.List = append(block.List, goStmts...)
	}

	if len(node.Statements) > 0 {
		// Leave if block scope.
		t.symbols = t.symbols.Outer
	}

	return block, label, nil
}

func (t *Transpiler) convertForBlock(node *ast.Block) (*goast.BlockStmt, error) {
	block := &goast.BlockStmt{
		List: make([]goast.Stmt, 0, len(node.Statements)),
	}

	if len(node.Statements) > 0 {
		// Enter for block scope.
		t.symbols = NewEnclosedSymbolTable(t.symbols)
	}

	for i, stmt := range node.Statements {
		goStmts, err := t.convertStmt(t.Node(stmt))
		if err != nil {
			return nil, fmt.Errorf("converting statement %d in block: %w", i, err)
		}

		block.List = append(block.List, goStmts...)
	}

	if len(node.Statements) > 0 {
		// Leave for block scope.
		t.symbols = t.symbols.Outer
	}

	return block, nil
}
