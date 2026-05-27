# Plan: Change @cast to Return Option Type and Reject ascii→utf8

## Summary

Change `@cast<B, A any>(a A) B` to `@cast<B, A any>(a A) B?`, where the option returns `Some(value)` when the cast succeeds (same-size or widening) and `None` when narrowing (src > dst). Remove the existing `ascii` → `utf8` special case entirely — `ascii` (Go `[]byte`) and `utf8` (Go `string`) have different memory layouts and should not be bitcastable. A future `@as` builtin will handle best-effort type conversions where memory layouts differ.

## Files to Modify

### 1. `internal/parser/builtins.go` — `parseBuiltinCast`

**Changes:**
- Remove the `ascii` → `utf8` special case (lines 433-436). `ascii` (Go `[]byte`) and `utf8` (Go `string`) have different memory layouts — this cast was semantically incorrect for a bitcast operation. Removing this case means the cast falls through to the existing "reject string casts" check (lines 444-448), which correctly rejects any `@cast` involving string types.
- Change the return type from bare `targetType` to `&types.Option{Value: targetType}`. This wraps the target in an option type, matching the `B?` signature.
- The narrowing check (srcBits > dstBits) on line 468 is no longer a compile-time error — it becomes a runtime-returned `None`. **Remove** this error and instead let it pass through. The transpiler will generate the `None` path at runtime.

### 2. `internal/transpiler/builtin.go` — `convertBuiltin` and `convertCast`

**Changes:**
- In `convertBuiltin` (the `case BuiltinCast` branch, ~line 239), instead of returning the raw cast expression, wrap it in an option: return `component.BuiltinSome(castExpr)` for same-size casts, and `component.BuiltinNone[targetGoType]()` for narrowing (srcBits > dstBits). Actually, since we want to keep the static check for size in the transpiler (sizes are known at compile time), we can generate the appropriate Go code:
  - If srcBits == dstBits: generate `builtin.Some(result)` 
  - If srcBits < dstBits: generate `builtin.Some(result)` (promotion is always safe)
  - If srcBits > dstBits: generate `builtin.None[T]()` (narrowing always fails)
  
  Wait — re-reading the requirement: "@cast can only succeed between types with the same memory layout, and if the memory size of the starting type is smaller or equal to the size of the returning type." So narrowing (srcBits > dstBits) is an error. Currently it's rejected at parse time. Now with option return type:
  - Narrowing (src > dst): return `None` at runtime
  - Same-size or widening (src <= dst): return `Some(result)` — the current behavior

  Since sizes are known at compile time, we should generate the correct Go code branch:
  ```go
  // srcBits > dstBits
  func() T { var zero T; return zero }()  // none, zero value
  ```
  
  Actually, looking at how `builtin` package works — let me check if there's already a `Some`/`None` pattern.

- Remove the `ascii` → `utf8` special case in `convertCast` (lines 269-275).

### 3. `internal/transpiler/component/` — Option helper functions

Check if existing built-in `Some`/`None` component functions exist. If not, they may need to be added. Option types are represented at Go level. Need to understand how options are implemented at Go runtime level.

**Need to investigate:** How the cog runtime implements option types at the Go level. Look at `builtin/` directory for option/result runtime support.

### 4. Update parser tests in `internal/parser/builtins_test.go`

**Changes in `TestParseBuiltinCast`:**
- Remove or update the `ascii to utf8` test (line 310-322) — this should now be an error.
- Remove the `narrowing rejected` test (line 290-298) — narrowing is no longer a parse error since it returns `None` at runtime.
- Update test expectations: the return type of `@cast` expressions is now an option type, not a bare type. The parser tests that check `f.LenNodes() == 0` only check that parsing succeeds (no errors), so they should still pass. But verify this.

### 5. Update transpiler tests in `internal/transpiler/builtin_test.go`

**Changes in `TestConvertBuiltinCast`:**
- All existing tests need updating because `@cast` now returns `B?` (an option type). The generated Go code will wrap results in `builtin.Some(...)`.
- Add a test case for narrowing that generates `builtin.None[T]()`.
- Remove the `ascii to utf8` transpiler test if it exists (currently there's none in the transpiler test file, only in the parser test file).

### 6. `internal/parser/builtins.go` — Verify the narrowing check removal

The narrowing check on line 468 currently emits a compile error. With option return, this becomes a runtime `None`. Remove the error but keep the validation as documentation/comment only.

## Investigation Required Before Implementation

1. **Runtime option representation:** Check `builtin/` directory for how option types are represented at Go level. Look for files like `builtin/option.go`, `builtin/result.go`, `builtin/if.go`. Understand how `Some`/`None` are generated in Go output.
2. **Transpiler option support:** Check how the transpiler handles option types in its type conversion (`convertType` in `internal/transpiler/type.go`). The `Option` type needs to map to the correct Go representation.
3. **Existing `component.BuiltinSome`/`BuiltinNone`:** Search for existing helper functions for option types in the component package.

## Execution Order

1. Investigate runtime option representation and transpiler option support
2. Modify `internal/parser/builtins.go`:
   - Remove ascii→utf8 special case
   - Remove narrowing error, let it pass through
   - Change return type to `&types.Option{Value: targetType}`
3. Modify `internal/transpiler/builtin.go`:
   - Remove ascii→utf8 special case in `convertCast`
   - Wrap result in `Some()` for valid casts or `None()` for narrowing
4. Update parser tests
5. Update transpiler tests
6. Run `GOEXPERIMENT=arenas rtk go vet ./...` and `GOEXPERIMENT=arenas rtk go test ./...`

## Future Work (Out of Scope)

After this change, implement `@as<B, A any>(a A) B?` — a best-effort type conversion builtin. Unlike `@cast` (which requires identical memory layout and only fails on narrowing), `@as` can handle cross-representation conversions like `ascii` ↔ `utf8`, string↔number parsing, etc. This is a separate task and not part of this plan.