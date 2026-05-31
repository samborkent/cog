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
    stateAlive   ownershipState = 0  // variable is live (default for var)
    stateBorrowed ownershipState = 1 << iota // currently borrowed
    stateConsumed                             // permanently consumed
    stateReserved                             // reserved by defer (rule 13)
)

type fieldState struct {
    state ownershipState
    owner string // struct variable name (for field tracking)
}
```

Add a `*isync.Map[string, ownershipState]` to `SymbolTable` for tracking top-level var state, and `*isync.Map[string, *isync.Map[string, ownershipState]]` for per-field tracking within structs.

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

These walk up scope chains like the existing `IsValueChecked`/`IsErrorChecked`.

**Files affected**: `internal/parser/ownership.go`, `internal/parser/symbol_table.go`

### Step 3: Borrow/Consume Rules in Parser Statement Processing

Modify `parseStatement` and related functions to enforce ownership rules:

#### Rule 4: `var` to `var` assignment consumes source

In `parseAssignment` (in `statement.go`), when the RHS is an identifier with `QualifierVariable` and pointer-like type:
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

#### Rule 13: Defer reserves

In the `defer` case in `parseStatement`:
- After parsing the defer expression, identify captured `var` variables
- Call `MarkReserved` on each
- Check forward: scan remaining statements in the scope for consumption of reserved variables (compile error)

#### Rule 15: For-loop borrows iterable

In `parseForStatement`:
- Before parsing the loop body, `MarkBorrowed` on the range expression's variable
- After the loop body, `ReleaseBorrowed`
- Handle nesting with a counter

#### Rule 16: Consumption is permanent across branches

After `parseIfStatement` or `parseMatch`:
- Any variable consumed in ANY branch → mark consumed for the rest of the scope
- Track per-branch consumption and union across branches

#### Rule 19: Partial moves for struct fields

In `parseAssignment` when a field selector is used as RHS:
- Mark only that specific field as consumed (`MarkFieldConsumed`)
- The struct variable and other fields remain alive
- For whole-struct operations (pass to proc, return), check `IsFullyAlive`

#### Rule 20: Match on enum borrows vs consumes

In `parseMatch`:
- If match arm bindings are immutable → borrow the subject (alive after match)
- If match arm bindings use `var` → consume the subject

#### Rule 22: Index expressions borrow

In expression parsing for index expressions:
- If LHS is a `var`, the container is borrowed for the read duration
- `&items[i]` extends the borrow for the reference's lifetime

#### Rule 24: Map/set insertion consumes key/value

In the map literal/set literal handling or key-value assignment parsing:
- Keys/values that are `var` identifiers are consumed

**Files affected**: `internal/parser/statement.go`, `internal/parser/for_statement.go`, `internal/parser/if_statement.go`, `internal/parser/match.go`, `internal/parser/expression.go`

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
| Consumed twice | `"cannot use %s: variable is consumed"` |
| Borrowed while mutating | `"cannot mutate %s: variable is borrowed"` |
| Move-out while borrowed | `"cannot move %s: %s is borrowed"` |
| Partial move then whole-struct pass | `"cannot move %s: field '%s' is already moved out"` |
| Use of dead field | `"%s: field '%s' moved out here"` |
| Defer on already-consumed | `"cannot defer %s: %s is already consumed"` |
| Defer on reserved | `"cannot defer %s: %s is already reserved by a defer"` |
| Reassign after field consumed by defer | `"cannot reassign %s: field '%s' reserved by defer"` |
| func with var param | `"func cannot have mutable parameters"` |
| async func | `"func cannot be async"` |

### Step 6: Parser Ownership Tests

Add test cases covering all rules from `ownership.cogs` to existing parser test files:

- New test file: `internal/parser/ownership_test.go`
- For each scenario in `ownership.cogs`, write a test that verifies the correct error or no-error
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
- Structs: true if any field is pointer-like (transitive)
- Arrays: true if element type is pointer-like
- `&` references: false (immutable, copied by pointer-copy)
- Options: true if inner type is pointer-like
- Results: true if value or error type is pointer-like
- Alias resolution: recurse through `Underlying()`
- Generic instantiation: evaluate at instantiation time

This mirrors the existing `types.IsPointer` but is more precise (includes structs with pointer-like fields, excludes immutable references).

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

The transpiler needs to know the type of each dyn field. Modify `dynamics` map or add a parallel map for types.

### Step 10: Map/Set Insertion Copy (Rule 24)

In `internal/transpiler/expression.go`, in map/set literal handling and infix index assignment:

- When key/value is a `var` pointer-like type, wrap in `cog.Copy(...)` before insertion
- The parser already marks the source consumed; the transpiler ensures the Go copy preserves independence

### Step 11: Transpiler Ownership Tests

Add tests that verify generated Go output for ownership scenarios:

- `var := immutable` with pointer-like type → generates `cog.Copy(...)` wrapper
- `var := immutable` with primitive type → generates plain assignment
- `var := var` with pointer-like type → plain assignment (ownership transfers)
- Dynamic read/write with pointer-like types → wraps in `cog.Copy(...)`

Use the existing `transpile`/`mustContain` helper from `internal/transpiler/helpers_test.go`.

---

## Implementation Order

The recommended order minimizes risk and allows incremental verification:

| Step | Description | Testable | Risk |
|------|-------------|----------|------|
| 1 | Ownership state types | ✅ Unit test | Low |
| 2 | Core ownership API | ✅ Unit test | Low |
| 7 | PointerLike detection | ✅ Unit test | Low |
| 3 | Borrow/consume rules in parser | ✅ Integration test | Medium |
| 4 | Func restriction checks | ✅ Integration test | Low |
| 5 | Error messages | ✅ Already covered | Low |
| 6 | Parser ownership tests | ✅ Verify | Low |
| 8 | Deep copy generation | ✅ Verify output | Medium |
| 9 | Dynamic deep copy | ✅ Verify output | Low |
| 10 | Map/set insertion copy | ✅ Verify output | Low |
| 11 | Transpiler ownership tests | ✅ Verify | Low |

## Key Design Decisions

1. **No Go runtime changes needed**: The existing `cog.Copy()` function already handles deep copying. The ownership model is a *compile-time* analysis — at the Go level, we just need to:
   - Generate `cog.Copy()` for `immutable → var` transitions (rule 5)
   - Generate deep copies in `dyn` read/write (rule 18)
   - The parser enforces all other rules via error messages

2. **Parser-only tracking for borrow/consume**: The borrow/consume state doesn't generate unique Go code — it's a compile-time safety check. Only the deep-copy rules (5, 18) affect code generation.

3. **Reference-counted borrow**: For nested borrows (rule 15: for-loop nesting), use a counter instead of a boolean flag, matching the spec.

4. **Partial-move via per-field state**: Track each struct field independently in a nested map, matching the existing `fields` map pattern in `SymbolTable`.

5. **`func` vs `proc` distinction**: Already available via `types.Procedure.Function`. The transpiler already uses `t.inFunc` for some restrictions.
