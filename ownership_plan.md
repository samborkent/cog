# Ownership Model Implementation Plan

## Overview

Implement the Cog ownership model as described in `ownership_model.md` and tested in `ownership.cogs`. The model tracks `var` mutable variables through borrow/consume semantics during parsing (phase 1), then generates appropriate Go code in the transpiler (phase 2).

~~**Note**: Step 0b (block expressions) has been removed from the plan — Cog does not have block expressions. The `ast.BlockExpr` type and `parseBlockExpression` method were created and then deleted.~~

The plan has **two major phases**, each split into incremental steps that can be verified independently:

## Validation Policy

After every functionally complete step that can compile and run, the implementer **must** run these commands and fix any failures before proceeding to the next step:

```bash
GOEXPERIMENT=arenas rtk go vet ./...   # compilation check
GOEXPERIMENT=arenas rtk go test ./...  # all tests pass
rtk go run ./cmd/main.go -file <test-file>  # transpile a test program
```

- `task vet` runs `go vet` (compilation check)
- `task test` runs all tests
- `task run` runs the transpiler on a sample file to verify end-to-end
- Always use `rtk` prefix and `GOEXPERIMENT=arenas`

---

## Phase 1: Parser (Ownership Tracking)

### Goal

Add a borrow/consume/dead tracking system to the `SymbolTable` that is checked during parsing. Errors are emitted as parse errors (compile-time).

### Step 2: Ownership State Types (`internal/parser/ownership.go`)

Create new types parallel to the existing `checkState` pattern:

```go
type ownershipState uint8

const (
    stateAlive    ownershipState = iota // variable is live (default for var)
    stateConsumed                       // permanently consumed
    stateReserved                       // reserved by defer (rule 13)
)

type fieldState struct {
    state ownershipState
    owner string // struct variable name (for field tracking)
}
```

Note: `stateBorrowed` is intentionally absent from `ownershipState`. Borrows are tracked separately via per-scope reference-counted borrow counters (integers), not via the bitfield — this allows nested borrows (rule 15) where a single variable can be borrowed multiple times simultaneously.

Add three maps to `SymbolTable`:
- `ownership *isync.Map[string, ownershipState]` — top-level var consumed/reserved state (walks up scope chains)
- `fieldOwnership *isync.Map[string, *isync.Map[string, ownershipState]]` — per-field consumed/reserved state within structs
- `borrowCounts *isync.Map[string, int32]` — per-scope borrow counter (local only; does NOT walk up scope chains). Each enclosed scope starts with an empty borrow counter map so inner scopes can borrow independently of outer borrows.

**Files affected**: `internal/parser/symbol_table.go`, new file `internal/parser/ownership.go`

### Step 3: Core Ownership API on SymbolTable

Add methods to `SymbolTable`:

```go
// MarkConsumed marks a variable (or field) as permanently dead.
func (s *SymbolTable) MarkConsumed(name string) error
func (s *SymbolTable) MarkFieldConsumed(structVar, field string) error

// MarkBorrowed / ReleaseBorrowed — reference-counted borrow tracking.
func (s *SymbolTable) MarkBorrowed(name string)
func (s *SymbolTable) ReleaseBorrowed(name string)

// MarkReserved marks a variable as reserved by defer (rule 13).
func (s *SymbolTable) MarkReserved(name string) error
func (s *SymbolTable) MarkFieldReserved(structVar, field string) error

// IsAlive checks if a variable is safe to use.
func (s *SymbolTable) IsAlive(name string) bool
func (s *SymbolTable) IsFieldAlive(structVar, field string) bool

// IsFullyAlive checks that all fields of a struct are alive.
func (s *SymbolTable) IsFullyAlive(name string) bool
```

Ownership state methods (`IsAlive`, `IsFieldAlive`, `MarkConsumed`, `MarkFieldConsumed`, `MarkReserved`, `MarkFieldReserved`, `IsFullyAlive`) walk up scope chains like the existing `IsValueChecked`/`IsErrorChecked`.

Borrow counter methods (`MarkBorrowed`, `ReleaseBorrowed`, `IsBorrowed`) operate only on the current scope, without walking up. When a borrow-scoped variable is referenced, the deepest scope's counter is incremented; when the borrow ends, the same scope's counter is decremented.

**Files affected**: `internal/parser/ownership.go`, `internal/parser/symbol_table.go`

### Step 4: Borrow/Consume Rules in Parser Statement Processing

Modify `parseStatement` and related functions to enforce ownership rules:

#### Rule 2: Taking `&` reference of `var` mutable value consumes the variable

In expression parsing for the prefix `&` operator (in `internal/parser/ebnf_parser.go`, the `unary` function):
- When the operand is a `var` variable identifier, call `MarkConsumed` on it
- The resulting `&` reference is immutable — the variable's data is now accessible only through the reference
- Checking `IsAlive` afterward should report the variable as dead
- Implementation: insert the `MarkConsumed` call after `p.unary(ctx, exprType)` returns and before the `p.ast.NewPrefix(...)` call, using the returned expression to detect if it's a `var` identifier

#### Rule 3: Dereferencing `var &` mutable reference consumes the reference

In expression parsing for the prefix `*` operator:
- When the operand is a `var &` mutable reference identifier, call `MarkConsumed` on it
- The resulting dereferenced value is a copy (for value types) or an owned view

#### Rule 4: `var` to `var` assignment consumes source

In `parseAssignment` (in `assignment.go`), when the RHS is an identifier with `QualifierVariable`:
- Call `MarkConsumed` on the source — ownership transfers uniformly for **all** types
- Error if the source is already dead
- **Even non-pointer-like types must be consumed**: rule 4 says any `var` to `var` assignment transfers ownership, regardless of whether the type is pointer-like. Non-pointer-like types (primitives, structs of primitives) are cheap to copy, but ownership still transfers — the source is dead after assignment. This prevents aliasing bugs uniformly — distinguishing pointer-like vs. primitive at this level would create a confusing gap where some `var`→`var` assignments consume and others don't.
- At the transpiler level, `var → var` assignment of primitives generates a plain Go assignment (the consumption is purely a compile-time tracking artifact), while pointer-like `var → var` generates the same plain assignment (ownership semantics are compile-time, not runtime copies).

