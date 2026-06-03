package transpiler

import (
	"fmt"
	goast "go/ast"

	"github.com/samborkent/cog/internal/ast"
	"github.com/samborkent/cog/internal/types"
)

func (t *Transpiler) convertDeferExpr(expr ast.Expr, srcLine uint32) ([]goast.Stmt, error) {
	switch n := expr.(type) {
	case *ast.Call:
		goExpr, err := t.convertExpr(n)
		if err != nil {
			return nil, err
		}

		callExpr, ok := goExpr.(*goast.CallExpr)
		if !ok {
			return nil, fmt.Errorf("defer call expression is not a call")
		}

		lineComment := fmt.Sprintf("\n//line %s:%d", t.file.Name, srcLine)
		lineDecl := &goast.DeclStmt{Decl: t.commentDecl(lineComment)[0]}

		t.deferStack = append(t.deferStack, deferInfo{
			callStmts: []goast.Stmt{
				lineDecl,
				&goast.ExprStmt{X: callExpr},
			},
		})

		return nil, nil

	case *ast.ProcedureLiteral:
		body := t.Node(n.Body).(*ast.Block)

		stmts := make([]goast.Stmt, 0, len(body.Statements))

		prevInFunc := t.inFunc
		prevUsesDyn := t.usesDyn
		prevLoopDepth := t.loopDepth
		prevDeferStack := t.deferStack
		t.deferStack = nil
		t.loopDepth = 0
		t.usesDyn = false
		if procType, ok := n.ProcedureType.(*types.Procedure); ok {
			t.inFunc = procType.Function
		} else {
			t.inFunc = false
		}

		for _, s := range body.Statements {
			stmt, err := t.convertStmt(t.Node(s))
			if err != nil {
				return nil, err
			}

			stmts = append(stmts, stmt...)
		}

		t.injectDeferredBody(stmts)

		bodyUsesDyn := t.usesDyn
		t.usesDyn = prevUsesDyn || bodyUsesDyn
		t.inFunc = prevInFunc
		t.loopDepth = prevLoopDepth
		t.deferStack = prevDeferStack

		return stmts, nil

	default:
		return nil, fmt.Errorf("defer requires a procedure call or closure, got %T", n)
	}
}

func (t *Transpiler) injectDeferredBody(stmts []goast.Stmt) {
	if len(t.deferStack) == 0 {
		return
	}

	deferredStmts := t.buildReversedDeferredStmts()

	body := &goast.BlockStmt{List: append([]goast.Stmt(nil), stmts...)}

	injectBeforeReturns(body, deferredStmts)

	body.List = append(body.List, deferredStmts...)

	copy(stmts, body.List)
}

func (t *Transpiler) injectDeferred(block *goast.BlockStmt) {
	if len(t.deferStack) == 0 {
		return
	}

	deferredStmts := t.buildReversedDeferredStmts()

	injectBeforeReturns(block, deferredStmts)

	block.List = append(block.List, deferredStmts...)
}

func (t *Transpiler) buildReversedDeferredStmts() []goast.Stmt {
	total := 0
	for _, d := range t.deferStack {
		total += len(d.callStmts)
	}

	stmts := make([]goast.Stmt, 0, total)
	for i := len(t.deferStack) - 1; i >= 0; i-- {
		stmts = append(stmts, t.deferStack[i].callStmts...)
	}

	return stmts
}

func injectBeforeReturns(block *goast.BlockStmt, deferredStmts []goast.Stmt) {
	for i := 0; i < len(block.List); i++ {
		switch s := block.List[i].(type) {
		case *goast.ReturnStmt:
			preamble := make([]goast.Stmt, len(deferredStmts))
			copy(preamble, deferredStmts)
			block.List = append(block.List[:i], append(preamble, block.List[i:]...)...)
			i += len(preamble)

		case *goast.BranchStmt:
			preamble := make([]goast.Stmt, len(deferredStmts))
			copy(preamble, deferredStmts)
			block.List = append(block.List[:i], append(preamble, block.List[i:]...)...)
			i += len(preamble)

		case *goast.IfStmt:
			if s.Body != nil {
				injectBeforeReturns(s.Body, deferredStmts)
			}
			if s.Else != nil {
				if elseBlock, ok := s.Else.(*goast.BlockStmt); ok {
					injectBeforeReturns(elseBlock, deferredStmts)
				}
			}

		case *goast.ForStmt:
			if s.Body != nil {
				injectBeforeReturns(s.Body, deferredStmts)
			}

		case *goast.RangeStmt:
			if s.Body != nil {
				injectBeforeReturns(s.Body, deferredStmts)
			}

		case *goast.SwitchStmt:
			if s.Body != nil {
				for _, cc := range s.Body.List {
					if caseClause, ok := cc.(*goast.CaseClause); ok {
						injectBeforeReturnsInCase(caseClause, deferredStmts)
					}
				}
			}

		case *goast.LabeledStmt:
			if blockStmt, ok := s.Stmt.(*goast.BlockStmt); ok {
				injectBeforeReturns(blockStmt, deferredStmts)
			}
		}
	}
}

func injectBeforeReturnsInCase(caseClause *goast.CaseClause, deferredStmts []goast.Stmt) {
	for i := 0; i < len(caseClause.Body); i++ {
		switch s := caseClause.Body[i].(type) {
		case *goast.ReturnStmt:
			preamble := make([]goast.Stmt, len(deferredStmts))
			copy(preamble, deferredStmts)
			caseClause.Body = append(caseClause.Body[:i], append(preamble, caseClause.Body[i:]...)...)
			i += len(preamble)

		case *goast.IfStmt:
			if s.Body != nil {
				injectBeforeReturns(s.Body, deferredStmts)
			}
			if s.Else != nil {
				if elseBlock, ok := s.Else.(*goast.BlockStmt); ok {
					injectBeforeReturns(elseBlock, deferredStmts)
				}
			}
		}
	}
}