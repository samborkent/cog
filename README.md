# cog

cog is a Go-based hobby programming language that brings some additional features. It is wholly incomplete and a work-in-progress.

The following basic features are missing that need to be implemented before Cog can be used to write useful programs:

- for-loops
- slices, arrays
- Go-to-Cog type conversions
- Multi-file programs
- Cog packages / imports

## Features

### Implemented

- Refined syntax
    - Function declaration `main : proc(ctx : context) = { ... }`
    - Typed variable declaration `foo : uint64 = 10`
    - Type alias `String ~ string`
- Immutability by default
- Type qualifiers
    - `var` for mutable variables. Not allowed in package scope.
    - `dyn` for dynamically scoped variables. Only allowed in package scope.
- Extended types
    - Enum `enum[any]`
    - Map `map[comparable]any`
    - Set `set[comparable]` (alias for `map[comparable]struct{}`)
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
- Break from if-statements
- Distinction between `func` and `proc`
    - `func` is a function without any side-effects with at least 1 return value.
        - It cannot reference dynamically scoped variables.
        - `func` cannot be called async
    - `proc` is a function that may have side-effects, where return values are optional.
        - It can reference dynamically scoped variables.
        - `proc` may be called async.
- Optional function parameters `foo(optional :? utf8)`
    - With default values `foo(default :? utf8 = 10)`
- Value switch
    - `switch var { case val: ... }`

### Planned

- Explicit exports using `export`
- Type qualifiers
    - `comp` for compile time constants. Similar to Zig' `comptime`. When used on variables, like C++ `constexpr`, when used for functions like C++ `consteval`.
- Variables need to be passed to scope explicitely (no catch all closures)
    - `(foo, bar) { // foo & bar are available in this scope }
- Additional safety regarding mutability and ownership.
- Type switch
    - `switch t { type uint64: ... }`
    - For `t ~ any | interface | union`
- Select statement
- Generics with additional builtin generic constraints:
    - `string ~ ascii | utf8`
    - `int ~ int8 | int16 | int32 | int64 | int128`
    - `uint ~ uint8 | uint16 | uint32 | uint64 | uint128`
    - `float ~ float16 | float32 | float64`
    - `complex ~ complex32 | complex64 | complex128`
    - `signed ~ int | float | complex`
    - `number ~ signed | uint`
- Allocation builtins:
    - `@ptr[T valueType]() *T`
    - `@slice[T any, I uint](len : I, cap :? I = len) []T`
    - `@map[K comparable, V any, I uint](cap :? I = 8) map[K]V`
    - `@set[K comparable, I uint](cap :? I = 8) set[K]`
    - `@cast[A, B any](x A) B` to cast types instead of `float32()`, etc.
        - It must panic if casting cannot be done without overflow or precision loss.
        - Also implement `@convert[A, B any](x A) B`, which will perform best-effort conversion, allowing some precision loss and handling overflows.
- Additional types:
    - `int128` (usine [github.com/ryanavella/wide](github.com/ryanavella/wide))
    - `uint128` (using [lukechampine.com/uint128](lukechampine.com/uint128))
    - `float16` (using [github.com/x448/float16](github.com/x448/float16))
    - `complex32` (using `float16`)
    - `ascii` string where every character is a single byte
    - `utf8` alias for Go `string`
    - `signal[T any]` alias of `chan[T any]struct{}`
    - `any!` result type (alias of `any | error`)
        - Error can be extracted with `err!`
        - E.g `res := someFunc(); if res! { @print(res) }`
- Arena based allocations (using `arena` experiment)
    - Allocations are handled through an arena contained within `context`.
    - A new arena is created when entering a new `proc` scope.
    - Arena is cleared when leaving `proc` scope.\
- Builtin operations for 2D / 3D / 4D slices.
- Builtin `upx` binary packer for smaller binaries.
- Script mode
    - Files without package declaration will be parsed as script.
    - This is basically just a `main` package, and script gets inserted in `main()` body.
- Canonical syntax highlighting
- LSP

## Syntax

### Operators

* `:` - declare a value identifier with type
* `=` - assign a value to a value identifier
* `:=` - short hand for `: <inferred type> =`
* `~` - declare a type alias

## TODO

### Short-term

- Get rid of `Go()` methods in `ast`. Instead create functions in `transpiler/comp` package.
- Handle set type parsing in `parseType`.
- Refactor `parseTypedDeclaration` to use same logic as `parseCombinedType`
- Fix global type definition ordering bug for complex type (e.g. enum[planet] before planet)

### Long-term

- TESTS!
- Audit all uses of `types.Underlying().Kind()`
- Allow package-less files (scripts)
    - These files cannot be imported, and will be excuted as if wrapped in a main function.
    - Should `ctx` be predefined in a script?
- Disallow `main` in declarations besides `main : proc(ctx: context)`.
- Fork and rework float16, uint128 and int128 imported packages.

## Example code

```go
package main

