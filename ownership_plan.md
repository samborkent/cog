# Ownership Model Implementation Plan

## Overview

Implement the Cog ownership model as described in `ownership_model.md` and tested in `ownership.cogs`. The model tracks `var` mutable variables through borrow/consume semantics during parsing (phase 1), then generates appropriate Go code in the transpiler (phase 2).

The plan has **two major phases**, each split into incremental steps that can be verified independently:

---

## Phase 1: Parser (Ownership Tracking)

### Goal

Add a borrow/consume/dead tracking system to the `SymbolTable` that is checked during parsing. Errors are emitted as parse errors (compile-time).

### Step 1: Ownership State Types (`internal/parser/ownership.go`)

Create new types parallel to the existing `checkState` pattern:

```go
type ownershipState uint8

const (
    stateAlive     ownershipState = 0  // variable is live (default for var)
    stateConsumed  ownershipState = 1 << iota // permanently consumed
    stateReserved                            // reserved by defer (rule 13)
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

### Step 2: Core Ownership API on SymbolTable

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

### Step 3: Borrow/Consume Rules in Parser Statement Processing

Modify `parseStatement` and related functions to enforce ownership rules:

#### Rule 2: Taking `&` reference of `var` mutable value consumes the variable

In expression parsing for the prefix `&` operator (in `internal/parser/expression.go`):
- When the operand is a `var` variable identifier, call `MarkConsumed` on it
- The resulting `&` reference is immutable — the variable's data is now accessible only through the reference
- Checking `IsAlive` afterward should report the variable as dead

#### Rule 3: Dereferencing `var &` mutable reference consumes the reference

In expression parsing for the prefix `*` operator:
- When the operand is a `var &` mutable reference identifier, call `MarkConsumed` on it
- The resulting dereferenced value is a copy (for value types) or an owned view

#### Rule 4: `var` to `var` assignment consumes source

In `parseAssignment` (in `assignment.go`), when the RHS is an identifier with `QualifierVariable` and pointer-like type:
- Call `MarkConsumed` on the source
- Error if the source is already dead

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

#### Rule 10: Return transfers ownership

In the `parseReturn` handler:
- The returned expression's `var` ownership transfers to caller → mark consumed

#### Rule 11: Block expression borrows

Block expressions are currently not first-class. When block expressions borrow:
- At block entry, `MarkBorrowed` on all `var` variables referenced inside
- At block exit, `ReleaseBorrowed`

#### Rule 12: Async closure consumes `var` mutable variables

In `parseProcedureLiteral` when the closure has `async` annotation:
- Identify `var` variables referenced inside the closure body
- Call `MarkConsumed` on each — the closure outlives the enclosing scope
- Consumed variables cannot be used after the async closure definition

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
- Implementation approach: during the block/for entry, tentatively mark the variable as borrowed. If an async closure is encountered later that captures it, upgrade the entry to consumed (change state in the owning scope's ownership map and decrement any borrow counter). Any code between block entry and the async closure that relied on the borrow is still valid (borrow semantics are a subset of consume — the variable was alive until the consume point)

#### Rule 15: For-loop borrows iterable

In `parseForStatement`:
- Before parsing the loop body, `MarkBorrowed` on the range expression's variable
- After the loop body, `ReleaseBorrowed`
- Handle nesting with a counter (the borrow counter in the current scope)
- Before finalizing borrow vs. consume, scan the loop body for async closures capturing the iterable (see Rule 14)

#### Rule 16: Consumption is permanent across branches

After `parseIfStatement` or `parseMatch`:
- Any variable consumed in ANY branch → mark consumed for the rest of the scope
- Track per-branch consumption and union across branches
- For partial moves (rule 19), apply the union per-field: if `c.data` is consumed in any branch, `c.data` is dead after the conditional; `c.label` remains alive

#### Rule 19: Partial moves for struct fields

In `parseAssignment` when a field selector is used as RHS:
- Mark only that specific field as consumed (`MarkFieldConsumed`)
- The struct variable and other fields remain alive
- For whole-struct operations (pass to proc, return, pass to func), check `IsFullyAlive`
- Re-assigning a moved-out field (`c.data = @slice<int64>(5)`) makes that field alive again. The re-assignment is tracked by removing the field's consumed state. After re-assignment, `IsFullyAlive` returns true

#### Rule 20: Match on enum borrows vs consumes

In `parseMatch`:
- If match arm bindings are immutable → borrow the subject (alive after match)
- If match arm bindings use `var` → consume the subject

#### Rule 22: Index expressions borrow

In expression parsing for index expressions:
- If LHS is a `var`, the container is borrowed for the read duration
- `&items[i]` extends the borrow for the reference's lifetime — the reference's last-use analysis determines when the borrow is released. For simplicity in v1, the borrow is released at the end of the enclosing statement or scope (conservative)

#### Rule 24: Map/set insertion consumes key/value

In the map literal/set literal handling or key-value assignment parsing:
- Keys/values that are `var` identifiers are consumed

**Files affected**: `internal/parser/statement.go`, `internal/parser/assignment.go`, `internal/parser/for_statement.go`, `internal/parser/if_statement.go`, `internal/parser/match.go`, `internal/parser/expression.go`

### Step 4: Func Restriction Checks

In `parseProcedureLiteral` and declaration parsing:

- **Rule 6**: `func` cannot have `var` parameters, cannot contain `async`, cannot be `async` itself
- **Rule 9**: `proc` can be `async`; `func` cannot
- Check: when parsing `Function: true` procedures, reject `QualifierVariable` on parameters, reject `async` keyword usage

**Files affected**: `internal/parser/declaration.go`, `internal/parser/parser.go`

### Step 5: Error Messages

Define ownership-specific error messages in an error table:

| Violation | Error |
|-----------|-------|
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

### Step 6: Parser Ownership Tests

Add test cases covering all rules from `ownership.cogs` to existing parser test files:

- New test file: `internal/parser/ownership_test.go`
- For each scenario in `ownership.cogs`, write a test that verifies the correct error or no-error. Cover both **positive tests** (should compile) and **negative tests** (should error) including:
  - `main_revive` — partial move + re-assignment revives struct (no error)
  - `main_partial_borrow` — narrower field borrow (no error)
  - `main_enum_borrow` — immutable match borrows enum (no error)
  - `main_chain` — chained method call (no error)
  - `main_enum_consume` — all-immutable match arms (no error)
  - `main_defer_ok` — defer with no subsequent consumption (no error)
  - `main_cond_partial` — conditional partial move (no error)
  - `main_return_partial` — return after re-assign revives struct (no error)
  - `main_defer_reassigned_after` — defer followed by reassignment (should error)
  - `main_defer_double` — two defers on same variable (should error)
  - `main_defer_already_dead` — defer on already-consumed variable (should error)
  - `main_map_key_var` — map insertion consumes key (should error: key dead after)
  - Rule 6 violations: `func` with `var` param (should error)
  - Rule 9 violations: `async func` (should error)
- Use the existing `parse` and `parseShouldError` helpers from `internal/parser/helpers_test.go`

---

## Phase 2: Transpiler (Code Generation)

### Goal

Generate correct Go code that enforces ownership at runtime:
1. Deep copy when immutable → `var` assignment (rule 5)
2. No-op borrowing (already safe in Go, only affects parser tracking)
3. Dynamic variable deep copy on read/write (rule 18)

### Step 7: PointerLike Detection

Add a function to determine if a type is pointer-like (needs deep copy on `immutable → var` assignment):

```go
// In internal/types/is.go or new file internal/types/ownership.go
func IsPointerLike(t Type) bool
```

Logic:
- Primitives (int, float, bool, string): false
- Slices, Maps, Sets: true
- Structs: `types.Struct.IsComplex` (already computed during parsing) — true if any field transitively contains a pointer-like type. If `IsComplex` is false, the struct is a plain value type (all fields are primitives or structs of primitives)
- Arrays: true if element type is pointer-like
- `&` references: false (immutable, copied by pointer-copy)
- Options: true if inner type is pointer-like
- Results: true if value or error type is pointer-like
- Either: true if left or right type is pointer-like
- Alias resolution: recurse through `Underlying()`
- Generic instantiation: evaluate at instantiation time
- Interface/any: false (interface header copy, not deep copy — though interface usage as value type is restricted by rule 17)

This mirrors the existing `types.IsPointer` but is more precise (includes structs with pointer-like fields via `IsComplex`, excludes immutable references). It avoids re-traversing struct field graphs by using the pre-computed `IsComplex` field.

### Step 8: Deep Copy Generation in Transpiler

In `internal/transpiler/declaration.go`, modify the `convertDecl` method:

**For `immutable → var` assignments** (rule 5):
- When the RHS is immutable and the LHS has `QualifierVariable`
- And the type `IsPointerLike`:
  - Wrap the RHS expression in `cog.Copy(...)` call
  - Example: `b : var []int64 = a` → `var b = cog.Copy(a)`

**For `var → var` assignments** (rule 5):
- Already handled by parser (consumes source)
- The Go code is a simple assignment (ownership semantics are compile-time)

### Step 9: Dynamic Variable Deep Copy (Rule 18)

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

### Step 10: Map/Set Insertion Copy (Rule 24)

In `internal/transpiler/expression.go`, in the `MapLiteral` and `SetLiteral` cases:

- For each key/value pair where the source expression is a `var` pointer-like type, wrap the converted Go expression in `cog.Copy(...)`. Since `MapLiteral` builds `goast.KeyValueExpr` entries, the wrapping happens per-element before insertion
- For `SetLiteral`, wrap each value expression similarly
- For infix index assignment (`m[key] = val`), handled in the transpiler's statement conversion: when the assigned value is a `var` pointer-like type, wrap in `cog.Copy(...)`
- The parser already marks the source consumed; the transpiler ensures the Go copy preserves independence

**Implementation detail**: The transpiler currently has no runtime notion of `var` vs immutable — the qualifier is only on the AST `Identifier`. The key/value expressions need to be inspected for their qualifier during transpilation, or the parser must annotate the AST nodes to signal "this expression needs a copy." The recommended approach: add a boolean flag to the `ast.MapPair` struct or to `ast.MapLiteral`/`ast.SetLiteral` indicating which key/value indices came from `var` sources, set during parser ownership tracking.

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

The recommended order minimizes risk and allows incremental verification:

| Step | Description | Testable | Risk |
|------|-------------|----------|------|
| 1 | Ownership state types | ✅ Unit test | Low |
| 2 | Core ownership API | ✅ Unit test | Low |
| 7 | PointerLike detection | ✅ Unit test | Low |
| 3 | Borrow/consume rules in parser | ✅ Integration test | High |
| 4 | Func restriction checks | ✅ Integration test | Low |
| 5 | Error messages | ✅ Already covered | Low |
| 6 | Parser ownership tests | ✅ Verify | Low |
| 8 | Deep copy generation | ✅ Verify output | Medium |
| 9 | Dynamic deep copy | ✅ Verify output | Low |
| 10 | Map/set insertion copy | ✅ Verify output | Medium |
| 11 | Transpiler ownership tests | ✅ Verify | Low |

Note: Step 3 risk is raised to **High** because it encompasses the widest surface area: 13 distinct rules, including complex interactions between borrow-scopes and async closures (rule 14), defer forward-checking (rule 13), and conditional branch union (rule 16). Step 10 risk is raised to **Medium** because it requires AST annotation plumbing between parser and transpiler.

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
