# Type Switch Plan (`match` Statement)

## Goal

Add a type-switching construct that is clearer than Go's `switch x.(type)` and supports Cog-specific needs like `~`-style underlying matching for aliases.

Proposed user-facing syntax:

```cog
match x {
case ~ascii:
case utf8:
}
```

With optional binding for typed access in branches:

```cog
match t := x {
case utf8:
	 @print(t)
case uint64:
	 @print(t)
}
```

---

## High-Level Decisions

1. Prefer a dedicated `match` keyword over extending `switch` with `type ...` branch syntax.
2. Use `case` branches for type cases.
3. Support both:
	- Union-based matching (compile-time resolvable)
	- Any/interface-based matching (runtime type switch; interfaces are future scope)
4. Allow `~Type` matching:
	- Compile-time for unions
	- Runtime/reflect fallback for any/interface (future extension)
5. Keep must-check analysis unchanged: still only applies to option/result types.

---

## Why `match` Instead of `switch type`

1. Avoids Go's esoteric `.(type)` syntax and keeps grammar explicit.
2. Leaves normal `switch` semantics untouched (`case` on values/conditions).
3. Makes non-mixing rule natural: `switch` is value/condition dispatch, `match` is type dispatch.

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
	- For union subject: each case must map to a union variant.
	- For any/interface subject (future): case types must be valid assertions.
5. Duplicate coverage of same union variant is an error.

---

## `default` Policy (Open Design)

Questions:

1. What does `default` mean in type matching?
2. What control flow is lost without it?

Recommended direction:

1. For unions:
	- Prefer exhaustive matching.
	- If exhaustive is enforced, disallow `default` as redundant.
2. For any/interface (future):
	- Allow `default` because runtime value may be many concrete types.

Pragmatic rollout option:

1. Phase 1: allow `default` for unions too (simpler adoption).
2. Phase 2: add lint/error mode for exhaustive-only unions.

---

## Transpilation Strategy

### 1. Union subject (`A | B`)

Current union runtime shape is struct-like with discriminant (`Tag`) and payload fields (`Either`, `Or`).

Transpile:

```go
switch x.Tag {
case false: // Either
	 // optional: t := x.Either
case true:  // Or
	 // optional: t := x.Or
}
```

If binding is used (`match t := x`), prepend per-branch binding assignment:

```go
t := x.Either // or x.Or
```

This avoids reflection and keeps union matching compile-time resolvable and efficient.

### 2. Any/interface subject (future interfaces)

Transpile to Go type switch:

```go
switch t := x.(type) {
case MyType:
case string:
default:
}
```

For `~Type` in this mode, fallback to reflect-based matching when needed.

Optimization rule:

1. Prefer compile-time resolvable checks whenever possible.
2. Only fallback to runtime reflect when static resolution is impossible.

---

## Implementation Plan

### Phase 1: Tokens and Grammar

1. Add `Match` token in token enum.
2. Add `"match"` keyword lookup.
3. Extend grammar with `match_statement` and type-case clause production.

### Phase 2: AST

1. Add `ast.Match` statement node.
2. Add `ast.MatchCase` node.
3. Reuse existing `ast.Default` node for default branch.

Suggested fields:

1. `Match.Subject Expression`
2. `Match.Binding *Identifier` (nil if absent)
3. `Match.Cases []*MatchCase`
4. `Match.Default *Default`
5. `MatchCase.MatchType types.Type`
6. `MatchCase.Tilde bool`
7. `MatchCase.Tag bool` (for union lowering: false=either, true=or)

### Phase 3: Parser

1. Parse `match` in statement parser.
2. Support:
	- `match expr { ... }`
	- `match ident := expr { ... }`
3. Determine subject category:
	- Union: resolve case types against union variants.
	- Any/interface (future): parse but can be feature-gated until interfaces are ready.
4. In each case arm, create enclosed scope and define binding variable with narrowed type.
5. Validate duplicates and incompatible types.

### Phase 4: Transpiler

1. Add `*ast.Match` conversion path.
2. Union lowering:
	- Emit `switch subject.Tag`.
	- Emit `case false/true` based on resolved variant.
	- Emit per-branch binding assignment.
3. Any/interface lowering (future): emit `TypeSwitchStmt`.

### Phase 5: Tests

1. Parser tests:
	- Basic union match.
	- Binding form.
	- `~` case.
	- Duplicate/incompatible case errors.
2. Transpiler tests:
	- Verify emitted `switch tag` shape and bindings.
3. Integration example in example program.

---

## Open Questions to Resolve Before Finalizing Behavior

1. Union exhaustiveness:
	- Required compile error when missing variant, or optional?
2. `default` in union matches:
	- Always disallow, always allow, or allow only when non-exhaustive mode enabled?
3. Multi-variant unions in future:
	- If unions become N-ary, should discriminant evolve from bool to small integer?
4. Reflect policy for `~` in any/interface mode:
	- Full support from day one, or feature-flagged follow-up?

---

## Recommended First Milestone

Implement `match` for union subjects only, including binding and `~` support, with transpilation to `switch subject.Tag`.

Reason:

1. No interface subsystem dependency.
2. No reflect dependency.
3. Delivers most of the ergonomic and correctness value immediately.

Then layer in any/interface type-switch support once interface types exist in parser/type system.
