# cog

cog is a Go-based hobby programming language that brings some additional features. It is wholly incomplete and a work-in-progress.

The following basic features are missing that need to be implemented before Cog can be used to write useful programs:

- Function declarations
- Cog function calls (Go std lib function calls are working)
- Go-to-Cog type conversions
- Multi-file programs
- Cog packages / imports

## Features

### Implemented

- Refined syntax
    - Function declaration `main : proc(ctx : context) = { ... }`
    - Typed variable declaration `foo : uint64 = 10`
    - Type alias `String ~ string`
- Extended types
    - Enum `enum[any]`
    - Set `set[any]` (alias for `map[any]struct{}`)
    - Either `this | that`
    - Tuple `this, and, that`
    - Option `foo : uint64?; if (foo?) { ... }`
- Clear builtin functions with `@` prefix
    - `@print(msg any)` print to std out
    - `@if[T any](if : bool, then : T, else :? T)`
- `context` included as base type
- `main` takes a `ctx` argument to control lifetime of application
- Call Go std library functions
    - Import using `goimport`
    - Call using `@go` namespace prefix (e.g. `@go.strings.ToUpper("call me"))

### Planned

- Explicit exports using `export`
- Optional function parameters `foo(optional :? utf8)`
    - With default values `foo(default :? utf8 = 10)`
- Distinction between `func` and `proc`
    - `func` is a function without any side-effects with at least 1 return value. It cannot take a `context` argument.
    - `proc` is a function that may have side-effects, where return values are optional. It always takes `ctx` as first argument.
- Additional generic constraints:
    - `string ~ ascii | utf8`
    - `int ~ int8 | int16 | int32 | int64 | int128`
    - `uint ~ uint8 | uint16 | uint32 | uint64 | uint128`
    - `float ~ float16 | float32 | float64`
    - `complex ~ complex32 | complex64 | complex128`
    - `signed ~ int | float | complex`
    - `number ~ signed | uint`
- Allocation builtins:
    - `@ptr[T valueType]() *T`
    - `@slice[T any, I uint](ctx : context, len : I, cap :? I = len) []T`
    - `@map[K comparable, V any, I uint](ctx : context, cap :? I = 8) map[K]V`
    - `@set[K comparable, I uint](ctx : context, cap :? I = 8) set[K]`
- Additional types:
    - `int128`
    - `uint128` (using [lukechampine.com/uint128])
    - `float16` (using [github.com/x448/float16])
    - `complex32` (using `float16`)
    - `ascii` string where every character is a single byte
    - `utf8` alias for Go `string`
    - `signal[T any]` alias of `chan[T any]struct{}`
- Arena based allocations (using `arena` experiment)
    - Allocations are handled through an arena contained within `context`.
    - A new arena is created when entering a new `proc` scope.
    - Arena is cleared when leaving `proc` scope.
- Builtin `upx` binary packer for smaller binaries.

## Syntax

### Operators

* `:` - declare a value identifier with type
* `=` - assign a value to a value identifier
* `:=` - short hand for `: <inferred type> =`
* `~` - declare a type alias

## TODO

### Short-term

- Refactor `parseTypedDeclaration` to use same logic as `parseCombinedType`
- Fix global type definition ordering bug for complex type (e.g. enum[planet] before planet)
- Remove if-condition parentheses
- Change `and`/`or` and `&&`/`||`, remove `xor`.
- Handle set type parsing in `parseType`.

### Long-term

- Audit all uses of `types.Underlying().Kind()`
- Allow package-less files (scripts)
    - These files cannot be imported, and will be excuted as if wrapped in a main function.
    - Should `ctx` be predefined in a script?
- Disallow `main` in declarations besides `main : proc(ctx: context)`.
- Implement `@cast[A, B any](x A) B` to cast types instead of `float32()`, etc. It must panic if casting cannot be done without overflow or precision loss.
    - Also implement `@convert[A, B any](x A) B`, which will perform best-effort conversion, allowing some precision loss and handling overflows.
