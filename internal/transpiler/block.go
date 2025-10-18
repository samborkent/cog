package transpiler

import (
	"fmt"
	goast "go/ast"
	gotoken "go/token"
	"strconv"

	"github.com/samborkent/cog/internal/ast"
)

var ifLabelCounter = 0

var noOp = &goast.EmptyStmt{}

func (t *Transpiler) convertIfBlock(node *ast.Block) (*goast.BlockStmt, *goast.LabeledStmt, error) {
	block := &goast.BlockStmt{
		List: make([]goast.Stmt, 0, len(node.Statements)),
	}

	var label *goast.LabeledStmt

	for i, stmt := range node.Statements {
		breakExpr, ok := stmt.(*ast.Break)
		if ok {
			if breakExpr.Label != nil {
				block.List = append(block.List, &goast.BranchStmt{
					Tok:   gotoken.GOTO,
					Label: breakExpr.Label.Go(),
				})

				continue
			} else {
				label = &goast.LabeledStmt{
					Label: &goast.Ident{Name: "BreakIf" + strconv.Itoa(ifLabelCounter)},
					Stmt:  noOp,
				}

				ifLabelCounter++
			}

			block.List = append(block.List, &goast.BranchStmt{
				Tok:   gotoken.GOTO,
				Label: label.Label,
			})

			continue
		}

		goStmts, err := t.convertStmt(stmt)
		if err != nil {
			return nil, nil, fmt.Errorf("converting statement %d in block: %w", i, err)
		}

		block.List = append(block.List, goStmts...)
	}

	return block, label, nil
}