#### Rule 5: Immutable to `var` deep copies

When assigning an immutable value of pointer-like type to a `var`:
- No ownership transfer, but mark that transpiler needs `cog.Copy`

#### Rule 7: `func` call borrows args

When a `func` call is detected (via `Procedure.Function == true`):
- For each `var` argument, call `MarkBorrowed` before parsing the body
- Call `ReleaseBorrowed` after the call expression is fully parsed

#### Rule 8: `proc` call consumes args

When a `proc` call is detected:
- For each argument that is a `var` identifier, call `MarkConsumed`

**Parameter qualifier alignment**: When a `proc` has a `var` parameter, the parameter inside the proc body must be defined with `QualifierVariable`, not `QualifierImmutable`. Currently `primary.go:549` hardcodes `QualifierImmutable` for all parameters. Fix: after parsing the procedure type (where parameter qualifiers are part of the type definition — the `proc` type carries a `Mutable` flag per parameter), use that flag to set the correct qualifier when defining the parameter symbol.

The parameter type (`types.Parameter`) needs a `Mutable bool` field. Set it during parameter parsing in `parseProcedureType`:

The parameter loop at `type.go:723-786` currently starts each parameter by expecting `tokens.Identifier`. The `var` keyword would appear before the type, after the colon: `proc(x : var T)`. The loop must be extended:
```go
for p.lex.This().Type != tokens.EOF && ctx.Err() == nil {
    tok := p.lex.This()
    if tok.Type == tokens.RParen {
        break
    }
    if tok.Type != tokens.Identifier {
        p.error(tok, "expected parameter identifier", "parseParameters")
        return nil
    }
    param := &types.Parameter{
        Name: tok.Literal,
        Mutable: false,                                  // <-- default
    }
    p.lex.Step() // consume identifier

    if p.lex.This().Type == tokens.Question {
        param.Optional = true
        haveOptional = true
        p.lex.Step() // consume ?
    }

    if p.lex.This().Type != tokens.Colon {
        p.error(p.lex.This(), "expected ':' after input parameter identifier", "parseParameters")
        return nil
    }
    p.lex.Step() // consume :

    // Check for mutable qualifier before the type.
    if p.lex.This().Type == tokens.Variable {             // <-- new
        param.Mutable = true                              // <-- new
        p.lex.Step() // consume var                       // <-- new
    }

    paramType := p.parseCombinedType(ctx, false, false)
    ...
}
```

For `func` type checking: after the parameter loop, if `procType.Function == true` and any `param.Mutable == true`, emit a compile error (`"func cannot have mutable parameters"`).

For `proc`: proc parameters may be mutable. The `func` rejection check prevents `var` usage in `func` types.

When defining parameter symbols in `primary.go:543-552`, use `QualifierVariable` for mutable params and `QualifierImmutable` for immutable params:
```go
qualifier := ast.QualifierImmutable
if param.Mutable {
    qualifier = ast.QualifierVariable
}
ident := &ast.Identifier{
    ...
    Qualifier: qualifier,
}
```

#### Rule 10: Return transfers ownership

In the `parseReturn` handler (`statement.go:428-462`):
- If the returned expression is a `var` struct variable, call `p.symbols.IsFullyAlive(ident.Name)` to verify no fields have been moved out. If any field is dead, emit a compile error (`"cannot move %s: field '%s' is already moved out"`).
- If the returned expression is a `var` identifier of pointer-like type, call `MarkConsumed` on it — ownership transfers to the caller.
- If the returned expression is a `var` struct with all fields alive, call `MarkConsumed` on the entire struct — ownership transfers as a unit.
- **Rationale for `IsFullyAlive`**: A struct with a moved-out field cannot be meaningfully returned because the returned value would have a dead field. The check mirrors the whole-struct pass-to-proc check (Rule 19) — both are whole-value transfer operations.

#### Rule 11: Block expression borrows

~~**Block expressions (`{ stmts; expr }`) were planned but REMOVED — Cog does not have block expressions.** The plan originally specified `ast.BlockExpr` and `parseBlockExpression` as Step 0b, but these were incorrect and have been deleted.~~

If block expressions are added to Cog in the future, the ownership model would:
- At block entry: `MarkBorrowed` on all `var` variables referenced inside
- At block exit: `ReleaseBorrowed` on each
- The borrow scope is the enclosing scope's current `borrowCounts` map, not the block's inner scope

#### Rule 12: Async closure consumes `var` mutable variables

In `parseProcedureLiteral` when the closure has `async` annotation:
- Identify `var` variables referenced inside the closure body by walking the AST
- Call `MarkConsumed` on each — the closure outlives the enclosing scope
- Consumed variables cannot be used after the async closure definition
- Requires a helper to walk the closure body's expression tree and collect all `ast.Identifier` nodes with `QualifierVariable` that resolve to local scope variables

**AST walker helper**: The Cog AST is a flat `[]Node` array indexed by `NodeIndex`. A generic walker must understand each node type's children. The helper narrows the focus to `var` variables:

