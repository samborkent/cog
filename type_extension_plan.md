# Type System Extension Plan: Generics & Foundations

> **Prerequisites** for the `match` type-switch feature (see `type_switch_plan.md`).

## Current State

| Feature | Status |
|---|---|
| `any` token | Exists in lexer (`tokens.Any`), registered in `tokens.Keywords` and `tokens.Types` |
| `any` type | **Missing** — no `types.Any`, no `AnyKind` in kind enum |
| Builtin constraints (`string`, `int`, `uint`, `float`, `complex`) | Defined in `types.Generics` map with `Generic` struct |
| Composite constraints (`signed`, `number`) | **Missing** |
| `Satisfies()` helper | **Missing** — no check if concrete type satisfies a constraint |
| Generic builtin calls (`@if<T>`, `@slice<T>`, etc.) | Implemented via `parseTypeArguments` + ad-hoc per-builtin validation |
| Generic user calls (`myFunc<T>(...)`) | **Not implemented** |
| Generic type aliases (`Slice<T any> ~ []T`) | **Not implemented** |
| Generic proc/func definitions | **Not implemented** — `types.Procedure` has no `TypeParams` |
| Interfaces | **Not implemented** |

## Dependency Graph

```
Phase 1 (any) ───────────────┐
                              ├──→ Phase 3 (TypeParam infra) ──→ Phase 4 (Generic aliases)
Phase 2 (constraints) ───────┘                                        │
                                                                      ├──→ Phase 5 (Generic procs)
                                                                      │
                                                               Phase 6 (Interfaces, optional)
                                                                      │
                                                               match statement (type_switch_plan.md)
```

Phases 1 and 2 are independent → **parallel**.
Phase 3 depends on 1 + 2.
Phases 4 and 5 depend on 3, but are **parallel with each other**.
Phase 6 is optional — `match` on unions works without it.

---

## Phase 1: `any` Type

**Goal**: Make `any` a first-class type in the type system.

### Steps

1. Add `AnyKind` to `types.Kind` enum (after `GenericKind`).
2. Create `types.Any` singleton — dedicated struct returning `AnyKind` from `Kind()`, `"any"` from `String()`, self from `Underlying()`.
3. Wire `tokens.Any` → `types.Any` in `types.Lookup` map.
4. In parser type resolution (`parseType` / `parseCombinedType`), handle `tokens.Any` → return `types.Any`.
5. In transpiler `convertType`, map `AnyKind` → `goast.Ident{Name: "any"}`.
6. Update `types.Equal`: `AnyKind` equals only `AnyKind`.
7. Update `types.AssignableTo`: any concrete type is assignable **to** `any` (boxing). `any` is not assignable to a concrete type without assertion.

### Files

- `internal/types/kind.go` — add `AnyKind`
- New `internal/types/any.go` — `types.Any` singleton
- `internal/types/helpers.go` — `Equal`, `AssignableTo` updates
- `internal/parser/type.go` — handle `tokens.Any`
- `internal/transpiler/type.go` — `AnyKind` case

### Tests

- Parse `x : any = 5`
- Type alias `A ~ any`
- Function parameter `f : proc(x : any)`
- `AssignableTo(int32, any)` → true
- `AssignableTo(any, int32)` → false
- `Equal(any, any)` → true, `Equal(any, int32)` → false

---

## Phase 2: Complete Builtin Generic Constraints

**Goal**: Add missing composite constraints and a `Satisfies` helper.

### Steps

1. Add `signed` and `number` entries to `types.Generics` map:
   - `signed`: all int + float + complex basic types (flatten `int`, `float`, `complex` constraint lists)
   - `number`: all of `signed` + all `uint` basic types
2. Add `Satisfies(concrete Type, constraint Type) bool` to `types/helpers.go`:
   - If constraint is `*Generic`: check if `concrete.Kind()` matches any type in constraint's list (recursively for sub-generics).
   - If constraint is `any` / `AnyKind`: always true.
   - If constraint is concrete: fall back to `Equal`.
3. Export the `constraints` field on `Generic` (or add `Constraints() []Type` accessor).

### Files

- `internal/types/generics.go` — add `signed`, `number`; export constraints
- `internal/types/helpers.go` — add `Satisfies`

### Tests

- `Satisfies(int32, Generics["int"])` → true
- `Satisfies(utf8, Generics["int"])` → false
- `Satisfies(float32, Generics["signed"])` → true
- `Satisfies(uint64, Generics["number"])` → true
- `Satisfies(bool, Generics["number"])` → false
- `Satisfies(anything, types.Any)` → true

