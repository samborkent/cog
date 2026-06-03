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

// findDeadField returns the name of the first moved-out field on a struct variable,
// or empty string if the variable is not a struct or all fields are alive.
func (p *Parser) findDeadField(varName string) string {
	if sym, ok := p.symbols.Resolve(varName); ok && sym.Identifier.ValueType.Kind() == types.StructKind {
		if st, ok := sym.Identifier.ValueType.Underlying().(*types.Struct); ok {
			for _, f := range st.Fields {
				if !p.symbols.IsFieldAlive(varName, f.Name) {
					return f.Name
				}
			}
		}
	}
	return ""
}

// applyCallOwnership applies Rules 7 & 8 to a call expression: func borrows args, proc consumes args.
// Returns false if an error was emitted.
func (p *Parser) applyCallOwnership(callToken tokens.Token, procType *types.Procedure, args []ast.ExprIndex) bool {
	for _, arg := range args {
		if argIdent, ok := p.ast.Expr(arg).(*ast.Identifier); ok &&
			argIdent.Qualifier == ast.QualifierVariable {
			if procType.Function {
				// Rule 7: func borrows.
				// Rule 19: Cannot borrow a struct with moved-out fields.
				if !p.symbols.IsFullyAlive(argIdent.Token.Literal) {
					deadField := p.findDeadField(argIdent.Token.Literal)
					p.error(callToken, fmt.Sprintf("cannot borrow %s: field '%s' is moved out", argIdent.Token.Literal, deadField), "applyCallOwnership")
					return false
				}
				p.symbols.MarkBorrowed(argIdent.Token.Literal)
			} else {
				// Rule 8: proc consumes.
				if !p.symbols.IsAlive(argIdent.Token.Literal) {
					p.error(callToken, fmt.Sprintf("cannot use %s: variable is consumed", argIdent.Token.Literal), "applyCallOwnership")
					return false
				}
				if err := p.symbols.MarkConsumed(argIdent.Token.Literal); err != nil {
					p.error(callToken, err.Error(), "applyCallOwnership")
					return false
				}
			}
		}
	}
	return true
}

// collectCapturedStructFields walks an AST expression tree and returns
// struct variable → field name pairs accessed on var structs (for defer field reservation).
func (p *Parser) collectCapturedStructFields(exprIdx ast.ExprIndex) map[string]map[string]struct{} {
	captured := make(map[string]map[string]struct{})
	p.walkExprVars(exprIdx, nil, captured)
	return captured
}

// collectCapturedVars walks an AST expression tree and returns all var identifiers
// that reference variables in the parser's scope chain.
func (p *Parser) collectCapturedVars(exprIdx ast.ExprIndex) map[string]struct{} {
	captured := make(map[string]struct{})
	p.walkExprVars(exprIdx, captured, nil)
	return captured
}