```go
// collectCapturedVars walks an AST node tree and returns all var identifiers
// that reference variables in the parser's scope chain.
func (p *Parser) collectCapturedVars(nodeIdx ast.NodeIndex) map[string]struct{} {
    captured := make(map[string]struct{})
    p.walkForVarIdentifiers(nodeIdx, captured)
    return captured
}

func (p *Parser) walkForVarIdentifiers(nodeIdx ast.NodeIndex, captured map[string]struct{}) {
    n := p.ast.Node(nodeIdx)
    switch n := n.(type) {
    case *ast.Block:
        for _, stmt := range n.Statements {
            p.walkForVarIdentifiers(stmt, captured)
        }
    case *ast.ExpressionStatement:
        p.walkExprForVarIdentifiers(n.Expr, captured)
    case *ast.Assignment:
        p.walkExprForVarIdentifiers(n.Value, captured)
    case *ast.Return:
        p.walkExprForVarIdentifiers(n.Expr, captured)
    case *ast.IfStatement:
        p.walkExprForVarIdentifiers(n.Condition, captured)
        if n.Consequence != nil {
            p.walkForVarIdentifiers(*n.Consequence, captured)
        }
        if n.Alternative != nil {
            p.walkForVarIdentifiers(*n.Alternative, captured)
        }
    case *ast.ForStatement:
        if n.Range != nil {
            p.walkExprForVarIdentifiers(*n.Range, captured)
        }
        p.walkForVarIdentifiers(n.Body, captured)
    // ... extend for Switch, Match, Defer, Branch, etc.
    }
}

func (p *Parser) walkExprForVarIdentifiers(exprIdx ast.ExprIndex, captured map[string]struct{}) {
    e := p.ast.Expr(exprIdx)
    switch e := e.(type) {
    case *ast.Identifier:
        if e.Qualifier == ast.QualifierVariable {
            if _, ok := p.symbols.Resolve(e.Token.Literal); ok {
                captured[e.Token.Literal] = struct{}{}
            }
        }
    case *ast.Call:
        for _, arg := range e.Args {
            p.walkExprForVarIdentifiers(arg, captured)
        }
    case *ast.Selector:
        for _, f := range e.Fields {
            p.walkExprForVarIdentifiers(p.ast.AddExpr(f), captured)
        }
    case *ast.Infix:
        p.walkExprForVarIdentifiers(e.Left, captured)
        p.walkExprForVarIdentifiers(e.Right, captured)
    case *ast.Prefix:
        p.walkExprForVarIdentifiers(e.Right, captured)
    case *ast.Suffix:
        p.walkExprForVarIdentifiers(p.ast.Expr(e.Left), captured)
    // ... extend for all expression types
    }
}
```

**Integration into `parseProcedureLiteral`**: After parsing the closure body and before returning, if the procedure literal is async:

```go
if procType.Function == false && p.lex.Peek(-1).Type == tokens.Async {
    // Walk the body to find captured var variables.
    captured := p.collectCapturedVars(bodyIdx)
    for name := range captured {
        p.symbols.MarkConsumed(name)
    }
}
```

Note: This runs after the body has been fully parsed, ensuring all identifier references are already registered.

#### Rule 13: Defer reserves

In the `defer` case in `parseStatement`:
- After parsing the defer expression, identify captured `var` variables
- Call `MarkReserved` on each
- **Forward-propagating reservation**: Mark the variable as reserved in the symbol table immediately. Subsequent consumption of the reserved variable (assignment, move, pass to proc) in the same scope checks the reservation and emits a compile error. The reservation itself does not prevent borrowing (reads are still safe). This avoids a post-parse pass — the check happens naturally at each consumption point.
- If a variable was already consumed before the defer, the `MarkReserved` call itself fails with "cannot defer %s: already consumed"
- If a variable is already reserved by an earlier defer (later in source order, first-to-execute at scope exit), `MarkReserved` fails with "cannot defer %s: already reserved by a defer"
- For field-level reservation: `MarkFieldReserved` applies the same logic per-field

#### Rule 14: Async closure inside borrow-scope overrides borrow with consumption

Before deciding borrow vs. consume for a block expression (rule 11) or for-loop (rule 15):
- Scan all nested statements for `async` closures that capture `var` outer variables
- If any such async capture is found, the outer variable is *consumed* (not borrowed)
- This means scanning the statement tree before applying the borrow, or performing the scan during expression parsing and upgrading borrows to consumes retroactively

Implementation approach: during the block/for entry, tentatively mark the variable as borrowed. If an async closure is encountered later that captures it, upgrade the entry to consumed:

```go
// Upgrade borrow to consume (called when async closure captures a borrowed var):
func (s *SymbolTable) UpgradeBorrowToConsume(name string) {
    // 1. Clear borrow counter for this variable in the current scope.
    if count, ok := s.borrowCounts.Load(name); ok && count > 0 {
        s.borrowCounts.Store(name, 0)
    }
    
    // 2. Mark as consumed in the owning scope (walks up).
    s.MarkConsumed(name)
}
```

Key invariants:
- The borrow counter in the current scope is **decremented to zero** during the upgrade. This prevents a subsequent `ReleaseBorrowed` from underflowing or incorrectly releasing the now-consumed variable.
- Any code between block entry and the async closure that relied on the borrow is still valid (borrow semantics are a subset of consume — the variable was alive until the consume point).
- After the upgrade, all subsequent uses of the variable report "cannot use: variable is consumed".
- The upgrade is applied inline during expression parsing: when `parseProcedureLiteral` encounters `async`, it walks the body's expression tree to identify captured `var` identifiers. For each one that is currently borrowed (not just borrowed, but any var reference from an enclosing scope), call `UpgradeBorrowToConsume`.

#### Rule 15: For-loop borrows iterable

In `parseForStatement`:
- Before parsing the loop body, `MarkBorrowed` on the range expression's variable (**in the enclosing scope**, not the body's scope — the borrow counter lives one level above the body)
- After the loop body, `ReleaseBorrowed`
- Handle nesting with a counter (the borrow counter in the current scope)
- Before finalizing borrow vs. consume, scan the loop body for async closures capturing the iterable (see Rule 14)