goimport (
    "strings"
)

a : int64 = 0

export isExported := true
NotExported := true

String ~ utf8
export notExported ~ uint64
export ExportedString ~ String

main : proc() = {
    str := @go.strings.ToUpper("str")

    b : float32 = 0.0

    var language := "cog" // utf8
    var lang : utf8 = "cog"
    var lng : ascii = "cog"

    leeng := lng
    c1 := `hello
    
        world`
    c2 := "hello\n\n\tworld"

    @print(c1)
    @print(c2)

    language = "go"

    if true {
        break
    } else {
        lng = "else"
    }

    if true != false {}
    if true == false && true != false {}
    if (language == "cog") != (lng == "cog") {}
    if 5 <= 6 {}
    if !true {}

    fl := -0.6e-7

    collection : set[string] = { "hello1", "hello2" }

    maths := 5 * 6 / (2 + 3)

ifLabel:
    if true {
        if true {
            break ifLabel
        }
    }

    newString := definedHere

    newLang := @if(language == "cog", 25 + 10 - 6, 5)

    earth : planet = {
        radius = 10,
        mass = 20,
    }

    _ = earth.radius

caseSwitch:
    switch {
    case 5e-6 <= 6:
        break
    case 5 >= 0.6:
        break caseSwitch
    default:
        lang = "foo"
    }

    switch language {
    case "en":
    case "nl":
    default:
    }

    enum1 := Status.Open
    enum2 := Status.Closed

    if enum1 == enum2 {
        @print(enum1)
    }

    tuple : Tuple = {"hello", 10, false}

    var utf : utf8?  = "hello"
    // option : Option? // not allowed
    utf = "option"
    
    if utf? {
        @print("hello")
    }
    
    var option : uint64?
    
    if option? {
        @print("do not print")
    }

    option = 10

    if option? {
        @print("do print")
    }

    upperCaseString := upper(language)
    @print(upperCaseString)

    @print(upper("foo", "bar"))

    // _ = Planets.Earth.mass

    someFunc("")
    @print(val) // default
}

definedHere := "defined globally!"

planet ~ struct {
    name : ascii

    export pressure : float64

    export (
        radius : float64
        mass : float64
    )
}

Status ~ enum[utf8] {
    Open := "open",
    Closed := "closed",
}

Planets ~ enum[planet] {
    Earth := {
        radius = 0.5,
        mass = 0.1,
    },
}

Tuple ~ utf8 & uint64 & bool

Either ~ utf8 | uint64

Option ~ utf8?

upper : func(str : utf8, optional? : utf8, alsoOptional? : utf8 = "wassup") utf8 = {
    return @go.strings.ToUpper(str) + optional + alsoOptional
}

dyn val : utf8 = "default"
dyn other : uint64 // valid, will have zero value as default

someFunc : proc(str : utf8) = {
    @print(val) // default
    val = "overwrite"
    @print(val) // overwrite
}
```
