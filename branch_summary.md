# Branch: `refactor/type-alias`

## Goal

Consolidate type representation so that `types.Alias` is the canonical container for all named types (structs, enums, interfaces, type aliases, generic aliases, forward references). Remove the redundant `ast.Identifier`-as-type pattern and the `ast.Type.Identifier`/`ast.Type.TypeParameters` fields, replacing them with an embedded `*types.Alias`. This makes types first-class objects in the symbol table with their own `*isync.Map[string, *ast.Type]`, separate from variable identifiers.

## What Changed (52 files, +1047/-1070)

### Core architecture (committed)

**`internal/ast/type.go`** — `ast.Type` now embeds `*types.Alias` directly instead of `Identifier`, `TypeParameters`, and `Alias` as separate fields. The `NewType` constructor packs the alias fields into a `types.Alias` struct.

**`internal/ast/method.go`** — `ast.Method` gains `ReceiverType types.Type` and `Body ExprIndex` (a `ProcedureLiteral`), replaces `Declaration NodeIndex` with the body expression. `Receiver` renamed to `ReceiverIdent` for clarity.

**`internal/parser/symbol_table.go`** — Split into two maps:
- `idents` (`*isync.Map[string, Symbol]`) — variables, parameters, loop vars
- `types` (`*isync.Map[string, *ast.Type]`) — type declarations

New methods: `DefineType`, `DefineGlobalType`, `ResolveType`, `DefineIdent`, `DefineGlobalIdent`, `ResolveIdent`, `FillExports`. Removed `DefineGlobal`, `Define`, `Resolve`, `ForEachGlobal`, `DefineMethod`, `ResolveField`. The old `Resolve`/`Define` were conflating variables and types under one map keyed by `*ast.Identifier`.

**`internal/parser/method.go`** — Major rewrite:
- Uses `ResolveType`/`DefineType` instead of `Resolve`/`DefineGlobal` for receiver lookup
- Calls `types.Alias.RegisterMethods()` instead of the old `DefineMethod` + `Struct.Methods` append
- Receiver variable type is set from the resolved type alias directly (`receiverVarType = receiverType.Alias`)
- Duplicate method detection uses `receiverType.Alias.Method(name)` instead of `definedMethods` map
- Parses full `proc`/`func` type + body instead of delegating to `parseTypedDeclaration`

**`internal/parser/ownership.go`** — Updated all `Resolve` calls to `ResolveIdent`, `Define` → `DefineIdent`.

**`internal/parser/declaration.go`** — Updated symbol table calls, removed `QualifierMethod` special-case gate on `DefineIdent`.

**`internal/parser/parser.go`** — Removed `definedMethods` map field. Interface methods from constraints now register via `tp.RegisterMethods(iface.Methods...)` instead of per-method `DefineMethod` calls. `DefineIdent` replaces all `Define` calls. Template param definitions use `DefineType` not `Define`.

**`internal/types/alias.go`** — Added `methods []*Method`, `lazy func() Type`, `NewForwardAlias`, `Methods()`, `Method(name)`, `RegisterMethods(...)`, `ensureResolved()`, `IsTypeParam()`, `ConstraintString()`, `SatisfiedBy(concrete)`. The lazy resolver pattern enables forward type references.

**`internal/types/enum.go`**, **`interface.go`**, **`struct.go`** — Added `Methods()` accessor on `Interface` for compatibility with the new method registration path.

**`internal/parser/type_alias.go`** — Type alias parsing now constructs `ast.Type` with embedded `*types.Alias` directly (not via separate fields).

**`internal/parser/type.go`** — Procedure type parsing updated for the new receiver type handling.

**`cmd/main.go`** — Removed `populateImportExports` helper function; replaced with `symbols.FillExports(imp)`.

**`types.go`** — `Either` takes simplified generic `[L, R any]` form, added `NewEither`/`NewOr` constructors.

### Other notable changes

- `internal/sync/` → `internal/isync/` (rename, `map.go` updated)
- Tokens: new `type.go` with type-related token definitions
- New test file: `example/globals.cogs`
- README updated with script-mode constraint note and `.cog`/`.cogs` entry-point design idea

## Current State

- **All `types` package tests pass** (the type system is in good shape)
- **`go vet` clean** — no compile errors
- **956 tests pass, 94 fail**
- **Failures fall into categories:**

| Category | Count | Root Cause |
|---|---|---|
| Switch/ident/bool parsing | ~15 | `parseStatement` rejecting `case` token — related to `case` keyword handling regression |
| Method transpilation | ~8 | `ast.Method` struct changed; transpiler not updated for new `Body`/`Type` fields |
| Generic match | ~10 | Match generic type assertion regression |
| Type alias (option, forward ref, struct literal) | ~8 | Type alias parsing changed; some test inputs produce different errors or no longer parse |
| Cross-file / multi-file / imports | ~8 | Old `Resolve`/`Define` API removed; some code paths still use old patterns |
| ConvertExpr / ConvertStmt / ConvertType | ~20 | Various expression/statement visitors not updated for new `ast.Type` and `ast.Method` structure |
| Integration tests (ArithmeticExpressions) | 1 | Process killed (OOM/timeout) — possibly pre-existing |

## What's Left To Do

### High priority (blocking compilation)
1. **Fix statement parsing** — `parseStatement` doesn't handle `switch`/`case` keywords (likely missing `case` in a token dispatch or the switch entry point changed)
2. **Update transpiler for new `ast.Method`** — `transpiler/statement.go`, `transpiler/method.go` still reference `Declaration NodeIndex` and old `Method.Type`; need to read `Type` as `*types.Procedure` and `Body` as the procedure literal expression
3. **Fix type alias forward references in parser** — `TestDefineGlobalResolvesForwardStub` fails because `makeType` helper produces an `*ast.Type` whose alias is not properly compared during `DefineGlobalType` stub resolution (checks `types.IsNone(existing.Alias.Derived)` but the stub may have `Derived == nil` instead of `types.None`)
4. **Switch statement token dispatch** — `switch` keyword not being recognized in the statement parsing path; check `parseSwitch` entry or token type matching

### Medium priority
5. **Update convert/transpile visitors** — `ConvertExpr`, `ConvertStmt`, `ConvertType` in `internal/transpiler/component/` need updates for the new `ast.Type` and `ast.Method` shapes
6. **Generic match refinement** — match-on-generic-type tests failing; likely the `any` constraint or type-switch path
7. **Cross-file/multi-file import compatibility** — verify `FillExports` and the new `ResolveType`/`ResolveIdent` split work for imported packages

### Low priority / cleanup
8. **Script mode tests** — `TestScriptSwitchStatement` and `TestScriptTranspileSwitch` failing due to switch parsing
9. **Integration tests** — `TestArithmeticExpressions` OOM (check if this is pre-existing)
10. **Check unused/dead code** — `definedMethods` removed, ensure nothing else was orphaned
11. **Method/interface tests** — `interface_as_constraint` and `this_field_access` failures suggest the new method attachment path needs edge-case handling for untyped receivers

## How to Approach

1. **Fix switch/case parsing first** — one root cause likely fixing ~15 tests
2. **Fix the transpiler `Method` handling** — another ~10+ tests
3. **Fix forward reference stub resolution** — gate on `Derived == nil` vs `IsNone`
4. **Fix the remaining convert/visitor patterns** — bulk update for new struct shapes
5. **Fix individual regressions** — method tests, generic match, cross-file
6. **Verify with full test suite and integration tests**