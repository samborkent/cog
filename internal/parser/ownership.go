package parser

import (
	"fmt"

	"github.com/samborkent/cog/internal/ast"
	"github.com/samborkent/cog/internal/tokens"
	"github.com/samborkent/cog/internal/types"
	isync "github.com/samborkent/cog/internal/sync"
)

type ownershipState uint8

const (
	stateAlive    ownershipState = iota
	stateConsumed
	stateReserved
)

type branchConsumption struct {
	vars   map[string]struct{}
	fields map[string]map[string]struct{}
}

// applyCallOwnership applies Rules 7 & 8 to a call expression: func borrows args, proc consumes args.
// Returns false if an error was emitted.
func (p *Parser) applyCallOwnership(callToken tokens.Token, procType *types.Procedure, args []ast.ExprIndex) bool {
	for _, arg := range args {
		if argIdent, ok := p.ast.Expr(arg).(*ast.Identifier); ok &&
			argIdent.Qualifier == ast.QualifierVariable {
			if procType.Function {
				// Rule 7: func borrows.
				p.symbols.MarkBorrowed(argIdent.Token.Literal)
			} else {
				// Rule 8: proc consumes.
				if !p.symbols.IsAlive(argIdent.Token.Literal) {
					p.error(callToken, fmt.Sprintf("cannot use %s: variable is consumed", argIdent.Token.Literal), "applyCallOwnership")
					return false
				}
				p.symbols.MarkConsumed(argIdent.Token.Literal)
			}
		}
	}
	return true
}

// collectCapturedStructFields walks an AST expression tree and returns
// struct variable → field name pairs accessed on var structs (for defer field reservation).
func (p *Parser) collectCapturedStructFields(exprIdx ast.ExprIndex) map[string]map[string]struct{} {
	captured := make(map[string]map[string]struct{})
	p.walkExprForStructFieldAccess(exprIdx, captured)
	return captured
}

func (p *Parser) walkExprForStructFieldAccess(exprIdx ast.ExprIndex, captured map[string]map[string]struct{}) {
	e := p.ast.Expr(exprIdx)
	if e == nil {
		return
	}
	switch e := e.(type) {
	case *ast.Selector:
		if len(e.Fields) >= 2 && e.Fields[0] != nil &&
			e.Fields[0].Qualifier == ast.QualifierVariable {
			sv := e.Fields[0].Token.Literal
			fn := e.Fields[len(e.Fields)-1].Token.Literal
			if _, ok := p.symbols.Resolve(sv); ok {
				if captured[sv] == nil {
					captured[sv] = make(map[string]struct{})
				}
				captured[sv][fn] = struct{}{}
			}
		}
		for _, f := range e.Fields {
			if f != nil {
				p.walkExprForStructFieldAccess(p.ast.AddExpr(f), captured)
			}
		}
	case *ast.Call:
		for _, arg := range e.Arguments {
			p.walkExprForStructFieldAccess(arg, captured)
		}
	case *ast.Infix:
		p.walkExprForStructFieldAccess(e.Left, captured)
		p.walkExprForStructFieldAccess(e.Right, captured)
	case *ast.Prefix:
		p.walkExprForStructFieldAccess(e.Right, captured)
	case *ast.Suffix:
		p.walkExprForStructFieldAccess(e.Left, captured)
	case *ast.Grouped:
		p.walkExprForStructFieldAccess(e.Expr, captured)
	}
}

// collectCapturedVars walks an AST expression tree and returns all var identifiers
// that reference variables in the parser's scope chain.
func (p *Parser) collectCapturedVars(exprIdx ast.ExprIndex) map[string]struct{} {
	captured := make(map[string]struct{})
	p.walkExprForVarIdentifiers(exprIdx, captured)
	return captured
}