### Go transpilation of constraints

| Cog constraint | Go output |
|---|---|
| `any` | `any` |
| `string` | custom interface `~string \| ~[]byte` (or `any` + cog compile-time check) |
| `int` | custom interface `~int8 \| ~int16 \| ...` |
| `uint` | custom interface `~uint8 \| ~uint16 \| ...` |
| `float` | custom interface (note: `float16` is library type) |
| `complex` | custom interface (note: `complex32` is library type) |
| `signed` | union of int + float + complex interfaces |
| `number` | union of signed + uint interfaces |

**Library-type problem**: `float16`, `complex32`, `int128`, `uint128` are not Go builtins — they cannot appear in Go `~` constraint unions.

**Recommendation**: Emit `any` as the Go constraint for constraints involving library types. Rely on cog's compile-time `Satisfies` check for correctness. For constraints with only Go-native types, emit proper Go constraint interfaces.

---

## Phase 3: Type Parameter Infrastructure

**Goal**: Represent type parameters on types and procedures. *Depends on Phases 1 + 2.*

### Steps

1. Create `TypeParam` struct in new `internal/types/typeparam.go`:
   - Fields: `Name string`, `Constraint Type` (a `*Generic`, `types.Any`, or concrete type)
   - Implements `types.Type`: `Kind()` → `GenericKind`, `String()` → `Name`, `Underlying()` → self
   - **Distinct from `Generic`**: `Generic` = constraint family (set of allowed types); `TypeParam` = named placeholder with a constraint.

2. Add `TypeParams []*TypeParam` field to `types.Procedure`.

3. Add `TypeParams []*TypeParam` field to `types.Alias`.

4. Add `parseTypeParams(ctx) []*types.TypeParam` to `arguments.go` (for declaration mode):
   - Parses `<T any, K comparable>` — `Identifier Identifier` pairs as name + constraint.
   - Keep existing `parseTypeArguments` for instantiation (`<int32, utf8>`).
   - **Detection heuristic**: If first token after `<` is `Identifier` and next is also `Identifier` (a constraint name) or `>` or `,`, it's a declaration.

5. In transpiler, emit Go `ast.FieldList` for `TypeParams`:
   - On `goast.FuncType` → `TypeParams` field (Go 1.18+)
   - On `goast.TypeSpec` → `TypeParams` field (Go 1.18+)

### Files

- New `internal/types/typeparam.go`
- `internal/types/procedure.go` — add `TypeParams` field
- `internal/types/alias.go` — add `TypeParams` field
- `internal/parser/arguments.go` — add `parseTypeParams`
- `internal/transpiler/type.go` — generic FieldList emission
- `internal/transpiler/declaration.go` — TypeParams on TypeSpec / FuncType

---

## Phase 4: Generic Type Aliases

**Goal**: Support `Slice<T any> ~ []T`. *Depends on Phase 3.*

### Steps

1. In `findGlobalType` (`globals.go`): detect `<` after identifier before `~`. Parse type parameter list and store on the pre-registered alias symbol.

2. In `parseTypeAlias` (`type_alias.go`):
   - After consuming identifier, check for `<` → call `parseTypeParams`.
   - Push type params into enclosed scope (each param name resolves to its `TypeParam`).
   - Parse the body type — `T` resolves to the `TypeParam` in scope.
   - Store params on `ast.Type` node and `types.Alias`.
   - Pop scope.

3. **Instantiation** — when a generic alias is referenced with type arguments (`Slice<int32>`):
   - In `parseType`, detect `Identifier<...>` for a known generic alias.
   - Resolve alias, substitute each `TypeParam` with the corresponding concrete type argument.
   - Produce a concrete instantiated type (e.g., `[]int32`).
   - Cache instantiations by (alias + arg types) to avoid duplicates.

4. **Transpilation**:
   - Generic alias declaration: `type Slice[T any] []T`
   - Instantiation `Slice<int32>` → `Slice[int32]`

### EBNF Update

```ebnf
type_alias
    = IDENTIFIER, [ "<", type_param_list, ">" ], "~", combined_type;

type_param_list
    = type_param, { ",", type_param };

type_param
    = IDENTIFIER, IDENTIFIER;   (* name, constraint *)
```

### Files

