# Monomorphization Plan

## Overview

Replace Go-generic emission with **monomorphization**: for each generic function, emit one concrete Go function per call-site type combination. Generic type parameters are **opaque until type-asserted** via `match` — the transpiler uses the match cases to dead-branch-eliminate per-concrete-type copy. A `default` match case produces a single Go-generic fallback for all types without a specific case.

This eliminates Go's GCShape stenciling overhead (interface boxing, dictionary indirection, escape analysis defeats) while giving the programmer explicit control over specialization.

---

## Design Decisions

### 1. Opaque-until-asserted rule

A generic parameter `T` **cannot** be used in expressions, assignments, or as an argument to any function call until its type is narrowed by a `match` case branch. Before the match, the only legal operations on `T`-typed values are:

- Passing them to `match`
- Passing them to another function whose type-constraint matches `T`.
- Type-asserting them (excluded from this scope, design pending)
    - Perhaps builtin? (e.g. `@as<T ~ any>(x any) T?`)

This rule is enforced at parse time / type-check time, not at transpile time.

**Rationale:** if a value of type `T` could be used before the match, the transpiler would need to emit code that works for all possible types — which is exactly what Go generics do. By forbidding this, we guarantee that all emitted code operates on a known concrete type.

### 2. `match` on generic type parameters

Extends the existing `match` design (from `type_switch_plan.md`) to work on generic type parameters in addition to unions:

```cog
process : func<T ~ number>(x : T) = {
    match x {
    case int64:
        @print(x + 1)    // x is int64 here
    case float64:
        @print(x * 2.0)  // x is float64 here
    default:
        @print(x)         // x is still T (generic) here
    }
}
```

The `default` case is the **only** place where `T` may remain unresolved. Code in `default` is transpiled using Go generics with the original constraint. Code in specific `case` branches is transpiled as concrete typed code.

### 3. No single-line type assert

Dropped from design. If you need a single concrete type, write a non-generic function. The `match` statement is the sole mechanism for type narrowing.

### 4. No generic-to-generic calls

A generic function **cannot** call another generic function with an unresolved type parameter as type argument. This follows from the opaque-until-asserted rule: if `T` hasn't been matched, it can't be passed anywhere.