func (p *Parser) walkExprForVarIdentifiers(exprIdx ast.ExprIndex, captured map[string]struct{}) {
	e := p.ast.Expr(exprIdx)
	if e == nil {
		return
	}
	switch e := e.(type) {
	case *ast.Identifier:
		if e.Qualifier == ast.QualifierVariable {
			if _, ok := p.symbols.Resolve(e.Token.Literal); ok {
				captured[e.Token.Literal] = struct{}{}
			}
		}
	case *ast.Call:
		for _, arg := range e.Arguments {
			p.walkExprForVarIdentifiers(arg, captured)
		}
	case *ast.Selector:
		// Capture struct field references: if the root is a var variable,
		// the field access is a captured var resource that should be reserved.
		if len(e.Fields) > 0 && e.Fields[0] != nil &&
			e.Fields[0].Qualifier == ast.QualifierVariable {
			if _, ok := p.symbols.Resolve(e.Fields[0].Token.Literal); ok {
				captured[e.Fields[0].Token.Literal] = struct{}{}
			}
		}
		for _, f := range e.Fields {
			if f != nil {
				p.walkExprForVarIdentifiers(p.ast.AddExpr(f), captured)
			}
		}
	case *ast.Infix:
		p.walkExprForVarIdentifiers(e.Left, captured)
		p.walkExprForVarIdentifiers(e.Right, captured)
	case *ast.Prefix:
		p.walkExprForVarIdentifiers(e.Right, captured)
	case *ast.Suffix:
		if e != nil {
			p.walkExprForVarIdentifiers(p.ast.AddExpr(&ast.Identifier{Token: e.Operator}), captured)
		}
	case *ast.Grouped:
		p.walkExprForVarIdentifiers(e.Expr, captured)
	}
}

func (s *SymbolTable) owningScope(name string) *SymbolTable {
	if _, ok := s.table.Load(name); ok {
		return s
	}
	if s.Outer != nil {
		return s.Outer.owningScope(name)
	}
	return nil
}

func (s *SymbolTable) IsAlive(name string) bool {
	owning := s.owningScope(name)
	if owning == nil {
		return true
	}
	state, ok := owning.ownership.Load(name)
	return !ok || state == stateAlive || state == stateReserved
}

func (s *SymbolTable) IsFieldAlive(structVar, field string) bool {
	owning := s.owningScope(structVar)
	if owning == nil {
		return true
	}
	fields, ok := owning.fieldOwnership.Load(structVar)
	if !ok {
		return true
	}
	state, ok := fields.Load(field)
	return !ok || state == stateAlive || state == stateReserved
}

func (s *SymbolTable) IsFieldReserved(structVar, field string) bool {
	owning := s.owningScope(structVar)
	if owning == nil {
		return false
	}
	fields, ok := owning.fieldOwnership.Load(structVar)
	if !ok {
		return false
	}
	state, ok := fields.Load(field)
	return ok && state == stateReserved
}

func (s *SymbolTable) IsFullyAlive(name string) bool {
	owning := s.owningScope(name)
	if owning == nil {
		return true
	}
	fields, ok := owning.fieldOwnership.Load(name)
	if !ok {
		return true
	}
	for _, state := range fields.All() {
		if state == stateConsumed {
			return false
		}
	}
	return true
}

func (s *SymbolTable) MarkConsumed(name string) error {
	owning := s.owningScope(name)
	if owning == nil {
		return nil
	}
	if !owning.IsAlive(name) {
		return fmt.Errorf("cannot use %s: variable is consumed", name)
	}
	owning.ownership.Store(name, stateConsumed)
	return nil
}

func (s *SymbolTable) MarkFieldConsumed(structVar, field string) error {
	owning := s.owningScope(structVar)
	if owning == nil {
		return nil
	}
	if !owning.IsFieldAlive(structVar, field) {
		return fmt.Errorf("%s: field '%s' moved out here", structVar, field)
	}
	fields, ok := owning.fieldOwnership.Load(structVar)
	if !ok {
		fields = isync.NewMap[string, ownershipState](s.opts()...)
		owning.fieldOwnership.Store(structVar, fields)
	}
	fields.Store(field, stateConsumed)
	return nil
}

func (s *SymbolTable) ReviveField(structVar, field string) {
	owning := s.owningScope(structVar)
	if owning == nil {
		return
	}
	fields, ok := owning.fieldOwnership.Load(structVar)
	if ok {
		fields.Delete(field)
	}
}

func (s *SymbolTable) MarkBorrowed(name string) {
	count, _ := s.borrowCounts.Load(name)
	s.borrowCounts.Store(name, count+1)
}

func (s *SymbolTable) ReleaseBorrowed(name string) {
	count, ok := s.borrowCounts.Load(name)
	if ok && count > 0 {
		s.borrowCounts.Store(name, count-1)
	}
}

func (s *SymbolTable) IsBorrowed(name string) bool {
	count, ok := s.borrowCounts.Load(name)
	return ok && count > 0
}

