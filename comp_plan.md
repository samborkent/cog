## Plan: Comp Qualifier + Comptime Functions

Implement in two phases: Phase 1 adds `comp` qualifier + compile-time value folding for variables (MVP). Phase 2 adds compile-time-evaluated functions (`func` only, no `proc`) and call-site folding. Keep parser/type rules strict, run evaluator before transpilation, and preserve source mapping + deterministic diagnostics.

**Steps**
1. Phase 0 - Syntax + AST plumbing (*blocks all later phases*):
1. Add token `Comp` to `internal/tokens/type.go` and string mapping.
2. Add keyword lookup entry for `comp` in `internal/tokens/lookup.go`.
3. Add qualifier enum member for comp (e.g., `QualifierCompileTime`) in `internal/ast/identifier.go`.
4. Update parser qualifier detection in `internal/parser/statement.go` so declarations after `comp` set the new qualifier.
5. Add/extend parser tests for `comp` variable declarations and `comp` function declarations.

2. Phase 1 - MVP compile-time variables (*depends on Step 1*):
1. Create package `internal/comptime` with a minimal evaluator API (entrypoint receives `*ast.File`, returns transformed file + diagnostics).
2. Implement value domain for MVP scalar literals (`bool`, `ascii`, `utf8`, ints, uints, floats; optional support for enum literals if trivial).
3. Implement evaluator for expression subset: literals, identifier references, unary prefix (`!`, `-`), infix arithmetic/comparison/logical, parenthesized/grouped expressions.
4. Implement declaration pass over package scope in source order:
5. If declaration qualifier is comp, require initializer.
6. Evaluate initializer using only already-known compile-time symbols.
7. Replace expression node with folded literal AST node.
8. Record symbol value for downstream references.
9. Reject non-foldable expressions with hard error (`compile-time value required for comp declaration`).
10. Wire pass into pipeline before transpilation:
11. CLI path in `cmd/main.go`.
12. Integration helper paths in `integration_test.go`.
13. Transpiler behavior:
14. In `internal/transpiler/declaration.go`, map comp declarations to `const` when representable in Go const context.
15. Fallback to `var` with folded literal initializer when type cannot be Go const (same `mustBeVariable` constraints).
16. Add MVP tests:
17. Parser: recognizes `comp` qualifier.
18. Comptime evaluator unit tests for folding and diagnostics.
19. Integration tests for emitted Go (`const` vs `var` fallback).

3. Phase 2 - Comptime functions (`func`-only) (*depends on Phase 1*):
1. Extend parser validation rules in `internal/parser/declaration.go` / type handling:
2. If qualifier is comp and value type is procedure: require `Procedure.Function == true`.
3. Reject `comp` + `proc`.
4. Require explicit return type for comp functions (non-nil return type).
5. Add function registry in `internal/comptime` keyed by fully-resolved symbol name.
6. Implement interpreter for comp function bodies (initial supported statement subset):
7. Local immutable declarations.
8. Return statements.
9. If/else branching.
10. Expression statements only if pure and value-discard-safe.
11. Implement call evaluation:
12. At expression traversal, detect call to comp function.
13. Evaluate all arguments to compile-time values.
14. Bind params + defaults in local environment.
15. Execute function body; capture return value.
16. Replace original call AST with folded literal/composite AST.
17. Enforce purity constraints during function validation + execution:
18. Disallow dyn variable reads/writes.
19. Disallow `proc` calls.
20. Disallow go interop calls (`@go.*`).
21. Disallow impure builtins (`@print`, allocators, etc.); allowlist pure builtins only (start with `@if` if compatible).
22. Add cycle/termination guards:
23. Max recursion depth.
24. Max evaluation steps.
25. Memoization cache for `(function, arg-values)`.
26. Add source-aware diagnostics for comptime failures (include cog filename/line from AST node positions).
27. Add function-focused tests:
28. Valid: nested comptime calls, default params, branch-based constant result.
29. Invalid: `comp proc`, side effects, dyn access, impure builtin call, non-constant argument.