// walkExprVars walks an expression tree and collects:
//   - vars (map[string]struct{}): var identifiers resolved in scope
//   - fields (map[string]map[string]struct{}): structVar → fieldName for struct field accesses
//
// Either collector may be nil to skip that collection.
func (p *Parser) walkExprVars(exprIdx ast.ExprIndex, vars map[string]struct{}, fields map[string]map[string]struct{}) {
	e := p.ast.Expr(exprIdx)
	if e == nil {
		return
	}
	switch e := e.(type) {
	case *ast.Identifier:
		if vars != nil && e.Qualifier == ast.QualifierVariable {
			if _, ok := p.symbols.Resolve(e.Token.Literal); ok {
				vars[e.Token.Literal] = struct{}{}
			}
		}
	case *ast.Selector:
		if len(e.Fields) >= 2 && e.Fields[0] != nil &&
			e.Fields[0].Qualifier == ast.QualifierVariable {
			sv := e.Fields[0].Token.Literal
			fn := e.Fields[len(e.Fields)-1].Token.Literal
			if _, ok := p.symbols.Resolve(sv); ok {
				if vars != nil {
					vars[sv] = struct{}{}
				}
				if fields != nil {
					if fields[sv] == nil {
						fields[sv] = make(map[string]struct{})
					}
					fields[sv][fn] = struct{}{}
				}
			}
		}
		for _, f := range e.Fields {
			if f != nil {
				p.walkExprVars(p.ast.AddExpr(f), vars, fields)
			}
		}
	case *ast.Call:
		for _, arg := range e.Arguments {
			p.walkExprVars(arg, vars, fields)
		}
	case *ast.Infix:
		p.walkExprVars(e.Left, vars, fields)
		p.walkExprVars(e.Right, vars, fields)
	case *ast.Prefix:
		p.walkExprVars(e.Right, vars, fields)
	case *ast.Suffix:
		if e != nil {
			p.walkExprVars(p.ast.AddExpr(&ast.Identifier{Token: e.Operator}), vars, fields)
		}
	case *ast.Grouped:
		p.walkExprVars(e.Expr, vars, fields)
	case *ast.Index:
		p.walkExprVars(e.Expr, vars, fields)
		p.walkExprVars(e.Index, vars, fields)
	case *ast.Builtin:
		for _, arg := range e.Arguments {
			p.walkExprVars(arg, vars, fields)
		}
	case *ast.GoCallExpression:
		for _, arg := range e.Arguments {
			p.walkExprVars(arg, vars, fields)
		}
	case *ast.ArrayLiteral:
		for _, v := range e.Values {
			p.walkExprVars(v, vars, fields)
		}
	case *ast.SliceLiteral:
		for _, v := range e.Values {
			p.walkExprVars(v, vars, fields)
		}
	case *ast.MapLiteral:
		for _, pair := range e.Pairs {
			p.walkExprVars(pair.Key, vars, fields)
			p.walkExprVars(pair.Value, vars, fields)
		}
	case *ast.SetLiteral:
		for _, v := range e.Values {
			p.walkExprVars(v, vars, fields)
		}
	case *ast.StructLiteral:
		for _, fv := range e.Values {
			p.walkExprVars(fv.Value, vars, fields)
		}
	case *ast.TupleLiteral:
		for _, v := range e.Values {
			p.walkExprVars(v, vars, fields)
		}
	case *ast.EitherLiteral:
		p.walkExprVars(e.Value, vars, fields)
	case *ast.ResultLiteral:
		p.walkExprVars(e.Value, vars, fields)
	case *ast.ProcedureLiteral:
		body := p.ast.Node(e.Body)
		if block, ok := body.(*ast.Block); ok {
			for _, s := range block.Statements {
				p.walkNodeVars(s, vars, fields)
			}
		}
	}
}

