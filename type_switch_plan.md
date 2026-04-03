# Type Switch Plan (`match` Statement)

## Goal

Add a type-switching construct that is clearer than Go's `switch x.(type)` and supports Cog-specific needs like `~`-style underlying matching for aliases.

Crucially, this plan addresses the `|` syntax, which unifies two use-cases depending on context:
1. **Concrete Union (Either)**: A concrete sum type of exactly two types (`type | type`, behaving like Haskell's `Either`). Used for variables.
2. **Type Constraint Unions**: Used to bound generic type parameters (e.g., `T ~ string | int | bool`). Used exclusively in generic constraints.

`match` is primarily designed to safely dispatch **generic type parameters** bounded by type constraint unions or `any`. However, for maximal ergonomics, `match` is **also allowed on concrete `Either` types** as an alternative to `?` checks.

---

## Distinguishing "Either" and "Type Constraint Union"

Both use the `|` syntax, but their context, behavior, and valid types differ.

### 1. Either Type (`A | B`)

An `Either` type represents a concrete value that is either `A` (Left/Either) or `B` (Right/Or).
- **Semantics**: It behaves similarly to `Option` (`T?`) or `Result` (`T ! E`).
- **Left (Default)**: The first type (`A`) is the primary type. You can access it by just referencing the variable.
- **Right (Alternative)**: The second type (`B`) can be accessed using the `?` token, meaning "is it the alternative type?".
- **Limitations**:
  - Exactly two types.
  - Can only contain *concrete* types (no interfaces or `any`).
- **Usage**: Typically unwrapped via standard `?` checks, but **can also be used with `match`** to cleanly branch on the two types.

### 2. Type Constraint Union (`A | B | C ...`)

A type constraint defines a set of allowed types for a generic type parameter.
- **Semantics**: Defines a union of types for constraint validation and runtime generic dispatch.
- **Syntax rules**:
  - **Type System Uniformity**: The compiler type system represents both `Either` and `Type Constraint Unions` as a single unified `types.Union` entity (replacing the current strictly-two `Either`/`Or` fields with a `Variants []types.Type` slice). The distinction is enforced by the **type checker** based on usage context:
    - If a `types.Union` is used as a variable type, it is strictly validated to act as an `Either` (requiring exactly 2 concrete types). If it has >2 types, it emits an error.
    - If a `types.Union` is used as a generic constraint (e.g. inside `types.TypeParam`), it is validated as a Type Constraint Union and can contain any number of types, including `any` or other non-concrete types.
- **Limitations**: Can include non-concrete types like `any`, `comparable`, or other interfaces.
- **Usage**: Typically used as type parameter constraints, over which `match` safely dispatches.

---

## High-Level Decisions for `match`

1. Prefer a dedicated `match` keyword over extending `switch` with `type ...` branch syntax.
2. Use `case` branches for type cases.
3. Support matching on:
	- **Generic Type Parameters**: When the subject is a generic `<T>` bounded by a type constraint union or `any`.
	- **Concrete Unions (Either)**: When the subject is a concrete `A | B` variable.
4. Allow `~Type` matching:
	- Fallback to reflect-based matching or compile-time resolution when applicable.
5. Keep must-check analysis unchanged: still only applies to Option, Result, and Either types.

---

## Proposed Language Surface

### Basic form

```cog
match expr {
case TypeA:
case TypeB:
}
```

### Binding form (typed variable per branch)

```cog
match t := expr {
case TypeA:
	 // t has TypeA in this branch
case TypeB:
	 // t has TypeB in this branch
}
```

### Underlying type case

```cog
match t := expr {
case ~ascii:
case utf8:
}
```

---

## Semantic Rules

1. `match` is type-only dispatch.
2. No mixing of value `case` and type `case` in same construct.
3. Branch-local narrowing applies only to binding variable (`t`), not original expression variable.
4. Case types must be compatible with the matched subject type:
	- For generic variables (`T ~ A | B`): case must be a valid variant constraints.
	- For generic variables bounded by `any`: case can be any concrete type.
	- For Either variables (`A | B`): case must be either `A` or `B`.
5. Duplicate coverage of same variant is an error.
6. The subject of `match` MUST be a generic type parameter or an Either Union type. Since `any` functions only as a constraint (not a generic container like `interface{}` in Go), you cannot instantiate a `var x : any` and match over it.

---

## `default` Policy (Open Design)

Questions:

1. What does `default` mean in type matching?
2. What control flow is lost without it?

Recommended direction:

1. For exact type constraint unions:
	- Prefer exhaustive matching.
	- If exhaustive is enforced, disallow `default` as redundant.
2. For `any` constraints:
	- Allow `default` because runtime value may be any concrete type.

Pragmatic rollout option:

1. Phase 1: allow `default` generally (simpler adoption).
2. Phase 2: add lint/error mode requiring exhaustive cases for constraint unions.

---

## Transpilation Strategy

### 1. Concrete Unions (Either)

Since `Either` variables represent concrete values, they should not incur heap allocations from interface boxing (which `any` causes). 
We will introduce a runtime struct representation in `types.go`:

```go
type Either[L any, R any] struct {
    Left L
    Right R
    IsRight bool
}
```

When a `types.Union` is used as a variable (which implies exactly two types are specified), it will transpile to the `cog.Either[Left, Right]` type in Go.
`ast.UnionLiteral` will be updated to hold an `Index` integer that indicates which variant matched during type checking. 
When transpiling `ast.UnionLiteral`, if `Index == 0`, it emits `cog.Either[Left, Right]{Left: val}`. If `Index == 1`, it emits `cog.Either[Left, Right]{Right: val, IsRight: true}`.

If `match` is used on an `Either` struct type, transpile it directly to a Go `if/else` block based on `IsRight`:

```go
if !x.IsRight {
    t := x.Left
    // case MyType
} else {
    t := x.Right
    // case string
}
```

### 2. Generic Type Parameters

Transpile to Go type switch (converting generic parameters to `any` where necessary to satisfy Go's type constraints):

```go
switch t := any(x).(type) {
case MyType:
case string:
default:
}
```

For `~Type` matching, we may need to fallback to runtime reflect checks in the `default` block or via custom transpilation structures since Go's native type switch does not support underlying type `~` cases.

---

## Implementation Plan

### Phase 1: Tokens and Grammar

1. Add `Match` token in token enum.
2. Add `"match"` keyword lookup.
3. Extend grammar with `match_statement` and type-case clause production.
4. Refactor `types.Union` in `internal/types/union.go` to hold a slice `Variants []Type` (DONE). Update `TypeParam` in `internal/types/type_parameter.go` to replace `Constraints []Type` with `Constraint Type` (DONE).
5. Update `ast.UnionLiteral` to track `Index int` indicating which variant of the union matched the expression (NEW).
6. Implement `Either[L any, R any]` in `types.go` (NEW).
7. Modify the transpiler: `types.UnionKind` variables emit `cog.Either[L, R]`. Type boundaries emit interface unions `interface { L | R }` (NEW).

### Phase 2: AST

1. Add `ast.Match` statement node.
2. Add `ast.MatchCase` node.
3. Reuse existing `ast.Default` node for default branch.

Suggested fields:

1. `Match.Subject ast.Expression`
2. `Match.Binding *ast.Identifier` (nil if absent)
3. `Match.Cases []*ast.MatchCase`
4. `Match.Default *ast.Default`
5. `MatchCase.MatchType types.Type`
6. `MatchCase.Tilde bool`

### Phase 3: Parser

1. Parse `match` in statement parser.
2. Determine subject category:
	- Generic Type Parameter: bounded by `any` or union constraints. Resolve case types appropriately and transpile to Go type switch.
	- Concrete Union (Either): validate case types and transpile to Go type switch (as Either variables lower to `any`).
3. In each case arm, create enclosed scope and define binding variable with narrowed type.
4. Validate duplicates and incompatible types.

### Phase 4: Transpiler

1. Add `*ast.Match` conversion path.
2. Emit Go runtime type switch (`switch t := any(subject).(type)`).

### Phase 5: Tests

1. Parser tests:
	- Basic type constraint match.
	- Binding form.
	- `~` underlying case.
	- Duplicate/incompatible case errors.
	- Verify `match` correctly supports `Either` type operands.
2. Transpiler tests:
	- Verify emitted `switch v := any(x).(type)` shape and bindings.
3. Integration example in example program.