**`break`/`continue` interaction**: These statements do not affect borrow tracking. `MarkBorrowed` and `ReleaseBorrowed` bracket the *parse* of the loop body, not its runtime execution. `parseBlockStatement` returns the body node (potentially containing `Branch` nodes for break/continue), and then `ReleaseBorrowed` runs immediately in the parser. The borrow counter is a compile-time artifact — `break`/`continue` inside the body are already parsed statements in the returned Block; the borrow release is unconditional after `parseBlockStatement` returns.

#### Rule 16: Consumption is permanent across branches

After `parseIfStatement` or `parseMatch` or `parseSwitch`:
- Any variable consumed in ANY branch → mark consumed for the rest of the scope
- Track per-branch consumption and union across branches
- For partial moves (rule 19), apply the union per-field: if `c.data` is consumed in any branch, `c.data` is dead after the conditional; `c.label` remains alive

**Cross-branch ownership propagation mechanism**: `parseBlockStatement` creates an enclosed scope internally and restores `p.symbols` to its parent before returning. Therefore, the block's inner scope is lost unless captured. The implementation must save a reference to the child scope before it is restored:

```go
// Pattern for capturing branch scope (applied in parseIfStatement, parseMatch, parseSwitch):

// Before calling parseBlockStatement, save the current scope.
scopeBeforeBranch := p.symbols

consequence := p.parseBlockStatement(ctx)
// p.symbols is now restored to scopeBeforeBranch

// The inner scope created by parseBlockStatement is scopeBeforeBranch's child.
// Walk scopeBeforeBranch (now current) for ownership state set by MarkConsumed
// calls that happened through scope-walking (those target the enclosing scope).
// NOTE: parseBlockStatement creates p.symbols = NewEnclosedSymbolTable(p.symbols)
// at entry, then restores p.symbols = p.symbols.Outer at exit. After exit,
// p.symbols == scopeBeforeBranch. Any MarkConsumed() calls in the branch's
// inner scope that walked up to find the original var would have operated on
// the *original declaring scope*, not the branch's inner scope.
//
// CORRECTION: MarkConsumed walks up the scope chain to find the variable's
// definition, then stores ownershipState in the scope where the variable is
// defined (the owning scope). So after parseBlockStatement restores scope,
// consumed state is already correctly recorded in the owning scope.
//
// The key insight: ownership changes made by MarkConsumed/MarkFieldConsumed
// during branch parsing are applied to the scope chain by walking up — they
// end up in the right place automatically. The cross-branch union problem is:
// a var consumed in branch A should be dead even if branch B only borrows.
// This is handled by rule 16's conservative approach: if any branch consumed,
// the var is dead after the conditional. BUT consumption from branch A's scope
// does NOT automatically propagate because branch B's scope doesn't see it.
//
// Solution: after ALL branches are parsed (not per-branch), check if the
// variable was consumed in at least one branch. If so, mark it consumed.
// This requires tracking which vars were consumed across branches.

// Save consumed sets per branch.
type branchConsumption struct {
    vars    map[string]struct{}
    fields  map[string]map[string]struct{} // structVar → {fieldName}
}

// After each branch's scope is restored, capture the branch's consumption:
func (p *Parser) captureBranchConsumption(saved *SymbolTable) branchConsumption {
    bc := branchConsumption{
        vars:   make(map[string]struct{}),
        fields: make(map[string]map[string]struct{}),
    }
    if saved.ownership != nil {
        for name, state := range saved.ownership.All() {
            if state == stateConsumed {
                bc.vars[name] = struct{}{}
            }
        }
    }
    if saved.fieldOwnership != nil {
        for structVar, fields := range saved.fieldOwnership.All() {
            for fieldName, fieldState := range fields.All() {
                if fieldState == stateConsumed {
                    if bc.fields[structVar] == nil {
                        bc.fields[structVar] = make(map[string]struct{})
                    }
                    bc.fields[structVar][fieldName] = struct{}{}
                }
            }
        }
    }
    return bc
}
```

After all branches, union the consumption sets:

```go
// After all branches parsed (consequence + alternative, or all match arms):
for name := range branchConsumptions[0].vars {
    // Propagate consumption to outer scope.
    p.symbols.MarkConsumed(name)
}
// Similarly for fields, propagating if consumed in any branch:
for structVar, fields := range branchConsumptions[0].fields {
    for fieldName := range fields {
        // Check if this field was consumed in any branch.
        consumed := false
        for _, bc := range branchConsumptions[1:] {
            if _, ok := bc.fields[structVar][fieldName]; ok {
                consumed = true
                break
            }
        }
        if consumed {
            p.symbols.MarkFieldConsumed(structVar, fieldName)
        }
    }
}
```

**Alternate simpler approach**: After `parseBlockStatement` restores the scope, call `MarkConsumed` for each variable that was consumed in the branch. Since `MarkConsumed` walks up and operates on the owning scope, the inner scope state is irrelevant — the owning scope's ownership map already reflects the consumption. The issue is that a subsequent branch's borrow doesn't override a prior branch's consumption in the owning scope. The solution: before entering each subsequent branch, save and restore the owning scope's ownership state so each branch starts with a clean slate. After all branches, merge (union) the consumption states.

**Clean-slate approach**: Before parsing each branch, snapshot the owning scope's ownership state. After the branch, diff the state. After all branches, apply the union:

```go
// Before parsing a branch:
snapshot := p.snapshotOwnership()

// After the branch:
delta := p.diffOwnership(snapshot)
branchConsumptions = append(branchConsumptions, delta)

// Then restore the snapshot before the next branch:
p.restoreOwnership(snapshot)

// After all branches, union:
for _, consumed := range unionConsumptions(branchConsumptions) {
    p.symbols.MarkConsumed(consumed)
}
```

