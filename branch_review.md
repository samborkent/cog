# Code Review: `feature/change-cast-signature`

**Branch commits:**
- `refactor: change @cast signature` — `@cast` now returns `cog.Option[T]`, narrowing returns `None` instead of being rejected
- `feat: @as builtin` — new `@as<T>` builtin for semantic value-preserving conversions

**Test status:** 1232/1232 passing, `go vet` clean

---

## Critical Issues

### 1. Float32 → integer conversion will fail at Go compile time

**File:** `internal/transpiler/builtin.go`, lines 822–864

`math.Trunc` and `math.IsNaN` both require `float64` arguments. When `srcType` is `float32`, the generated code passes a `float32` expression directly, causing Go compilation errors.

**Location:**
- Line 836: `math.IsNaN(arg)` — arg is `float32`
- Line 844: `math.Trunc(arg)` — arg is `float32`

**Fix:** Wrap `arg` in `float64()` before passing to math functions when source kind is `Float32`:

```go
truncArg := arg
if srcKind == types.Float32 {
    truncArg = &goast.CallExpr{Fun: &goast.Ident{Name: "float64"}, Args: []goast.Expr{arg}}
}
```

### 2. IIFE generation produces non-idiomatic Go

**File:** `internal/transpiler/builtin.go` — 4 locations:
- Float-to-integer (line 822)
- Complex-to-complex (line 877)
- Complex-to-number (line 922)
- Narrowing integer (line 975)

The transpiler generates immediately-invoked function expressions (`func() T { ... }()`) for overflow/validity checks. This is correct but produces verbose Go output. Consider moving these patterns into the `builtin/` runtime package as reusable helpers.

---

## Warnings

### 3. Missing test: float32 → integer

No test case covers `@as<float32>` to an integer type, which would catch the `math.Trunc` compilation error. Add a test in `internal/transpiler/builtin_test.go`.

### 4. Missing test: cross-family narrowing in `@as`

Tests cover `int64 → int8` narrowing but not:
- `uint64 → uint8`
- `int64 → uint8` (signed-to-unsigned narrowing)
- `int32 → uint8`

### 5. Missing test: explicit widening `Some`

`@cast` widening returns `cog.Option[T]{..., Set: true}`. While individual tests check for `Set: true` in their output patterns, a dedicated test confirming widening always returns `Some` would be valuable.

---

## Info / Minor

### 6. Duplicate double-wrapping skip logic in `statement.go`

**File:** `internal/transpiler/statement.go`, lines 61–77 and 148–160

The same `@cast` skip pattern is duplicated in both the `Assignment` and `Declaration` cases. Extract into a helper method:

```go
func (t *Transpiler) wrapOptionIfNotCast(expr goast.Expr, assignmentExpr ast.Expr, optionType goast.Expr) goast.Expr {
    if builtin, ok := assignmentExpr.(*ast.Builtin); ok && builtin.Name == "cast" {
        return expr
    }
    return &goast.CompositeLit{
        Type: optionType,
        Elts: []goast.Expr{
            &goast.KeyValueExpr{Key: &goast.Ident{Name: "Value"}, Value: expr},
            &goast.KeyValueExpr{Key: &goast.Ident{Name: "Set"}, Value: &goast.Ident{Name: "true"}},
        },
    }
}
```

### 7. `parseBuiltinAs` doesn't validate against `tokenType` context

**File:** `internal/parser/builtins.go`, line 462

Unlike `parseBuiltinCast`, `parseBuiltinAs` doesn't validate `targetType` against the `tokenType` context parameter. Add the same check pattern used in `parseBuiltinIf` (lines 28–32).

### 8. Missing `@as` support for 128-bit types

`convertAs` doesn't handle:
- `int128`/`uint128` ↔ string
- `float128` ↔ string (if exists)
- Complex types ↔ string

Likely edge cases, but should be documented or supported to avoid silent zero-value returns.

### 9. Example file formatting issue (pre-existing)

**File:** `example/example.cog`, line 506:

```
BarRef : &ascii = &"hello"    _ = FooRef
```

Two statements on the same line without a separator.