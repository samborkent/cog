# Ownership Model — Leftover Rules

Three rules from the ownership plan were not implemented because they depend on Cog language features that are not yet present in the grammar. This document explains each rule, what it depends on, and how to implement it once those features are added.

---

## Rule 12: Async closure consumes `var` mutable variables

**When the dependency is added**: `async` keyword support in the grammar and lexer.

**What it does**: When an `async` closure captures a `var` variable from the enclosing scope, the variable is consumed (permanently dead) because the closure outlives that scope. The async closure runs at some later point, so the captured variable cannot be used in the original scope after the async closure definition.

**Implementation**:

1. The `async` keyword token (`tokens.Async`) must be recognized before a procedure literal: `f := async { ... }`.
2. In `parseProcedureLiteral` (in `primary.go`), after parsing the closure body and before returning, check `p.lex.Peek(-1).Type == tokens.Async`.
3. Walk the closure's body expression tree using the existing `p.collectCapturedVars(exprIdx)` helper to find all `var` identifiers referenced inside.
4. Call `p.symbols.MarkConsumed(name)` for each captured variable.

The AST walker (`collectCapturedVars` → `walkExprForVarIdentifiers`) is already implemented in `internal/parser/ownership.go` and can be reused directly.

**Files affected**: `internal/parser/primary.go` (in `parseProcedureLiteral`)

---

## Rule 14: Async closure inside borrow-scope overrides borrow with consumption

**When the dependency is added**: Together with Rule 12 (async keyword).

**What it does**: If a borrow-scope (e.g., a for-loop or a block expression) contains an `async` closure that captures the borrowed variable, the borrow is upgraded to consumption. The captured variable is dead after the borrow-scope ends.

**Example**: 
```
items : var []int64 = @slice<int64>(3)
for item in items {   // items is borrowed
    f := async {      // items is now consumed, not borrowed
        @print(item)
    }
}
// items is dead here — the async closure captured it
```

**Implementation**:

1. The existing `UpgradeBorrowToConsume(name)` method on `SymbolTable` is already implemented in `internal/parser/ownership.go`:
   ```go
   func (s *SymbolTable) UpgradeBorrowToConsume(name string) {
       if count, ok := s.borrowCounts.Load(name); ok && count > 0 {
           s.borrowCounts.Store(name, 0)
       }
       s.ownership.Store(name, stateConsumed)
   }
   ```
2. In `parseProcedureLiteral`, when `async` is detected, walk the closure body to find captured `var` identifiers (same walker as Rule 12). For each one that is currently borrowed (borrow counter > 0 in the current scope), call `p.symbols.UpgradeBorrowToConsume(name)` instead of `MarkConsumed`.

**Files affected**: `internal/parser/primary.go` (in `parseProcedureLiteral`)

---

## Rule 25: Chained method calls borrow receiver for chain duration

**When the dependency is added**: When the grammar supports `.Method()` calls after an expression (not just after an identifier/selector).

**What it does**: A method chain like `items.sort().reverse()` should borrow `items` for the duration of the entire chain, releasing only after the final `.reverse()` call completes.

**Why it's not implementable now**: The parser's `primary()` function handles `.` only immediately after an identifier. After a method call returns from `primary()`, the calling expression parser does not handle a trailing `.` — so `items.sort().reverse()` is parsed as `items.sort()` followed by an unparsed `.reverse()` token. The grammar simply doesn't support chaining method calls on arbitrary expression results.

**Implementation** (once chaining is supported):

1. Inside the selector+method-call loop in `primary.go` (around line 277-369), track the first `var` receiver encountered.
2. On first method call: call `p.symbols.MarkBorrowed(chainRootVar)`.
3. When the dot-loop exits (no more `.`): call `p.symbols.ReleaseBorrowed(chainRootVar)`.

The borrow counter supports nesting: `items.sort().reverse()` would increment once (counter → 1) and decrement once (counter → 0). If the chain is inside a for-loop over `items`, the counter correctly reflects both borrows.

**Files affected**: `internal/parser/primary.go` (selector+method-call parsing block, around lines 277-369)

---

## Summary

| Rule | Depends on | Implementation effort |
|------|-----------|---------------------|
| 12 | `async` keyword in grammar/lexer | Low — walker already exists |
| 14 | Rule 12 (`async` keyword) | Low — `UpgradeBorrowToConsume` already implemented |
| 25 | Chained `.Method()` after expressions | Medium — requires grammar changes + borrow tracking |