Inside a `match` case, `T` is resolved to a concrete type — calling another generic function with that concrete type is fine (it's just a normal call).

Inside a `default` case, `T` is unresolved. Calls to other functions are allowed only if those functions accept the same constraint or a wider one. This is transpiled as a Go generic-to-generic call within the generic fallback function.

**Consequence:** you cannot compose generic "pass-through" wrappers. This is intentional: if you don't inspect the type, the function shouldn't be generic. Use concrete types or unions instead. Builtins (`@slice`, `@map`, etc.) that legitimately need to hold arbitrary types are handled specially by the transpiler.

### 5. `default` case and the hybrid model

When a `match` has a `default` case, the transpiler produces:

- **One concrete function per specific `case` type** — monomorphized, no generics
- **One Go-generic function for `default`** — with the original constraint, containing only the default case body

Call sites for types that match a specific case are rewritten to the concrete function. All other call sites go to the generic fallback.

When there is **no** `default` case, the match must be **exhaustive** over the constraint. Every type in the constraint must have a case. The transpiler produces only concrete functions, no generic fallback.

As follow-up optimization, we should only generate the concrete which are actually used.

### 6. Type parameter limit

Maximum of **3** type parameters per function, matching the existing `@map<K, V, I>` design. The combinatorial concern is mitigated by the fact that only *call-site combinations* produce implementations, not the full constraint cross-product.

**Practical estimates (call-site combinations, not constraint space):**

| Type params | Typical call-site variants | Worst-case concrete functions per generic |
|---|---|---|
| 1 | 2–5 | 5 + 1 default |
| 2 | 3–10 | 10 + 1 default |
| 3 | 5–15 | 15 + 1 default |

Real programs call `@map<utf8, int64, uint32>` and perhaps a handful of other combinations — not 5 × 5 × 5 = 125. The `default` fallback further limits this: only types with explicit `match` cases get monomorphized.

### 7. Setup code before `match`, and wrap-up code after `match`.

Allowed. Code before the `match` that doesn't touch the generic-typed values is duplicated into each concrete implementation. This is common for non-generic setup (logging, validation of non-generic args, etc.).

After `match`, `T` is unspecified again, so cannot be addressed.

### 8. Methods as plain functions with Go method wrappers

Cog uses `export` keyword for visibility, not capitalization. The transpiler prefixes unexported uppercase names with `_` to prevent unwanted Go exports (e.g., `String` → `_String`). This breaks Go interface satisfaction, which requires exact method name matching.

Monomorphization eliminates this conflict for cog-level generics, but Go stdlib interop (e.g. `fmt.Stringer`, `io.Reader`) still requires real Go methods with correct names. The solution is a **two-layer emission**:

1. **Plain functions** — carry the method body, used by monomorphized generic code via direct calls. Named with receiver prefix: `Foo.String` → `_Foo_String(this _Foo)`.
2. **Go method wrappers** — one-line forwarders that satisfy Go interfaces. Always use the **original cog name** (no `_` prefix), since methods are receiver-scoped and don't pollute the package namespace. An unexported type with exported methods is normal Go — the stdlib itself does this internally.

The `export` keyword on methods controls **cog-level** visibility (enforced by the cog compiler when other cog packages import the type), not Go-level method naming. The Go wrapper always uses the original name for interface satisfaction.

**Cog-level interface constraints** (e.g. `Stringer`) are still enforced at compile time only via `Satisfies`/`Implements`. No cog `interface` types are emitted into Go code — they exist purely for the cog type checker.

This also sidesteps the Go restriction that methods cannot have type parameters (until Go 1.27+), since the monomorphized call path uses plain functions.

**Example:**

```cog
Stringer ~ interface {
    String : func() utf8
}

Print : func<T ~ Stringer>(x : T) = {
    @print(x.String())
}

Foo ~ struct { value : utf8 }
Foo.String : func() utf8 = { return "Hello, world!" }

main : proc() = {
    Print(Foo{})
}
```

Transpiles to:

```go
// Plain function — method body lives here, called by monomorphized code
func _Foo_String(this _Foo) string {
    return "Hello, world!"
}

// Go method wrapper — satisfies Go interfaces (fmt.Stringer, etc.)
func (this _Foo) String() string {
    return _Foo_String(this)
}

// Monomorphized generic — calls plain function directly
func _Print_Foo(x _Foo) {
    builtin.Print(_Foo_String(x))
}

type _Foo struct { value string }

func main() {
    _Print_Foo(_Foo{})
}
```

**Key properties:**
- Monomorphized code calls `_Foo_String(x)` — no interface dispatch overhead
- `_Foo` satisfies `fmt.Stringer` via the Go method wrapper — stdlib interop works
- Method naming never conflicts — methods are receiver-scoped, not package-scoped
- Wrapper is a single forwarding call — zero logic duplication

### 9. Cog-only constraints — no Go-expressibility requirement

Cog constraints do not need to be expressible as Go type constraints. They are enforced entirely by the cog compiler at compile time. Go only sees monomorphized concrete functions — no type parameters, no constraints.

This means cog can support constraints that are impossible in Go:

```cog
// ascii is []byte in Go — no ~ support. Go can't express ~ascii.
// interface inside a constraint — not allowed in Go.
// type parameter on interface itself — not allowed in Go.
stringer ~ string | interface<S ~ string> {
    String : func() S
}
```

The cog compiler checks at each call site: does the concrete type satisfy `stringer`? Either it's a `string` type (utf8 or ascii), or it has a `String() S` method where `S` satisfies `string`. This is pure cog type-system logic — Go never evaluates it.

For monomorphized output, `Print<T ~ stringer>(x : T)` called with `Foo` just becomes `_Print_Foo(x _Foo)`. The constraint is erased.

For the `default` fallback, the transpiler emits the **widest Go-safe approximation** of the cog constraint. Since the cog compiler already rejected all invalid call-site types, the Go constraint only needs to be wide enough to accept the remaining valid types. In many cases `any` suffices. For interface constraints with methods, the fallback emits a plain Go `interface { Method() ReturnType }` using the non-mangled method names (which exist as Go method wrappers per §8).

**Consequence:** cog's constraint system is strictly more expressive than Go's. Constraints involving `ascii` (~`[]byte`), parameterized interfaces, interfaces nested inside union constraints, and structural matching beyond what Go supports are all valid. The cost is that `default` fallback constraints may be looser in Go than in cog — but this is safe because cog already guards the entry points.

---

## Transpilation Examples

### Input

```cog
process : func<T ~ int>(x : T) int64 = {
    logger := initLogger()

    match x {
    case int32:
        return int64(x)
    case int64:
        return x
    default:
        return int64(x)
    }
}

main : proc() = {
    a := process(42)         // int64 → calls process_int64
    b : int32 = 10
    c := process(b)          // int32 → calls process_int32
    d : int16 = 5
    e := process(d)          // int16 → calls process (generic default)
}
```

### Output

```go
// Concrete: int32 case
func process_int32(x int32) int64 {
    logger := initLogger()
    return int64(x)
}

// Concrete: int64 case
func process_int64(x int64) int64 {
    logger := initLogger()
    return x
}

// Generic fallback: default case
func process[T ~int8 | ~int16 | ~int128](x T) int64 {
    logger := initLogger()
    return int64(x)
}

func main() {
    a := process_int64(42)
    var b int32 = 10
    c := process_int32(b)
    var d int16 = 5
    e := process(d)
}
```

Note: the generic fallback's constraint **excludes** the monomorphized types (`int32`, `int64`), keeping only `~int8 | ~int16 | ~int128`.

---

## Implementation Phases

### Phase 1: `match` statement for unions

*Implements the existing `type_switch_plan.md` — prerequisite for everything else.*

**1.1 — Tokens & grammar**
- Add `Match` token to `tokens.Type` enum in `internal/tokens/type.go`
- Add `"match"` to keyword lookup in `internal/tokens/lookup.go`
- Add `match_statement` production to `cog.ebnf`

**1.2 — AST nodes**
- Create `internal/ast/match.go` with `Match` statement node:
  - `Subject Expression`
  - `Binding *Identifier` (nil if no binding)
  - `Cases []*MatchCase`
  - `Default *Block` (nil if exhaustive)
- Create `internal/ast/match_case.go` with `MatchCase` node:
  - `MatchType types.Type`
  - `Tilde bool`
  - `Body *Block`

**1.3 — Parser**
- Add `match` case to statement parser in `internal/parser/statement.go`
- Implement `parseMatch` in new file `internal/parser/match.go`
- Union subject: resolve case types against union variants
- Binding form: create scoped variable with narrowed type per case
- Validate: no duplicate cases, all cases compatible with subject

**1.4 — Transpiler**
- Add `*ast.Match` handling in `internal/transpiler/statement.go`
- Union lowering: emit `switch subject.Tag { case false: ..., case true: ... }`
- Emit per-branch binding assignment when binding form is used

**1.5 — Tests**
- Parser tests: basic union match, binding form, `~` case, duplicate error, exhaustiveness
- Transpiler tests: verify emitted `switch tag` shape and bindings
- Integration test in example program

**Files modified:** `internal/tokens/type.go`, `internal/tokens/lookup.go`, `cog.ebnf`, `internal/ast/match.go` (new), `internal/ast/match_case.go` (new), `internal/parser/statement.go`, `internal/parser/match.go` (new), `internal/transpiler/statement.go`

---

### Phase 2: `match` on generic type parameters

*Extends Phase 1's `match` to work inside generic function bodies.*

**2.1 — Parser: generic match subject**
- When parsing `match x`, if `x` resolves to a `*types.TypeParam`, enter generic match mode
- Each `case` type must satisfy the type parameter's constraints (checked via `TypeParam.SatisfiedBy`)
- In each case branch, create a scoped variable (or narrow existing) with the concrete case type
- `default` case: the binding stays typed as `T` (the type parameter)

**2.2 — Exhaustiveness checking**
- If no `default` case: every concrete type in the constraint's `Constraints` slice must be covered by a `case`
- The checker walks `types.Generic.Constraints` (e.g., `Generics["int"].Constraints` has 5 entries: `int8`, `int16`, `int32`, `int64`, `int128`)
- Report compile error for missing types

**2.3 — AST annotation**
- The `Match` node gains a `GenericParam *types.TypeParam` field (nil for union matches)
- Each `MatchCase` records the concrete `types.Type` it handles

**2.4 — Tests**
- Parser tests: generic match with all cases, generic match with default, exhaustiveness error, constraint satisfaction error
- Verify type narrowing: inside case branch `T` variables are typed concretely

**Files modified:** `internal/parser/match.go`, `internal/ast/match.go`, `internal/types/generics.go` (possible helper for exhaustiveness)

---

### Phase 3: Opaque-until-asserted enforcement

*Compile-time rule: `T`-typed values cannot be used before `match`.*

**3.1 — Scope tracking for generic params**
- In the parser, when entering a generic function body, mark each `TypeParam` variable as `opaque` in the scope
- Only `match` unblocks the variable (sets it to `narrowed` within case branches)
- In `default` blocks, the variable stays `opaque` but is marked `default-context` — allowed for constraint-compatible operations

**3.2 — Usage validation**
- In `expression()`, `parseLiteral()`, `parseCallArguments()`, and all paths that resolve an identifier to its type: if the resolved type is `*types.TypeParam` and the variable is marked `opaque`, emit a parse error: `"cannot use generic parameter %q before type match"`
- In `default` blocks: allow usage only in contexts where Go generics support the operation (assignment, constraint-satisfying operations, passing to functions with matching or wider constraints)

**3.3 — Tests**
- Error tests: use of `T` before match, use of `T` in function call before match
- Success tests: use of `T` inside match case, non-generic setup code before match

**Files modified:** `internal/parser/scope.go` (or relevant scope tracking), `internal/parser/ebnf_parser.go`, `internal/parser/match.go`

---

### Phase 4: Monomorphization transpiler pass

*The core transformation: collect call sites, emit concrete + fallback functions, rewrite calls.*

**4.1 — Call-site collection pre-pass**
- Before `Transpile()` emits declarations, walk the entire AST
- For each `*ast.Call` with `TypeArgs`, record: function name → set of concrete type-arg tuples
- Store in `Transpiler.monomorphMap: map[string][]MonoInstance`
- Each `MonoInstance` records the concrete types and the suffix name

```go
type MonoInstance struct {
    TypeArgs  []types.Type    // concrete types for this instance
    Suffix    string          // e.g., "_int64" or "_int32_utf8"
    MatchCase *ast.MatchCase  // which case branch to emit (nil → default)
}
```

**4.2 — Match case routing**
- For each generic function declaration, find its `match` statement
- For each `MonoInstance`, determine which match case handles its type:
  - If a specific `case int64:` exists → route to that case
  - Otherwise → route to `default`
- Group default-routed instances: they all share one generic fallback

**4.3 — Concrete function emission**
- For each (function, specific case) pair, emit a Go function:
  - Name: `funcName_TypeName` (multi-param: `funcName_Type1_Type2`)
  - Parameters: substitute `T` → concrete type
  - Body: setup code (everything before `match`) + case branch body
  - No type parameters, no generics — fully concrete
- Function name mangling uses `types.Type.String()` to generate suffix

**4.4 — Generic fallback emission**
- If the function has a `default` case AND at least one call site routes to it:
  - Emit one Go generic function with the original name
  - Type parameters: original constraints **minus** the monomorphized types
  - Body: setup code + default branch body
- If no call site routes to default: don't emit the fallback at all

**4.5 — Call-site rewriting**
- For monomorphized types: emit plain `process_int64(args...)` — no type args
- For default-routed types: emit `process[int16](args...)` — keep Go generic syntax

**4.6 — Method-to-function lowering**
- Each method declaration emits **two** Go declarations:
  1. **Plain function**: `Foo.Method` → `func _Foo_Method(this _Foo, ...)` — carries the method body
  2. **Go method wrapper**: `func (this _Foo) Method() { return _Foo_Method(this) }` — one-line forwarder for Go interface satisfaction
- The wrapper always uses the **original cog method name** (no `_` prefix). Methods are receiver-scoped in Go, so this doesn't conflict with the package-level `_` prefix convention for unexported globals
- Monomorphized bodies call the plain function directly: `x.Method()` where `x : Foo` → `_Foo_Method(x)`
- Non-generic method calls (`foo.Method()` outside any generic context) also use the plain function
- Go stdlib interop works because the wrapper satisfies Go interfaces (`fmt.Stringer`, `io.Reader`, etc.)
- Inside `default` fallback bodies, method calls on unresolved `T` can use the Go method wrapper (since it exists as a real Go method), allowing Go generic-to-method dispatch

**4.7 — Constraint narrowing for fallback**
- The fallback's constraint must exclude monomorphized types
- Build a new tilde-union from `constraint.Constraints` minus the concrete types with specific cases
- Use existing `component.TildeUnion(...)` with the reduced set

**4.7 — Tests**
- Transpiler tests: verify concrete function names, setup code duplication, default fallback with narrowed constraint, call-site rewriting
- Integration tests: transpile and `go vet` / `go run` the output
- Edge cases: function with only `default` (pure generic), function with no `default` (all concrete), function with multiple type params

**Files modified:** `internal/transpiler/transpiler.go` (pre-pass), `internal/transpiler/declaration.go` (emission), `internal/transpiler/expression.go` (call rewriting), `internal/transpiler/monomorphize.go` (new)

---

### Phase 5: Multi-type-param monomorphization

*Extends Phase 4 to handle 2–3 type parameters.*

**5.1 — Nested match handling**
- Functions with multiple type params have nested or sequential matches:

```cog
combine : func<A ~ int, B ~ string>(a : A, b : B) = {
    match a {
    case int32:
        match b {
        case ascii: ...
        case utf8: ...
        }
    default:
        match b {
        case ascii: ...
        default: ...
        }
    }
}
```

- The monomorphizer must handle match nesting: each concrete instance is determined by the *combination* of case branches taken

**5.2 — Suffix generation**
- Multi-param suffix: `combine_int32_ascii`, `combine_int32_utf8`
- For default-routed params, the suffix omits that param and the function keeps it as a type parameter

**5.3 — Partial monomorphization**
- If only `A` has a specific match but `B` has only `default`:
  - Emit `combine_int32[B ~ascii | ~utf8](a int32, b B)` — monomorphized on `A`, still generic on `B`
- This avoids unnecessary duplication while specializing the known dimension

**5.4 — Tests**
- 2-param and 3-param combinations
- Partial monomorphization: one param concrete, one param default
- All-default (pure generic output)

**Files modified:** `internal/transpiler/monomorphize.go`

---

### Phase 6: Dead code elimination refinements

*Polish pass — remove unreachable code from emitted functions.*

**6.1 — Per-instance body trimming**
- Each concrete instance includes only: setup code + its specific case body
- The `match` statement itself is not emitted in concrete instances
- Variable declarations for the match subject are replaced with direct parameter usage

**6.2 — Unused import cleanup**
- After monomorphization, some instances may not use imports that the original function needed
- Run existing `finalizeImports` logic per-instance rather than per-file

**6.3 — Tests**
- Verify no `match`/`switch` in concrete output
- Verify no dead branches in concrete output
- Verify import cleanup

**Files modified:** `internal/transpiler/monomorphize.go`, `internal/transpiler/transpiler.go`

---

## File Change Summary

| File | Phase | Change |
|---|---|---|
| `internal/tokens/type.go` | 1 | Add `Match` token |
| `internal/tokens/lookup.go` | 1 | Add `"match"` keyword |
| `cog.ebnf` | 1 | Add `match_statement` production |
| `internal/ast/match.go` (new) | 1, 2 | `Match` and `MatchCase` AST nodes |
| `internal/ast/match_case.go` (new) | 1, 2 | `MatchCase` node |
| `internal/parser/match.go` (new) | 1, 2, 3 | Match parsing: unions + generic params + opaque enforcement |
| `internal/parser/statement.go` | 1 | Route `match` token to `parseMatch` |
| `internal/parser/scope.go` | 3 | Opaque tracking for generic params |
| `internal/parser/ebnf_parser.go` | 3 | Reject opaque `T` usage in expressions |
| `internal/transpiler/statement.go` | 1 | Union match → `switch tag` emission |
| `internal/transpiler/monomorphize.go` (new) | 4, 5, 6 | Call-site collection, concrete emission, rewriting |
| `internal/transpiler/transpiler.go` | 4 | Pre-pass hook, instance tracking |
| `internal/transpiler/declaration.go` | 4 | Concrete function emission, method-to-function lowering |
| `internal/transpiler/expression.go` | 4 | Call-site rewriting, method call → function call rewriting |
| `internal/transpiler/type.go` | 4 | Constraint narrowing for fallback |

---

## Open Questions

1. **Builtin generic functions** (`@slice`, `@map`, `@if`): these are compiler-provided and don't have user-written `match` bodies. They continue to be transpiled as today (special-cased in the transpiler). Monomorphization only applies to user-defined generic functions.

2. **Generic type aliases** (e.g., `Pair<K, V> ~ struct { ... }`): these are type-level, not function-level. They continue to use Go's generic type syntax. Monomorphization applies only to functions/procedures. Note that cog-level constraints on generic type aliases are still checked by the cog compiler — the Go type parameters use a Go-safe approximation of the constraint (widened if necessary, per §9).

3. **Recursive generic functions**: a generic function that calls itself with a different concrete type (e.g., `process<int32>` calling `process<int16>`) is legal as long as the recursive call uses a concrete type (matched in a case branch). The call-site collector must transitively discover these.

4. **Code size**: monomorphization trades binary size for speed. For most cog programs (CLIs, servers) this is a good trade. If it becomes a concern, the `default` fallback naturally limits duplication.

5. **`default` case with interface constraints**: in a `default` fallback body, method calls on `T`-typed values cannot be lowered to direct function calls because the concrete type is unknown. However, because Go method wrappers exist on each type (§8), the `default` fallback can use normal Go generic method dispatch — `x.Method()` works in Go generics when the constraint interface requires `Method()`. The fallback's constraint is emitted as a Go `interface { Method() ReturnType }` using the original (non-mangled) method names. No special handling needed.

6. **Non-generic methods**: methods on non-generic types (e.g., `Foo.String` called as `foo.String()` outside any generic context) also use the two-layer emission: plain function `_Foo_String(this)` + Go method wrapper `func (_Foo) String()`. Non-generic call sites use the plain function for consistency. This can be implemented independently of monomorphization as an early transpiler change — it unifies the calling convention across generic and non-generic code.