This clean-slate approach is preferred: it correctly handles the cross-branch union without double-marking or requiring branch-specific tracking. The `snapshotOwnership`, `diffOwnership`, `restoreOwnership` helpers operate on the ownership maps that walk up to the owning scope.

**Implementation placement**:
1. `parseIfStatement` (`if_statement.go`): after line 68 (`consequence := p.parseBlockStatement(ctx)`) and after line 105 (`alternative = p.parseBlockStatement(ctx)`)
2. `parseMatch` (`match.go`): after line 117-133 (each case arm's scope restored at line 133)
3. `parseSwitch` (`switch.go`): `parseBoolSwitch` at lines 71-82 (each case body) and `parseIdentSwitch` at lines 177-188

#### Rule 19: Partial moves for struct fields

In `parseAssignment` when a field selector is used as RHS:
- Mark only that specific field as consumed (`MarkFieldConsumed`)
- The struct variable and other fields remain alive
- For whole-struct operations (pass to proc, return, pass to func), check `IsFullyAlive`
- Re-assigning a moved-out field (`c.data = @slice<int64>(5)`) makes that field alive again. The re-assignment is tracked by removing the field's consumed state. After re-assignment, `IsFullyAlive` returns true

#### Rule 22: Index expressions borrow

In expression parsing for index expressions:
- If LHS is a `var`, the container is borrowed for the read duration
- `&items[i]` extends the borrow for the reference's lifetime — the reference's last-use analysis determines when the borrow is released. For simplicity in v1, the borrow is released at the end of the enclosing statement or scope (conservative)

#### Rule 24: Map/set insertion consumes key/value

In the map literal/set literal handling or key-value assignment parsing:
- Keys/values that are `var` identifiers are consumed

**Pre-condition: Map keys must satisfy `comparable`**: Go map keys must be comparable (no slices, no maps, no procs, and no structs/arrays containing them). The Cog type system already has `types.IsComparable()` at `is.go:68-114` and the `comparable` constraint at `generics.go:66-77`. The parser must validate map key types against `IsComparable`:

In `parseType` (`type.go:204-241`), after parsing the key type for a `map<K, V>`:
```go
if !types.IsComparable(keyType) {
    p.error(keyToken, fmt.Sprintf("map key type %q is not comparable", keyType))
    return nil
}
```

The same check applies to the `@map` builtin (`parseBuiltinMap` at `builtins.go:99-166`), after resolving the key type argument and before constructing the `types.Map`.

The same check also applies to `Set` types: a `Set` transpiles to `map[T]struct{}` in Go, so `T` must also be comparable. Validate in `parseType` at the `Set` case (`type.go:242-264`) and in `parseBuiltinSet`.

**`ownership.cogs` correction**: The test case `main_map_key_var` originally used `map[[]int64]utf8` — `[]int64` is not comparable (`types.IsComparable` returns false), so this is invalid Cog. Updated to use `map[utf8]var []int64` which is valid (utf8 is comparable, and the value being pointer-like only affects copy semantics).

#### Rule 25: Chained method calls borrow receiver for chain duration

In `primary.go:267-365` (selector + method call parsing), the parser chains `.Method()` calls via the `for p.lex.This().Type == tokens.Dot` loop. When the loop exits (no more `.` or method calls), the chain is complete. Implementation:

```go
// Track chain start for borrow/release.
var chainRootVar string // set when first .Method() encountered on a var identifier
var chainBorrowed bool

// Inside the dot-loop, when a method call is detected (line 331):
if p.match(p.lex.This(), tokens.LParen, tokens.LT) {
    if chainRootVar == "" && symbol.Identifier.Qualifier == ast.QualifierVariable {
        // First method call in chain on a var variable — borrow it.
        chainRootVar = symbol.Identifier.Token.Literal
        p.symbols.MarkBorrowed(chainRootVar)
        chainBorrowed = true
    }
    // ... parse call args and return ...
}

// After the dot-loop ends (no more method calls):
if chainBorrowed {
    p.symbols.ReleaseBorrowed(chainRootVar)
}
```

The borrow counter supports nesting: `items.sort().reverse()` calls `MarkBorrowed` once (counter → 1) and `ReleaseBorrowed` once (counter → 0). If the chain itself is inside a borrow-scope (e.g., inside a for-loop over `items`), the borrow counter correctly reflects both borrows.

**Files affected**: `internal/parser/statement.go`, `internal/parser/assignment.go`, `internal/parser/for_statement.go`, `internal/parser/if_statement.go`, `internal/parser/match.go`, `internal/parser/switch.go`, `internal/parser/ebnf_parser.go`, `internal/parser/primary.go`, `internal/parser/type.go`, `internal/parser/builtins.go`

### Step 5: Func Restriction Checks

In `parseProcedureLiteral` and declaration parsing:

- **Rule 6**: `func` cannot have `var` parameters, cannot contain `async`, cannot be `async` itself
- **Rule 9**: `proc` can be `async`; `func` cannot
- Check: when parsing `Function: true` procedures, reject `QualifierVariable` on parameters, reject `async` keyword usage

**Files affected**: `internal/parser/declaration.go`, `internal/parser/parser.go`

### Step 6: Comparable Constraint Validation + Error Messages

**Comparable validation** is a pre-condition for map key and set element types (enforced during type parsing, before any ownership tracking runs). Implemented in `internal/parser/type.go` and `internal/parser/builtins.go`:
- In `parseType` at the `map<K, V>` case: after parsing `keyType`, call `types.IsComparable(keyType)`. If false, emit error `"map key type %q is not comparable"`.
- In `parseType` at the `Set` case: after parsing element type, call `types.IsComparable`. If false, emit error `"set element type %q is not comparable"`.
- In `parseBuiltinMap` (`builtins.go`): same check on `typArgs[0]`.
- In `parseBuiltinSet` (`builtins.go`): same check on `typArgs[0]`.

**Error messages** are defined inline at each enforcement point using the existing `p.error()` mechanism.

Define ownership-specific error messages in an error table:

| Violation | Error format (first %s = var name, second %s = detail) |
|-----------|-------------------------------------------------------|
| Use of consumed variable | `"cannot use %s: variable is consumed"` |
| Consumed twice | `"cannot use %s: variable is consumed"` |
| Borrowed while mutating | `"cannot mutate %s: variable is borrowed"` |
| Move-out while borrowed | `"cannot move %s: %s is borrowed"` |
| Partial move then whole-struct pass | `"cannot move %s: field '%s' is already moved out"` |
| Partial move then whole-struct borrow | `"cannot borrow %s: field '%s' is moved out"` |
| Use of dead field | `"%s: field '%s' moved out here"` |
| Taking `&` of consumed variable | `"cannot take reference of %s: variable is consumed"` |
| Deref of consumed mutable reference | `"cannot dereference %s: variable is consumed"` |
| Defer on already-consumed | `"cannot defer %s: %s is already consumed"` |
| Defer on already-consumed field | `"cannot defer %s: field '%s' is already consumed"` |
| Defer on reserved variable | `"cannot defer %s: %s is already reserved by a defer"` |
| Defer on reserved field | `"cannot defer %s: field '%s' already reserved by a defer"` |
| Reassign after field reserved by defer | `"cannot reassign %s: field '%s' reserved by defer"` |
| func with var param | `"func cannot have mutable parameters"` |
| func with async | `"func cannot contain async calls"` |
| async func declaration | `"func cannot be async"` |
| async closure captures borrowed variable | `"cannot capture %s in async closure: variable is borrowed"` |
| Map key not comparable | `"map key type %q is not comparable"` |
| Set element not comparable | `"set element type %q is not comparable"` |

### Step 10: Parser Ownership Tests

Add test cases covering all rules from `ownership.cogs` to existing parser test files:

- New test file: `internal/parser/ownership_test.go`
- Comprehensive test matrix covering ALL 25 rules with both positive (should compile) and negative (should error) cases:

| # | Test name | Rule | What it tests | Expected |
|---|-----------|------|---------------|----------|
| 1 | `main_revive` | 19 | Partial move + re-assignment revives struct | ✅ No error |
| 2 | `main_partial_borrow` | 22 | Narrower field borrow, other field still mutable | ✅ No error |
| 3 | `main_enum_borrow` | 20 | Immutable match arm borrows enum, alive after | ✅ No error |
| 4 | `main_enum_consume` | 20 | All-immutable match — no consumption | ✅ No error |
| 5 | `main_chain` | 25 | Chained method call, receiver alive after | ✅ No error |
| 6 | `main_defer_ok` | 13 | Defer with no subsequent consumption | ✅ No error |
| 7 | `main_cond_partial` | 16 | Conditional partial move, other field alive after | ✅ No error |
| 8 | `main_return_partial` | 10,19 | Return after re-assign revives struct | ✅ No error |
| 9 | `main_func_borrow_after` | 7 | Func arg borrowed, caller can use after call | ✅ No error |
| 10 | `main_dyn_copy` | 18 | Dyn read on pointer-like type (deep copy) | ✅ No error |
| 11 | `main_for_borrow` | 15 | For-loop iterable borrowed, alive after loop | ✅ No error |
| ~~12~~ | ~~`main_block_borrow`~~ | ~~11~~ | ~~Block expression borrows var, alive after~~ | ~~REMOVED~~ |
| 13 | `main_defer_reassigned` | 13 | Defer followed by field reassignment | ❌ Should error |
| 14 | `main_defer_double` | 13 | Two defers on same variable | ❌ Should error |
| 15 | `main_defer_already_dead` | 13 | Defer on already-consumed variable | ❌ Should error |
| 16 | `main_ref_consumes` | 2 | Taking & of var consumes the variable | ❌ Should error |
| 17 | `main_deref_consumes` | 3 | Dereferencing var & consumes the reference | ❌ Should error |
| 18 | `main_var_to_var` | 4 | var to var assignment consumes source | ❌ Should error |
| 19 | `main_map_key_var` | 24 | Map insertion consumes key and val | ❌ Should error |
| 20 | `main_proc_consume` | 8 | Proc arg consumed, dead after call | ❌ Should error |
| 21 | `main_switch_consume` | 16 | Variable consumed in switch case dead after | ❌ Should error |
| 22 | `main_for_consume` | 16 | Variable consumed in for body dead next iteration | ❌ Should error |
| 23 | `main_func_var_param` | 6 | func with var parameter | ❌ Should error |
| 24 | `main_async_func` | 9 | async func declaration | ❌ Should error |
| 25 | `main_map_key_not_comparable` | 24 | Map with slice key (not comparable) | ❌ Should error |
| 26 | `main_set_element_not_comparable` | 24 | Set with slice element (not comparable) | ❌ Should error |
| 27 | `main_enum_consume_var` | 20 | Match arm with var binding consumes enum | ❌ Should error |
| 28 | `main_defer_field_after` | 13 | Defer on field, field reassigned after | ❌ Should error |
| 29 | `main_return_dead_field` | 10,19 | Return struct with moved-out field | ❌ Should error |
| 30 | `main_borrow_mutate` | 7 | Mutate var while func call borrows it | ❌ Should error |
| 31 | `main_async_capture_borrowed` | 14 | Async closure captures borrowed variable | ❌ Should error |

- Use the existing `parse` and `parseShouldError` helpers from `internal/parser/helpers_test.go`
- Each test case should be a self-contained Cog function body that exercises one ownership rule
- The `ownership.cogs` file already provides scenario bodies for tests 1-8, 13-15, 19, 22, 25, 29

---

## Phase 2: Transpiler (Code Generation)

### Goal

Generate correct Go code that enforces ownership at runtime:
1. Deep copy when immutable → `var` assignment (rule 5)
2. No-op borrowing (already safe in Go, only affects parser tracking)
3. Dynamic variable deep copy on read/write (rule 18)

### Step 1 (Phase 2, Step 1/4): PointerLike Detection

Note: This step has no parser or transpiler dependencies and can be implemented at any time, including before all parser ownership steps.

Add a function to determine if a type is pointer-like (needs deep copy on `immutable → var` assignment):

```go
// In internal/types/is.go — add alongside the existing IsPointer
func IsPointerLike(t Type) bool
```

This extends the existing `types.IsPointer` (`is.go:117-119`) which returns true for ReferenceKind, SliceKind, SetKind, MapKind, ProcedureKind. `IsPointerLike` differs:

Logic:
- Primitives (int, float, bool, string): false
- Slices, Maps, Sets: true (same as `IsPointer`)
- ProcedureKind: true (closures need deep copy; same as `IsPointer`)
- Structs: `types.Struct.IsComplex` — true if any field transitively contains a pointer-like type. If `IsComplex` is false, the struct is a plain value type.
- Arrays: true if element type is pointer-like
- `&` references: **false** (immutable, copied by pointer-copy; differs from `IsPointer`)
- Options: true if inner type is pointer-like
- Results: true if value or error type is pointer-like
- Either: true if left or right type is pointer-like
- Alias resolution: recurse through `Underlying()`
- Generic instantiation: evaluate at instantiation time
- Interface/any: false (interface header copy)

Implementation can reuse `IsPointer` as base, then add cases for Struct and Reference that diverge:

### Step 7 (Phase 2, Step 2/4): Deep Copy Generation in Transpiler

In `internal/transpiler/declaration.go`, modify the `convertDecl` method:

**For `immutable → var` assignments** (rule 5):
- When the RHS is immutable and the LHS has `QualifierVariable`
- And the type `IsPointerLike`:
  - Wrap the RHS expression in `cog.Copy(...)` call
  - Example: `b : var []int64 = a` → `var b = cog.Copy(a)`

**For `var → var` assignments** (rule 5):
- Already handled by parser (consumes source)
- The Go code is a simple assignment (ownership semantics are compile-time)

### Step 8 (Phase 2, Step 3/4): Dynamic Variable Deep Copy (Rule 18)

In `internal/transpiler/component/context.go`, modify `DynRead`:

- For pointer-like types, wrap the read in `cog.Copy(...)`:
  ```go
  func DynRead(fieldName string, isPointerLike bool) goast.Expr {
      read := &goast.SelectorExpr{
          X:   &goast.Ident{Name: dynVar},
          Sel: &goast.Ident{Name: fieldName},
      }
      if isPointerLike {
          return &goast.CallExpr{
              Fun: &goast.SelectorExpr{
                  X:   &goast.Ident{Name: "cog"},
                  Sel: &goast.Ident{Name: "Copy"},
              },
              Args: []goast.Expr{read},
          }
      }
      return read
  }
  ```

Similarly for `DynWrite` — wrap the value in `cog.Copy(...)` for pointer-like types.

The transpiler needs to know whether each dyn field's type is pointer-like. Add a `dynPointerLike map[string]bool` field to `Transpiler`, computed during `predeclareGlobals` by calling `types.IsPointerLike` on each `ident.ValueType` from the `dynamics` map. `DynRead` and `DynWrite` callers (in `expression.go` and `statement.go`) look up this map when building the read/write expression for a dyn variable.

### Step 9 (Phase 2, Step 4/4): Map/Set Insertion Copy (Rule 24)

In `internal/transpiler/expression.go`, in the `MapLiteral` and `SetLiteral` cases:

- For each key/value pair where the source expression is a `var` pointer-like type, wrap the converted Go expression in `cog.Copy(...)`. Since `MapLiteral` builds `goast.KeyValueExpr` entries, the wrapping happens per-element before insertion
- For `SetLiteral`, wrap each value expression similarly
- For infix index assignment (`m[key] = val`), handled in the transpiler's statement conversion: when the assigned value is a `var` pointer-like type, wrap in `cog.Copy(...)`
- The parser already marks the source consumed; the transpiler ensures the Go copy preserves independence

**Implementation detail**: The transpiler currently has no runtime notion of `var` vs immutable — the qualifier is only on the AST `Identifier`. The key/value expressions need to be inspected for their qualifier during transpilation, or the parser must annotate the AST nodes to signal "this expression needs a copy." The recommended approach: add boolean flags to `ast.KeyValue` indicating which expressions came from `var` sources:

```go
// In internal/ast/ast.go or internal/ast/expression.go:
type KeyValue struct {
    Key           ExprIndex
    Value         ExprIndex
    KeyConsumed   bool  // set when key is a var pointer-like type (needs deep copy)
    ValueConsumed bool  // set when value is a var pointer-like type (needs deep copy)
}
```

Set during parser ownership tracking in `primary.go`'s map literal parsing (line 479-482):
```go
pairs = append(pairs, ast.KeyValue{
    Key:           key,
    Value:         val,
    KeyConsumed:   p.isVarPointerLike(key),
    ValueConsumed: p.isVarPointerLike(val),
})
```

The transpiler then checks these flags when building `goast.KeyValueExpr` or `goast.CallExpr` entries:
```go
if pair.KeyConsumed {
    keyExpr = wrapInCogCopy(keyExpr)
}
if pair.ValueConsumed {
    valExpr = wrapInCogCopy(valExpr)
}
```

The `SetLiteral` case can carry a parallel `Consumed []bool` field, one per element, or reuse the `KeyValue` approach with `ValueConsumed` only.

### Step 11: Transpiler Ownership Tests

Add tests that verify generated Go output for ownership scenarios:

- `var := immutable` with pointer-like type → generates `cog.Copy(...)` wrapper
- `var := immutable` with primitive type → generates plain assignment (no `cog.Copy`)
- `var := var` with pointer-like type → plain assignment (ownership transfers at compile time)
- Dynamic read with pointer-like type → wraps in `cog.Copy(...)`
- Dynamic read with primitive type → plain read (no `cog.Copy`)
- Dynamic write with pointer-like type → wraps value in `cog.Copy(...)`
- Map/set insertion with `var` pointer-like key/value → wraps in `cog.Copy(...)` before insertion

Use the existing `transpile`/`mustContain`/`mustNotContain` helpers from `internal/transpiler/helpers_test.go`.

---

## Implementation Order

The recommended order is restructured to resolve **parser pre-requisites before ownership tracking** and to **interleave parser and transpiler phases** for incremental verification:

| Step | Description | Dependencies | Testable | Risk |
|------|-------------|--------------|----------|------|
| Step | Description | Status |
|------|-------------|--------|
| 0a | `types.Parameter.Mutable` field + `var` keyword in parameter grammar | ✅ DONE |
| ~~0b~~ | ~~BlockExpr~~ | ~~REMOVED~~ |
| 1 | `types.IsPointerLike()` in `internal/types/is.go` | ✅ DONE |
| 2 | Ownership state types (`ownershipState`, maps on `SymbolTable`) | ✅ DONE |
| 3 | Core ownership API (`MarkConsumed`, `MarkBorrowed`, `IsAlive`, etc.) | ✅ DONE |
| 4 | Borrow/consume rules in parser (Rules 2-16, 19) | ✅ DONE |
| 5 | Func restriction checks (Rules 6, 9: `func` vs `proc` enforcement) | ✅ DONE |
| 6 | Comparable constraint validation (map key, set element) | ✅ DONE |
| 7 | Deep copy generation in transpiler (Rule 5: `cog.Copy` on immutable→var) | ✅ DONE |
| 8 | Dynamic variable deep copy in transpiler (Rule 18) | ✅ DONE |
| 9 | Map/set insertion deep copy in transpiler (Rule 24) | ✅ DONE |
| 10 | Parser ownership tests | ✅ DONE |
| 11 | Transpiler ownership tests (7 test cases) | ✅ DONE |
| **20** | Match on enum borrows vs consumes | ❌ REMOVED — does not apply to Cog's match/enum design; match subjects are never consumed by the match expression |
| **22** | Index expressions borrow | ✅ DONE |
| **25** | Chained method calls borrow | ❌ SKIPPED — Cog parser does not support `.Method()` after a call expression (chained calls not in grammar) |

**Step 4 dependency updated**: was `Steps 0a, 0b, 3`, now `Steps 0a, 3` (0b removed). Step 4 risk remains **High** because it encompasses 13 rules with complex interactions: async closure inside borrow-scope (rule 14), defer forward-checking (rule 13), conditional branch union propagation (rule 16), switch coverage (rule 16 extension), cross-branch propagation mechanics, and chained method borrows (rule 25).

**Step 9 risk** is **Medium** because it requires AST annotation plumbing between parser and transpiler (adding `KeyConsumed`/`ValueConsumed` flags to `ast.KeyValue`).

**Step 10 risk** is **Medium** (not Low) because each test case requires a complete multi-declaration Cog program with type declarations, enum definitions, and proc bodies. Many tests involve type declarations (`Container`, `Result`, `Point`) that must be repeated per test or placed in shared test scaffolding. The 31-test matrix covers all 25 rules with both positive and negative cases, requiring careful assertion setup.

**Critical path**: Step 0a is a parser pre-requisite that ownership tracking (Step 4) depends on. Step 0b was removed — Cog does not have block expressions.

## Key Design Decisions

1. **No Go runtime changes needed**: The existing `cog.Copy()` function already handles deep copying. The ownership model is a *compile-time* analysis — at the Go level, we just need to:
   - Generate `cog.Copy()` for `immutable → var` transitions (rule 5)
   - Generate deep copies in `dyn` read/write (rule 18)
   - Generate `cog.Copy()` for map/set insertion of `var` values (rule 24)
   - The parser enforces all other rules via error messages

2. **Borrow counters are per-scope, ownership state walks up**: Borrow counters (`borrowCounts` map) exist only in the current `SymbolTable` scope and are not inherited from outer scopes — each borrow-scope (block, for-loop) tracks its own borrows. Ownership state (consumed, reserved) walks up the scope chain like the existing `checked` map, since consumption is permanent across scopes.

3. **Reference-counted borrow**: For nested borrows (rule 15: for-loop nesting), use an `int32` counter instead of a boolean flag. `MarkBorrowed` increments; `ReleaseBorrowed` decrements. A counter of 0 means not borrowed; > 0 means borrowed. This supports nesting within the same scope and across nested scopes.

4. **Partial-move via per-field state**: Track each struct field independently in a nested map (`fieldOwnership`), matching the existing `fields` map pattern in `SymbolTable`. Re-assignment of a moved-out field clears its consumed state (the field becomes alive again).

5. **`func` vs `proc` distinction**: Already available via `types.Procedure.Function`. The transpiler already uses `t.inFunc` for some restrictions. Parser-level func restrictions (rule 6) are enforced during declaration/parameter parsing as compile errors, not in the transpiler.

6. **Forward-propagating defer reservation**: Defer checks work by marking reserved state at parse time, then checking at each subsequent consumption point. This avoids a post-parse pass over the statement list.

7. **Annotation for transpiler copy signals**: The parser must annotate AST nodes (e.g., add a flag to `ast.MapPair` or to `ast.Identifier` usage) to signal to the transpiler which expressions need `cog.Copy()` wrapping. The transpiler has no runtime notion of `var` vs immutable — the qualifier is on the declaration identifier, not on usage expressions.