// walkNodeVars walks a statement node tree and collects variable references and
// struct field accesses into the provided collectors (either may be nil).
func (p *Parser) walkNodeVars(nodeIdx ast.NodeIndex, vars map[string]struct{}, fields map[string]map[string]struct{}) {
	n := p.ast.Node(nodeIdx)
	if n == nil {
		return
	}
	switch n := n.(type) {
	case *ast.Block:
		for _, s := range n.Statements {
			p.walkNodeVars(s, vars, fields)
		}
	case *ast.ExpressionStatement:
		p.walkExprVars(n.Expr, vars, fields)
	case *ast.Assignment:
		p.walkExprVars(n.Expr, vars, fields)
	case *ast.Return:
		p.walkExprVars(n.Value, vars, fields)
	case *ast.IfStatement:
		p.walkExprVars(n.Condition, vars, fields)
		if n.Consequence != nil {
			p.walkNodeVars(p.ast.AddNode(n.Consequence), vars, fields)
		}
		if n.Alternative != nil {
			p.walkNodeVars(p.ast.AddNode(n.Alternative), vars, fields)
		}
	case *ast.ForStatement:
		if n.Range != ast.ZeroExprIndex {
			p.walkExprVars(n.Range, vars, fields)
		}
		p.walkNodeVars(p.ast.AddNode(n.Loop), vars, fields)
	case *ast.Switch:
		if n.Identifier != nil {
			p.walkExprVars(p.ast.AddExpr(n.Identifier), vars, fields)
		}
		for _, c := range n.Cases {
			if c.Condition != ast.ZeroExprIndex {
				p.walkExprVars(c.Condition, vars, fields)
			}
			for _, s := range c.Body {
				p.walkNodeVars(s, vars, fields)
			}
		}
		if n.Default != nil {
			for _, s := range n.Default.Body {
				p.walkNodeVars(s, vars, fields)
			}
		}
	case *ast.Match:
		p.walkExprVars(n.Subject, vars, fields)
		for _, c := range n.Cases {
			for _, s := range c.Body {
				p.walkNodeVars(s, vars, fields)
			}
		}
		if n.Default != nil {
			for _, s := range n.Default.Body {
				p.walkNodeVars(s, vars, fields)
			}
		}
	case *ast.Defer:
		p.walkExprVars(n.Expr, vars, fields)
	case *ast.Branch:
	case *ast.Comment:
	case *ast.Declaration:
		if n.Assignment != nil {
			p.walkExprVars(n.Assignment.Expr, vars, fields)
		}
	case *ast.Type:
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
	if !ok {
		return true
	}
	switch state {
	case stateAlive, stateReserved:
		return true
	default:
		return false
	}
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
	if !ok {
		return true
	}
	switch state {
	case stateAlive, stateReserved:
		return true
	default:
		return false
	}
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
	if state, ok := owning.ownership.Load(name); ok && state == stateReserved {
		return fmt.Errorf("cannot use %s: reserved by defer", name)
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
	if owning.IsFieldReserved(structVar, field) {
		return fmt.Errorf("cannot consume %s: field '%s' reserved by defer", structVar, field)
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
	s.MarkConsumed(name)
}

// scopeSnapshot holds ownership state for a single scope level.
type scopeSnapshot struct {
	ownership      map[string]ownershipState
	fieldOwnership map[string]map[string]ownershipState
}

// snapshotScopeChain captures ownership state from the current scope and
// all enclosing scopes, keyed by scope chain depth (0 = innermost).
func (s *SymbolTable) snapshotScopeChain() []scopeSnapshot {
	var chain []scopeSnapshot
	for cur := s; cur != nil; cur = cur.Outer {
		own := make(map[string]ownershipState)
		for k, v := range cur.ownership.All() {
			own[k] = v
		}
		fieldOwn := make(map[string]map[string]ownershipState)
		for sv, fields := range cur.fieldOwnership.All() {
			fm := make(map[string]ownershipState)
			for f, st := range fields.All() {
				fm[f] = st
			}
			fieldOwn[sv] = fm
		}
		chain = append(chain, scopeSnapshot{
			ownership:      own,
			fieldOwnership: fieldOwn,
		})
	}
	return chain
}

// restoreScopeChain restores ownership state from a scope chain snapshot.
// The snapshot chain must match the current scope chain length.
func (s *SymbolTable) restoreScopeChain(snapshots []scopeSnapshot) {
	idx := 0
	for cur := s; cur != nil && idx < len(snapshots); cur = cur.Outer {
		snap := snapshots[idx]
		cur.ownership = isync.NewMap[string, ownershipState](cur.opts()...)
		for k, v := range snap.ownership {
			cur.ownership.Store(k, v)
		}
		cur.fieldOwnership = isync.NewMap[string, *isync.Map[string, ownershipState]](cur.opts()...)
		for sv, fields := range snap.fieldOwnership {
			fm := isync.NewMap[string, ownershipState](cur.opts()...)
			for f, st := range fields {
				fm.Store(f, st)
			}
			cur.fieldOwnership.Store(sv, fm)
		}
		idx++
	}
}

// diffScopeChain returns what was consumed since the snapshot, examining
// every scope level in the chain.
func (s *SymbolTable) diffScopeChain(snapshots []scopeSnapshot) branchConsumption {
	bc := branchConsumption{
		vars:   make(map[string]struct{}),
		fields: make(map[string]map[string]struct{}),
	}
	idx := 0
	for cur := s; cur != nil && idx < len(snapshots); cur = cur.Outer {
		snap := snapshots[idx]
		for name, state := range cur.ownership.All() {
			if state == stateConsumed {
				if prev, ok := snap.ownership[name]; !ok || prev != stateConsumed {
					bc.vars[name] = struct{}{}
				}
			}
		}
		for sv, fields := range cur.fieldOwnership.All() {
			for f, st := range fields.All() {
				if st == stateConsumed {
					prevFields, ok := snap.fieldOwnership[sv]
					if !ok || prevFields[f] != stateConsumed {
						if bc.fields[sv] == nil {
							bc.fields[sv] = make(map[string]struct{})
						}
						bc.fields[sv][f] = struct{}{}
					}
				}
			}
		}
		idx++
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

// lookupSnapState checks whether a variable had a given ownership state
// at snapshot time, scanning the full scope chain.
func lookupSnapState(snap []scopeSnapshot, name string, state ownershipState) bool {
	for _, s := range snap {
		if st, ok := s.ownership[name]; ok && st == state {
			return true
		}
	}
	return false
}