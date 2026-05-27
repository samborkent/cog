# Plan: Implement `@as<B, A any>(a A) B` Builtin

## Overview

Add a new `@as` builtin to Cog that performs **best-effort type conversions** (semantic value conversion, unlike `@cast`'s bitwise reinterpretation). It is a **compile-time code generation** builtin in the transpiler — no runtime dispatch/reflection, zero overhead.

## Conversion Rules

| Source → Target | Result |
|---|---|
| `bool` → any numeric | `0` or `1` |
| numeric → `bool` | `false` if zero/NaN, `true` otherwise |
| `true` / `false` → string (ascii/utf8) | `"true"` / `"false"` |
| integer ↔ integer (same or wider target) | direct numeric conversion (always safe) |
| integer → narrower integer | `0` if overflow (value != `T(value)`), else `T(value)` |
| float → integer | `0` if NaN, infinity, or fraction, else `int(value)` |
| integer → float | direct numeric conversion (always safe) |
| float → float (same or wider) | direct numeric conversion |
| float → narrower float | direct Go conversion (may be ±Inf) |
| string (ascii/utf8) → integer | `strconv.ParseInt(s, 10, 64)` → `0` on error, truncated to target size |
| string (ascii/utf8) → float | `strconv.ParseFloat(s, bits)` → `0.0` on error |
| string (ascii/utf8) → bool | `strconv.ParseBool(s)` → `false` on error |
| integer/float/bool → string (ascii/utf8) | `strconv.FormatInt`, `strconv.FormatFloat`, `strconv.FormatBool` — wrap in `cog.ASCII(...)` for ascii target |
| ascii ↔ utf8 | same-string passthrough (both are Go `string` under the hood — `cog.ASCII` / `string`) |
| complex with non-zero imag → anything | return zero value of `B` |
| complex with zero imag → number | extract real component as the target number type |
| number → complex | fill real component, set imag to zero |
| anything → same type | passthrough (identity) |
| unsupported pair | zero value of `B` (e.g., struct → int) |

## Files to Modify

### 1. `internal/parser/parser.go` (line ~291)

Register the parser:

```go
"as":    p.parseBuiltinAs,
```

### 2. `internal/parser/builtins.go` — new `parseBuiltinAs` function

- Signature: `@as<B, A any>(a A) B`
- Requires 1 or 2 type arguments: target type `B`, optional source type `A`
- Validates both types are `IsBasic` or `IsString`
- Returns `p.ast.NewBuiltin(t, "as", typArgs, ...)` with `ReturnType = targetType`

### 3. `internal/transpiler/builtin.go`

- Add `BuiltinAs Builtins = "as"` constant
- Add `case BuiltinAs:` in `convertBuiltin`
  - Convert argument expression
  - Call a new `convertAs(arg, srcKind, dstKind, srcType, dstType)` method
  - No Option wrapping — `@as` returns `B` directly

### 4. `internal/transpiler/builtin.go` — new `convertAs` method

Generate Go AST for the conversion based on (srcKind, dstKind). Patterns:

- **same type**: return arg directly
- **bool ↔ any numeric (non-complex)**: `builtin.If[T](arg, 1, 0)` / `arg != 0`
- **bool → string**: `builtin.If[string](arg, "true", "false")`
- **string → bool**: `go_strconv.ParseBool(arg)` — error is ignored, zero used
- **string → int**: `go_strconv.ParseInt(arg, 10, 64)` then truncate to target
- **string → float**: `go_strconv.ParseFloat(arg, bits)`
- **int → string**: `go_strconv.FormatInt(int64(arg), 10)`
- **float → string**: `go_strconv.FormatFloat(arg, 'f', -1, bits)`
- **numeric widening**: direct Go type conversion
- **numeric narrowing (integer)**: assign then compare — if `T(v) != v` then `0` else `T(v)`
- **float → integer**: `if go_math.IsNaN(arg) || arg != go_math.Trunc(arg) { 0 } else { T(arg) }`
- **numeric → narrower float**: direct Go conversion
- **ascii ↔ utf8**: cast to target Go type (both are string, but ascii needs `cog.ASCII(...)` wrapping when going from utf8 → ascii)
- **complex → number**: extract real with `real(v)` then convert to target type. If imaginary component is non-zero (`imag(v) != 0`), return zero of target instead.
- **number → complex**: `complex(T(v), 0)` — fill real, imag zero
- **complex → complex**: extract real via `real(v)`, check `imag(v) == 0` for narrowing (else zero), then convert
- **unsupported**: zero value (e.g., `T(0)`)
- Requires `addStdLibImport("strconv")` and/or `addStdLibImport("math")` as needed

### 5. `internal/transpiler/component/builtins.go` — helper

Add `BuiltinZero(targetType goast.Expr) goast.Expr` returning e.g. `*goast.CompositeLit{Type: targetType}` for complex types, or a `&goast.CallExpr{Fun: targetType, Args: []goast.Expr{&goast.BasicLit{...}}}` for numerics. Simplest: use Go ast that produces a typed zero literal.

### 6. `internal/parser/builtins_test.go` — parser tests

Test cases:
- `@as<int64>(42)` (valid identity)
- `@as<utf8>(42)` (int → string)
- `@as<int32, utf8>("42")` (string → int with explicit source type)
- `@as<bool>(1)` (int → bool)
- Wrong number of type args → error
- Complex source → parse error (or parse success, runtime zero)

### 7. `internal/transpiler/builtin_test.go` — transpiler tests

Test generated Go output patterns:
- `@as<int64, int8>(x)` → just `int64(x)` (widening)
- `@as<int8, int64>(x)` → overflow check pattern
- `@as<utf8>(true)` → `strconv.FormatBool` or `"true"/"false"`
- `@ascii>("hello") as utf8` → string identity
- `@utf8>("world") as ascii` → `cog.ASCII(...)` wrapping

### 8. `internal/transpiler/statement.go` — double-wrapping guard (lines ~61-76, ~148-159)

Currently guards `builtin.Name != "cast"`. Extend to also check `builtin.Name != "as"` since `@as` also returns a direct value (no Option wrapping needed — but actually `@as` returns a plain `B`, not an option, so the Option branch won't match. No change needed here since `@as` return type is `B` (plain), not `&types.Option{...}`).

## Verification

```bash
GOEXPERIMENT=arenas rtk go vet ./...
GOEXPERIMENT=arenas rtk go test -timeout=10s ./...
task run
task compile
task transpile
```

## Out of Scope

- `@as` does NOT accept container types, structs, enums, interfaces — only basic types + strings
- No runtime type dispatch — all conversion logic is generated as Go AST at compile time
- Overflow detection for narrowing is done via generated `T(v) != v` comparisons (relying on Go's truncation behavior)