4. Phase 3 - Transpiler integration hardening (*depends on Phases 1-2*):
1. Ensure transformed AST still works with line-directive/comment placeholder workflow in `internal/transpiler/print.go` + statement line mapping.
2. Verify folded expressions preserve expected types (especially option/enum/ascii).
3. Verify no behavior change for non-comp code paths.
4. Keep evaluator isolated from Go AST generation (operate on Cog AST only).

5. Phase 4 - Tooling + docs (*parallel with late Phase 3 where possible*):
1. Add `comp` to syntax highlighting grammar and keyword scope in `editors/vscode/syntaxes/cog.tmLanguage.json`.
2. Update docs in `README.md`:
3. Define `comp` variables and comp functions semantics.
4. Clarify `func` vs `proc` constraints for compile-time execution.
5. Add examples for folded variables and comptime function calls.

**Relevant files**
- `/Users/sam/git/cog/internal/tokens/type.go` - add token + string mapping for `comp`.
- `/Users/sam/git/cog/internal/tokens/lookup.go` - keyword map update.
- `/Users/sam/git/cog/internal/ast/identifier.go` - qualifier enum extension.
- `/Users/sam/git/cog/internal/parser/statement.go` - qualifier parsing (`var`/`dyn`/`comp`).
- `/Users/sam/git/cog/internal/parser/declaration.go` - comp declaration rules, comp func validation.
- `/Users/sam/git/cog/internal/parser/type.go` - enforce/validate procedure-type constraints for comp funcs if needed.
- `/Users/sam/git/cog/internal/comptime/*` - new evaluator + interpreter package.
- `/Users/sam/git/cog/internal/transpiler/declaration.go` - const/var emission for folded comp declarations.
- `/Users/sam/git/cog/internal/transpiler/expression.go` - ensure folded AST call results convert cleanly.
- `/Users/sam/git/cog/cmd/main.go` - invoke comptime pass in normal compile path.
- `/Users/sam/git/cog/integration_test.go` - ensure test pipeline includes comptime pass.
- `/Users/sam/git/cog/internal/parser/*_test.go` - parser coverage for comp syntax/rules.
- `/Users/sam/git/cog/internal/transpiler/*_test.go` - const/var output and type correctness.
- `/Users/sam/git/cog/README.md` - language semantics and examples.
- `/Users/sam/git/cog/editors/vscode/syntaxes/cog.tmLanguage.json` - highlight `comp` keyword.

**Verification**
1. Run `GOEXPERIMENT=arenas go test ./...` after each phase boundary.
2. Add table-driven evaluator tests for success/failure folding cases.
3. Add integration fixtures showing:
4. `comp` scalar declaration emits folded Go value.
5. `comp` function call folds at compile time.
6. Invalid comptime usage fails with deterministic error message and location.
7. Run `go run ./cmd -file example.cog` and inspect output for unchanged non-comp behavior.
8. Confirm line directives still compile and map errors to `.cog` lines.

**Decisions**
- Included scope (Phase 1): package-scope comp variables with strict compile-time fold requirement.
- Included scope (Phase 2): comp functions using `func` only, explicit return type required.
- Excluded from first implementation: full side-effect theorem proving, arbitrary loops in comptime interpreter, compile-time allocation builtins, go interop at comptime.
- Error policy: fail-fast compile errors when comptime evaluation cannot be proven or completed deterministically.

**Further Considerations**
1. Comptime loop support strategy:
Recommendation: defer loops until after step-limit and branch semantics are stable; start with if/return/declarations only.
2. Purity model implementation:
Recommendation: begin with explicit denylist + allowlist for builtins/calls, then tighten with AST-level static checks.
3. Type coverage expansion:
Recommendation: start with scalar values, add composite literals (`array`, `tuple`, `struct`, `set/map`) after scalar path is stable and tested.