- `internal/parser/type_alias.go`
- `internal/parser/globals.go` — `findGlobalType`
- `internal/parser/type.go` — instantiation in `parseType`
- `internal/transpiler/declaration.go` — generic type declaration
- `internal/transpiler/type.go` — instantiation references
- `cog.ebnf`

### Tests

- `Slice<T any> ~ []T` → parses, TypeParams = [{T, any}]
- `Pair<A any, B any> ~ A & B` → parses, TypeParams = [{A, any}, {B, any}]
- `x : Slice<int32>` → resolves to `[]int32`
- `m : Pair<utf8, uint64>` → resolves to `utf8 & uint64`
- Error: wrong number of type args
- Error: type arg doesn't satisfy constraint

---

## Phase 5: Generic Procedures / Functions

**Goal**: Support generic proc/func definitions and calls. *Depends on Phase 3. Parallel with Phase 4.*

### Steps

1. **Declaration syntax**: `identity<T any> : func(x : T) T = { return x }`
   - In `findGlobalDecl` (`globals.go`): detect `<` after identifier before `:`. Parse type params and store on symbol.
   - In declaration parsing: after identifier, check for `<` → call `parseTypeParams`. Push params into scope for parameter and return type parsing. Store on `types.Procedure.TypeParams`.

2. **Call syntax**: `identity<int32>(42)`
   - In identifier statement / expression handling: after identifier, if `<` and identifier resolves to a generic procedure → parse as type arguments, then parse `(` call.
   - **`<` ambiguity** (less-than vs type-args): Resolve by checking if the identifier is a known generic symbol with `TypeParams`. If yes → type args. If no → comparison.
   - Validate each type argument satisfies its constraint via `Satisfies`.
   - Substitute type params in procedure's parameter/return types for call type-checking.

3. **Type inference** (defer): Initially require explicit type arguments at all call sites.

4. **Transpilation**:
   - Declaration: `func identity[T any](x T) T { return x }`
   - Call: `identity[int32](42)`

### EBNF Update

```ebnf
identifier_statement
    = IDENTIFIER, [ "<", type_param_list, ">" ],
      ( "=", expression
      | ":", ( labeled_statement | typed_declaration )
      | ":=", expression
      | "~", combined_type
      | "(", call_arguments, ")" );
```

### Files

- `internal/parser/declaration.go`
- `internal/parser/statement.go`
- `internal/parser/expression.go` — generic call in expressions
- `internal/parser/call.go`
- `internal/parser/globals.go` — `findGlobalDecl`
- `internal/transpiler/declaration.go`
- `internal/transpiler/expression.go` — generic call emission
- `cog.ebnf`

### Tests

- Generic func: `identity<T any> : func(x : T) T = { return x }`
- Generic call: `identity<int32>(42)`
- Constraint violation error
- Multiple type params: `pair<A any, B any> : func(a : A, b : B) A & B`
- Error: missing type args on generic call
- Error: wrong number of type args

---

## Phase 6 (Optional): Interfaces

**Goal**: Basic interface types for method-set contracts. Enables `match` on interface subjects and richer constraint expressions.

### Steps

1. Add `InterfaceKind` to `types.Kind`.
2. Add `types.Interface` struct: `Methods []MethodSignature` (Name, Params, Returns).
3. Add `interface` keyword token.
4. Parse interface type declarations: `Stringer ~ interface { String : func() utf8 }`
5. Transpile to Go `interface{ String() string }`.
6. Support interface as constraint in generic parameters.
7. Support `match` on interface subjects → Go `switch t := x.(type)`.

**Not required** for `match` on union types.

---

## Verification (all phases)

1. `task test` - tests
5. `task run FILE=example` — integration

## Key Design Decisions

1. **`any` is a first-class type** with `AnyKind`. Transpiles to Go `any`.
2. **Constraints are types**: `Generic` already implements `types.Type`. New `TypeParam` wraps name + constraint.
3. **`TypeParam` ≠ `Generic`**: `Generic` = constraint family, `TypeParam` = named placeholder.
4. **Monomorphization at type level**: Instantiation substitutes type params with concrete types. Go output uses Go generics brackets `[T any]`.
5. **`<` ambiguity**: Resolved by identifier lookup — if it has `TypeParams`, parse `<` as type args.
6. **No type inference initially**: Require explicit type arguments. Inference is a follow-up.
7. **Library-type constraints**: Emit `any` as Go constraint for `float16`/`complex32`/`int128`/`uint128`. Cog compile-time `Satisfies` ensures correctness.