func (s *SymbolTable) MarkReserved(name string) error {
	owning := s.owningScope(name)
	if owning == nil {
		return nil
	}
	state, ok := owning.ownership.Load(name)
	if ok && state == stateReserved {
		return fmt.Errorf("cannot defer %s: already reserved by a defer", name)
	}
	if ok && state == stateConsumed {
		// Already consumed — skipping reservation so the downstream consumption
		// error will surface naturally when code tries to use the consumed var.
		return nil
	}
	owning.ownership.Store(name, stateReserved)
	return nil
}

func (s *SymbolTable) MarkFieldReserved(structVar, field string) error {
	owning := s.owningScope(structVar)
	if owning == nil {
		return nil
	}
	fields, ok := owning.fieldOwnership.Load(structVar)
	if !ok {
		fields = isync.NewMap[string, ownershipState](s.opts()...)
		owning.fieldOwnership.Store(structVar, fields)
	}
	state, ok := fields.Load(field)
	if ok && state == stateConsumed {
		return fmt.Errorf("cannot defer %s: field '%s' is already consumed", structVar, field)
	}
	if ok && state == stateReserved {
		return fmt.Errorf("cannot defer %s: field '%s' already reserved by a defer", structVar, field)
	}
	fields.Store(field, stateReserved)
	return nil
}

func (s *SymbolTable) UpgradeBorrowToConsume(name string) {
	if count, ok := s.borrowCounts.Load(name); ok && count > 0 {
		s.borrowCounts.Store(name, 0)
	}
	s.ownership.Store(name, stateConsumed)
}

// snapshotOwnership captures the current ownership state for cross-branch merging.
func (s *SymbolTable) snapshotOwnership() (map[string]ownershipState, map[string]map[string]ownershipState) {
	own := make(map[string]ownershipState)
	for k, v := range s.ownership.All() {
		own[k] = v
	}
	fieldOwn := make(map[string]map[string]ownershipState)
	for sv, fields := range s.fieldOwnership.All() {
		fm := make(map[string]ownershipState)
		for f, st := range fields.All() {
			fm[f] = st
		}
		fieldOwn[sv] = fm
	}
	return own, fieldOwn
}

// restoreOwnership restores ownership state from a snapshot.
func (s *SymbolTable) restoreOwnership(own map[string]ownershipState, fieldOwn map[string]map[string]ownershipState) {
	s.ownership = isync.NewMap[string, ownershipState](s.opts()...)
	for k, v := range own {
		s.ownership.Store(k, v)
	}
	s.fieldOwnership = isync.NewMap[string, *isync.Map[string, ownershipState]](s.opts()...)
	for sv, fields := range fieldOwn {
		fm := isync.NewMap[string, ownershipState](s.opts()...)
		for f, st := range fields {
			fm.Store(f, st)
		}
		s.fieldOwnership.Store(sv, fm)
	}
}

// diffOwnership returns what was consumed since the snapshot.
func (s *SymbolTable) diffOwnership(prevOwn map[string]ownershipState, prevFieldOwn map[string]map[string]ownershipState) branchConsumption {
	bc := branchConsumption{
		vars:   make(map[string]struct{}),
		fields: make(map[string]map[string]struct{}),
	}
	for name, state := range s.ownership.All() {
		if state == stateConsumed {
			if prev, ok := prevOwn[name]; !ok || prev != stateConsumed {
				bc.vars[name] = struct{}{}
			}
		}
	}
	for sv, fields := range s.fieldOwnership.All() {
		for f, st := range fields.All() {
			if st == stateConsumed {
				prevFields, ok := prevFieldOwn[sv]
				if !ok || prevFields[f] != stateConsumed {
					if bc.fields[sv] == nil {
						bc.fields[sv] = make(map[string]struct{})
					}
					bc.fields[sv][f] = struct{}{}
				}
			}
		}
	}
	return bc
}

// unionConsumption combines branch consumptions into a single set.
func unionConsumption(branches []branchConsumption) (map[string]struct{}, map[string]map[string]struct{}) {
	vars := make(map[string]struct{})
	fields := make(map[string]map[string]struct{})
	for _, bc := range branches {
		for n := range bc.vars {
			vars[n] = struct{}{}
		}
		for sv, fs := range bc.fields {
			for f := range fs {
				if fields[sv] == nil {
					fields[sv] = make(map[string]struct{})
				}
				fields[sv][f] = struct{}{}
			}
		}
	}
	return vars, fields
}