# cog

cog is a Go-based hobby programming language that brings some additional features. It is wholly incomplete and a work-in-progress.

The following basic features are missing that need to be implemented before Cog can be used to write useful programs:

- Go-to-Cog type conversions

## TODO:

### Bugs
- When declaring type alias or method in script mode, the type gets placed in global scope, instead of inside of main.
    This is required for method declaration (may only be global in Go), so we need to manually disallow using a type or method which is only defined later in the file in script mode.

### Idea:
- `main : proc()` is no longer the entry point. `.cog` files cannot have an entry point. `.cogs` are only entry points of `.cog` programs. Compiler will only look for `.cogs` files and their imports.
- `export` is not allowed in `.cogs`.

### Features
- Disallow use of `@go` inside of `func`, as we cannot control what code is invoked, and `func` may not have side-effects.
- Implement monomorphization once Go 1.27 generics methods have been released.
- Remove `@ref` allocator.
- Design how iterators should work.
    - Range over int (or other literal) should not be possible.
    - Instead we should range over an iterator function which takes literal as argument.

## Features

### Implemented

- Refined syntax
    - Function declaration `main : proc() = { ... }`
    - Typed variable declaration `foo : uint64 = 10`
    - Type alias `String ~ utf8`
- Immutability by default
- `main` can only be declared as `proc()`
- Type qualifiers
    - `var` for mutable variables. Not allowed in package scope.
    - `dyn` for dynamically scoped variables. Only allowed in package scope.
- Extended types
    - Array `[const]uint64`
    - Slice `[]uint64`
    - Enum `enum<any>`
    - Map `map<comparable, any>`
    - Set `set<comparable>` (alias for `map<comparable, struct{}>`)
    - Either `this ^ that`
    - Tuple `(this, that, other)`
    - Option `foo : uint64?; if foo? { ... }`
    - Result `bar : int64 ! MyError; if bar? { use bar } if !bar? { handle bar! }`
    - `ascii` string where every character is a single byte
    - `utf8` alias for Go `string`
    - Struct with explicit field exports
    - Interface `Stringer ~ interface { String : func() utf8 }`
    - `int128` (using [github.com/ryanavella/wide](github.com/ryanavella/wide))
    - `uint128` (using [lukechampine.com/uint128](lukechampine.com/uint128))
    - `float16` (using [github.com/x448/float16](github.com/x448/float16))
    - `complex32` (using `float16`)
    - Type constraints `String ~ utf8 | ascii`
- Typed composite literals: `[]int8{5, 4, 3}`, `[5]int8{...}`, `map<ascii, int8>{...}`, `set<ascii>{...}`
- Clear builtin functions with `@` prefix
    - `@print(msg any)` print to std out
    - `@if<T ~ any>(if : bool, then : T, else :? T)` conditional expression
    - `@cast<B, A ~ any>(x A) B?` bitwise type cast, returns `value` on success (same size or widening) and `None` on narrowing (src > dst)
    - `@as<B, A any>(x A) B` best-effort type conversion (semantic, not bitwise):
        - bool ↔ numeric, bool ↔ string, integer ↔ string, float ↔ string
        - ascii ↔ utf8, int128/uint128 ↔ string (via `.String()`)
        - float16 ↔ string (via `.String()`), complex32 ↔ complex64/complex128/number/string
        - narrowing with overflow detection, NaN/Inf/fraction → 0, complex imag→zero
- Allocation builtins with generic type arguments:
    - `@ref<T valueType>() &T`
    - `@slice<T any, I uint>(len : I, cap :? I = len) []T`
    - `@map<K comparable, V any, I uint>(cap :? I = 8) map<K, V>`
    - `@set<K comparable, I uint>(cap :? I = 8) set<K>`
- Call Go std library functions
    - Import using `goimport`
    - Call using `@go` namespace prefix (e.g. `@go.strings.ToUpper("call me")`)
- Break from if-statements
- Labeled control flow (`break label`, `continue label`)
- Distinction between `func` and `proc`
    - `func` is a function without any side-effects with at least 1 return value.
        - It cannot reference dynamically scoped variables.
        - `func` cannot be called async.
    - `proc` is a function that may have side-effects, where return values are optional.
        - It can reference dynamically scoped variables.
        - `proc` may be called async.
    - Context is only injected into `main` when the program uses procedures or dynamic variables.
- Optional function parameters `foo(optional? : utf8)`
    - With default values `foo(default? : utf8 = "wassup")`
- Value switch
    - `switch val { case expr: ... }`
- Conditional switch
    - `switch { case expr: ... }`
- For-loops
    - Infinite loop: `for { ... }`
    - Container loop: `for container { ... }`
    - Range with `in`: `for v, k in container { ... }`
    - Loop over string, slice, array, map, and set.
- Automatic arena based allocations (using `arena` experiment)
- Multi-file support
- Explicit exports using `export`
- Local package imports
    - Import using `import`
    - Access exported symbols with package selector (e.g. `geom.Distance(a, b)`)
- Script mode (`.cogs` files)
    - No package declaration needed
    - No `export` keyword allowed
    - Imports (`import`, `goimport`) are supported
    - Transpiles to `cmd/{script_name}/` with `package main` and `func main()`
- Result type `T ! E` with typed error handling
    - Error types: `MyError ~ error<utf8> { ... }` or typeless `MyError ~ error { ... }`
    - Only `error`, `error<ascii>`, and `error<utf8>` are allowed as error type parameters
    - Declaration: `r : var int64 ! MyError`
    - Functions can return result: `func(...) int64 ! MyError`
    - Check: `if r? { ... }` (no error), `if !r? { ... }` (has error)
    - Error extraction: `r!` gives the error value
    - Transpiles to `cog.Result[T, E]` generic Go struct
- Must-check analysis for option and result types
    - Cannot access option value without `?` check: `if opt? { use opt }`
    - Cannot access result value or error without `?` check
    - `?` = "is OK?" (bool) — works on both option and result
    - `!` = error extraction (value) — result types only, requires prior `?` check
    - Direct check (`if val?`) persists for rest of scope
    - Negated check (`if !val?`) is scoped to its block only
- Generic type aliases with type parameters and constraints
    - Declaration: `List<T ~ any> ~ []T`, `Dict<K ~ comparable, V ~ any> ~ map<K, V>`
    - Instantiation: `names : List<utf8>`, `lookup : Dict<utf8, int64>`
    - Builtin constraints: `any`, `comparable`, `ordered`, `number`, `string`, `int`, `uint`, `float`, `complex`, `signed`, `summable`
    - Union constraints: `T ~ string | int`
    - Interface constraints: `T ~ Stringer`
    - Constraint validation at instantiation
- Generic functions with type parameters on the `func` type
    - Declaration: `genFunc : func<T ~ any>(x : T) = { ... }`
    - With return type: `identity : func<T ~ any>(x : T) T = { return x }`
    - Inferred type arguments: `genFunc("hello")` infers `T = utf8`
    - Explicit type arguments: `genFunc<utf8>("hello")`
    - Constraint validation and type argument mismatch errors
    - Transpiles to Go generics: `func genFunc[T any](x T) { ... }`
- Interfaces
    - Declaration: `Stringer ~ interface { String : func() utf8 }`
    - Used as generic constraints: `func<T ~ Stringer>(x : T) = { x.String() }`
    - Struct satisfaction: a struct satisfies an interface if it declares methods matching every interface method signature
- Methods on struct types
    - Shorthand: `Foo.GetValue : func() utf8 = { ... }` (no receiver variable)
    - Reference shorthand: `&Foo.Mutate : proc() = { ... }` (pointer receiver in Go output)
    - Explicit receiver: `(f : Foo).GetValue : func() utf8 = { return f.value }`
    - Explicit reference receiver: `(f : &Foo).Get : func() utf8 = { return f.value }`
    - Mutable receiver: `(f : var &Foo).Set : proc(v : utf8) = { f.value = v }`
    - Exported methods: `export Foo.String : func() utf8 = { ... }` or `export (f : Foo).String : func() utf8 = { ... }`
    - Methods can be declared in any order relative to the struct definition
    - Method names are scoped to their receiver type (no conflict with global names)
    - `func` methods cannot have a `var` receiver (pure functions cannot mutate state)
    - Duplicate method names on the same type are rejected
    - Selector assignment (`f.value = x`) requires a `var` receiver
- Ownership model for `var` mutable variables:
    - `var` to `var` assignment transfers ownership — source is dead after
    - Immutable to `var` assignment: deep-copies slices/maps/sets (via `cog.Copy()`); primitives and structs of primitives simply copy
    - `func` calls borrow `var` arguments — caller can use the variable again after the call
    - `proc` calls consume `var` arguments — caller cannot use the variable after the call
    - Returning a `var` value transfers ownership to the caller; structs must have all fields alive to be returned
    - `defer` reserves a `var` for the deferred call — reassigning or moving it after the defer is rejected
    - `for` iteration borrows the iterable — it is alive again after the loop
    - Consumption is permanent across conditional branches: if any branch consumes a `var`, it is dead after the entire if/switch/match
    - `dyn` variables deep-copy on read and write for pointer-like types; primitives copy trivially
    - Struct fields support partial moves: `d := c.data` consumes only that field; reassigning `c.data = ...` revives it
    - Pointer-like vs primitive determination is transitive through type aliases and generics
    - Index expressions borrow the container
    - Map/set insertion consumes the key and value map entries; non-comparable types rejected as keys/elements

### Partly implemented

- Canonical syntax highlighting

### Planned

- Result type `T ! E` with typed error handling
    - Also allow `interface{ String() string }` and `interface{ Error() string }` as error types
- Automatically use `https://github.com/go4org/hashtriemap` as map type in concurrent scenarios.
- Type qualifiers
    - `comp` for compile time constants. Similar to Zig' `comptime`. When used on variables, like C++ `constexpr`, when used for functions like C++ `consteval`.
- Variables need to be passed to scope explicitely (no catch all closures)
    - `(foo, bar) { // foo & bar are available in this scope }
- Additional safety regarding mutability and ownership.
- Type switch
    - `switch t { type uint64: ... }`
    - For `t ~ any | interface | union`
- Select statement

- Additional types:
    - `signal<T any>` alias of `chan<T any>struct{}`
- Range operator `0..4 == [0, 1, 2, 3]`
- Builtin operations for 2D / 3D / 4D slices.
- Implement flat AST.
- Fork and rework float16, uint128 and int128 imported packages.
- Builtin `upx` binary packer for smaller binaries.
- LSP
- Adaptive GC (github.com/samborkent/adaptive-gc)
- Automatic struct alignment?

## Syntax

### Operators

* `:` - declare a value identifier with type
* `=` - assign a value to a value identifier
* `:=` - short hand for `: <inferred type> =`
* `~` - declare a type alias

## Example code

```go
package main

goimport (
    "strings"
)

import (
    "geom"
    "geom/metric"
)

a : int64 = 0

export isExported := true
NotExported := true

String ~ utf8
export notExported ~ uint64
export ExportedString ~ String

main : proc() = {
    _ = 10 // inline comment stays inline

    str := @go.strings.ToUpper("str")
    _ = str

    b : float32 = 0.0
    _ = b

    language : var = "cog" // utf8
    lang : var utf8 = "cog"
    _ = lang
    lng : ascii = "cog"
    @print(lng)
    c1 := `hello
    
        world`
    c2 := "hello\n\n\tworld"

    @print(c1)
    @print(c2)

    language = "go"

    if true {
        break
    } else {
        @print(lng)
    }

    if true != false {}
    if true == false && true != false {}
    if true != false {}
    if 5 <= 6 {}
    if !true {}

    fl := -0.6e-7
    _ = fl

    collection : set<utf8> = { "hello1", "hello2" }
    _ = collection

    maths := 5 * 6 / (2 + 3)
    _ = maths

ifLabel:
    if true {
        if true {
            break ifLabel
        }
    }

    newString := definedHere
    _ = newString

    newLang := @if(language == "cog", 25 + 10 - 6, 5)
    _ = newLang

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

    // TODO: temporarily disabled, not fully implemented.
    // tuple : Tuple = {"hello", 10, false}

    utf : var utf8?  = "hello"
    // option : Option? // not allowed
    utf = "option"
    
    // Must-check: cannot access option value without checking first.
    // @print(utf)  // ERROR: must check utf before accessing value
    
    // ? on option = "is set?"
    if utf? {
        // Inside checked block, value access is allowed.
        @print(utf)
    }
    // Direct check persists — value remains accessible.
    @print(utf)
    
    option : var uint64?
    
    if option? {
        @print("do not print")
    }

    option = 10

    // Negated check: !option? = "is NOT set?"
    if !option? {
        @print("not set")
        // @print(option)  // ERROR: proven not set, can't access value
    } else {
        @print(option)
    }

    // Negated check does NOT persist.
    // @print(option)  // ERROR: negated check is scoped

    // Result type: T ! E — typed error handling.
    // Must-check: cannot access value or error without checking first.
    result : var int64 ! DivError
    // @print(result)   // ERROR: must check result before accessing value
    // @print(result!)  // ERROR: must check result before accessing error

    // Static analysis: when assigned a value or error literal, the parser
    // knows statically which variant it is. No check needed.
    knownValue : var int64 ! DivError = 42
    @print(knownValue)  // OK: assigned value literal, proven safe
    knownError : var int64 ! DivError = DivError.DivByZero
    @print(knownError!)  // OK: assigned error literal, error access safe

    // Direct check: if result? = "is OK?"
    // Persists after if-block — value remains accessible.
    if result? {
        @print(result)
        // @print(result!)  // ERROR: proven no error, can't access error
    }
    @print(result)

    // Negated check: if !result? = "has error?"
    // Does NOT persist unless the block exits scope (return/break/continue).
    negatedResult : var int64 ! DivError = safeDivide(10, 0)
    if !negatedResult? {
        @print(negatedResult!)
        // @print(negatedResult)  // ERROR: proven error, can't access value
    }
    // @print(negatedResult)  // ERROR: negated check does not persist

    // Negated check with else: value safe in else branch.
    elseResult : var int64 ! DivError = safeDivide(10, 2)
    if !elseResult? {
        @print(elseResult!)
    } else {
        @print(elseResult)
    }

    // Early-exit promotion: when the error branch exits scope
    // (return/break/continue), the value check is promoted after the if.

    // return in function: processResult checks error, returns early on failure.
    processed : var int64 ! DivError = processResult(safeDivide(10, 2))
    if processed? {
        @print(processed)
    }
    @print(processed)

    // break in loop: error branch exits, so value is proven safe after.
    loopResult : var int64 ! DivError = safeDivide(20, 4)
    for {
        if !loopResult? {
            @print(loopResult!)
            break
        }
        // After error branch exits, value is proven safe.
        @print(loopResult)
        break
    }

    upperCaseString := upper(language)
    @print(upperCaseString)

    @print(upper("foo", "bar"))

    // _ = Planets.Earth.mass

    someFunc("")
    @print(val) // default

    m : Map = {
        "hello": 420,
        "world": 69,
    }

    @print(m)

    otherMap : map<uint64, ascii> = {
        10: "ten",
        20: "twenty",
    }

    localSlice : var []utf8
    localSlice = {"hello", "world"}
    localCopy := localSlice[0]
    _ = localCopy
    mapVal := otherMap[10]
    @print(mapVal)

    typedLiteralA := []int8{
		5, 4, 3, 2, 1,
	}
	_ = typedLiteralA
	typedLiteralB := [5]int8{
		5, 4, 3, 2, 1,
	}
	_ = typedLiteralB
	typedLiteralC := map<ascii, int8>{
		"hello": 5,
	}
	_ = typedLiteralC
	typedLiteralD := set<ascii>{
		"hello",
	}
	_ = typedLiteralD
	typedLiteralE := set<ASC>{
		"hello",
	}
	_ = typedLiteralE

    index : var = 0

outerLoop:
	for {
	innerLoop:
		for {
			index = index + 1

			if index < 5 {
				@print("continue")
				continue innerLoop
			}

			@print("break")
			break outerLoop
		}
	}

	// Loop over int not allowed.
	//for v in 10 {
	//	@print("loop")
	//}

	for _, i in "hello" {
		@print(i)
	}

	cont : []int8 = {5, 4, 3, 2, 1}
	for cont {
		@print("cont")
	}

	for v, k in map<utf8, ascii>{
		"hello": "world",
	} {
		@print(k)
		@print(v)
	}

    Stringy ~ ascii
    stringSet ~ set<utf8>

    _ = @map<uint8, uint64>()
	_ = @map<utf8, ascii, uint32>(1)

	settie := @set<utf8, uint16>(1000)
	_ = settie

	what := @if<uint64, bool>(5 != 6, 10, 6)
	_ = what

	ref := @ref<utf8>()
	_ = ref

	arg : uint64 = 10
	_ = @slice<int32>(arg)
	_ = @slice<int32, uint8>(10)

	arenaProc(arg)

// @cast: bitwise type reinterpretation, returns B? (option type).
    // Narrowing (src > dst) returns None; same-size or widening returns Some.
    bits : int32 = 42
    asFloat := @cast<float32>(bits)               // int32 -> float32 (same size, reinterpret bits)
    asBigFloat := @cast<float64>(bits)            // int32 -> float64 (widen then reinterpret)
    asUnsigned := @cast<uint32>(bits)             // int32 -> uint32 (same-width sign reinterpret)
    flag := true
    flagByte := @cast<uint8>(flag)                // bool -> uint8
    h : float16 = 1.5
    hWide := @cast<uint32>(h)                    // float16 -> uint32 (extract bits, widen)
    narrow : uint8 = 255
    wide := @cast<uint64>(narrow)                 // uint8 -> uint64 (zero-extend)
    small : int8 = 1
    withSource := @cast<int16, int8>(small)       // explicit source type annotation
    withLiteral := @cast<int16, int8>(1)          // literal inferred as int8 from type arg

    // Must check option before accessing value:
    if asFloat? { _ = asFloat }
    if asBigFloat? { _ = asBigFloat }
    if asUnsigned? { _ = asUnsigned }
    if flagByte? { _ = flagByte }
    if hWide? { _ = hWide }
    if wide? { _ = wide }
    if withSource? { _ = withSource }
    if withLiteral? { _ = withLiteral }

    // @as: semantic value conversion (not bitwise reinterpretation).
    // Returns B directly — no Option wrapping.
    myInt : int32 = 42
    myBool := true
    myFloat : float64 = 3.14
    myStr : utf8 = "42"

    // identity: same type = passthrough.
    same := @as<int32>(myInt)
    _ = same

    // numeric widening: safe direct conversion.
    wider := @as<int64>(myInt)
    _ = wider

    // numeric narrowing: returns 0 on overflow.
    narrower := @as<int8>(myInt)
    _ = narrower

    // bool ↔ numeric.
    boolAsInt := @as<int8>(myBool)      // 1
    intAsBool := @as<bool>(myInt)        // true (if non-zero)
    @print(boolAsInt)
    @print(intAsBool)

    // bool ↔ string.
    boolStr := @as<utf8>(myBool)         // "true"
    _ = boolStr

    // string ↔ numeric (parse/format).
    strInt := @as<int32>(myStr)          // 42
    intStr := @as<utf8>(myInt)           // "42"
    @print(strInt)
    @print(intStr)

    // string ↔ float.
    strFloat := @as<float32>(myStr)      // 42.0
    floatStr := @as<utf8>(myFloat)       // "3.14"
    _ = strFloat
    _ = floatStr

    // string ↔ bool.
    strBool := @as<bool>("true")         // true
    boolStr2 := @as<utf8>(myBool)        // "true"
    _ = strBool
    _ = boolStr2

    // ascii ↔ utf8: same-string passthrough.
    val := "hello"
    asAscii := @as<ascii>(val)           // cog.ASCII wrapping
    backToUTF8 := @as<utf8>(asAscii)     // same string
    _ = asAscii
    _ = backToUTF8

    // integer → ascii: FormatInt wrapped in cog.ASCII.
    intAsAscii := @as<ascii>(myInt)      // cog.ASCII("42")
    _ = intAsAscii

    // float → integer: NaN/infinity/fraction becomes 0.
    fltInt := @as<int32>(myFloat)        // 3 (truncated)
    _ = fltInt

    // integer → float: always safe.
    intFlt := @as<float64>(myInt)        // 42.0
    _ = intFlt

    // Cast still works for bitwise reinterpretation.
    // ascii -> utf8 is not supported by @cast (different memory layouts).

    // float16: backed by x448/float16 package, arithmetic promotes to float32.
    half : float16 = 1.5
    halfNeg := -half
    halfSum := half + half
    halfCmp := half < halfNeg
    _ = halfSum
    _ = halfCmp

    // complex32: two float16 parts, arithmetic promotes to complex64.
    comp : complex32 = {1.0, 2.0}
    compNeg := -comp
    compSum := comp + comp
    compEq := comp == compNeg
    _ = compSum
    _ = compEq

    // uint128: backed by lukechampine.com/uint128, ops via methods.
    big : uint128 = 42
    bigSum := big + big
    bigMul := big * big
    bigCmp := big < bigSum
    _ = bigMul
    _ = bigCmp

    // int128: backed by ryanavella/wide, ops via methods.
    big128 : int128 = 42
    big128Neg := -big128
    big128Sum := big128 + big128
    big128Cmp := big128 < big128Neg
    _ = big128Sum
    _ = big128Cmp

    // @as conversions for non-standard types.
    // float32 → integer truncation: uses float64 wrapper for math.Trunc/math.IsNaN.
    f32Val : float32 = 1.5
    f32ToInt := @as<int32>(f32Val)
    @print(f32ToInt)

    // float16 → string conversion: uses Float16.String().
    halfAsStr := @as<utf8>(half)
    @print(halfAsStr)

    // int128 → string conversion: uses Int128.String().
    big128Str := @as<utf8>(big128)
    @print(big128Str)

    // uint128 → string conversion: uses Uint128.String().
    bigStr := @as<utf8>(big)
    @print(bigStr)

    // string → int128/uint128 conversion.
    parsedInt128 := @as<int128>("42")
    parsedUint128 := @as<uint128>("42")
    _ = parsedInt128
    _ = parsedUint128

    // complex32 → complex64 via Complex64().
    compPromoted := @as<complex64>(comp)
    _ = compPromoted

    // complex32 → number: extract real via Complex64().
    compReal := @as<float32>(comp)
    @print(compReal)

    // complex32 → string.
    compStr := @as<utf8>(comp)
    @print(compStr)

    // Cross-family narrowing: signed ↔ unsigned overflow checks.
    signedVal : int64 = 42
    unsignedNarrow := @as<uint8>(signedVal)
    _ = unsignedNarrow
    unsignedWide : uint64 = 42
    signedNarrow := @as<int8>(unsignedWide)
    _ = signedNarrow

    // cross-file: Coordinate type and formatCoord function defined in other.cog.
    loc : Coordinate = {
        lat = 52.37,
        lon = 4.89,
    }
    @print(formatCoord(loc))

    // Match on a generic type parameter constrained by a union.
    describe(42)
    describe("hello match")

    // Match with binding variable — each arm narrows the binding type.
    matchDescribe(3.14)
    matchDescribe("world")

    // Match with default case for unhandled types.
    describeDefault(true)
    describeDefault(99)

    // Match with tilde case — matches the exact type (no constraint propagation).
    matchExact("exact")
    matchExact(100)

    // imported package: geom.Origin and geom.Distance from geom/ subdirectory.
    @print(geom.Origin)
    @print(geom.Distance(geom.Origin, geom.Origin))

    // subpackage: geom/metric (imported as "metric").
    @print(metric.Pi)
    @print(metric.CircleArea(5.0))

    // Generic type alias with `any` constraint: works for every type.
    names : List<utf8> = @slice<utf8>(3)
    @print(names)

    // Generic type alias with `number` constraint: int + uint + float + complex.
    scores : NumSlice<int64> = @slice<int64>(5)
    @print(scores)

    // Generic type alias with `ordered` constraint: int + uint + float + string.
    words : SortableSlice<utf8> = @slice<utf8>(10)
    @print(words)

    // Generic type alias with `comparable` constraint on map keys.
    lookup : Dict<utf8, int64> = @map<utf8, int64>()
    @print(lookup)

    // Two-param generic: pair of any two types.
    // TODO: temporarily disabled, not fully implemented.
    // coord : Pair<float64, float64> = {1.0, 2.0}
    // @print(coord)

    // Multi-constraint: T must satisfy `string` or `int`.
    labels : TagSlice<utf8> = @slice<utf8>(3)
    @print(labels)

    // Generic function: type parameter on the func type itself.
    // Type argument is inferred from the argument type.
    genFunc("hello generics")
    genFunc(42)

    // Explicit type argument: genFunc<utf8>("explicit")
    genFunc<utf8>("explicit type arg")

    // Generic function with return type: T flows through.
    idResult := identity("identity")
    @print(idResult)

    fooVal := "hello"
    barRef : var = &fooVal

    bazVal := "world"
    barRef = &bazVal

    _ = barRef

    FooRef : &utf8 = &fooVal
    BarRef : &ascii = &"hello"    _ = FooRef
    _ = BarRef
    BazRefType ~ &complex128

    methodCaller : var Exported = {
        Method = 42,
    }

    methodCaller = Exported{
        Method = 67,
    }

    methodCaller.Export()

    Print(FooLoo{
        value = "Hello, world!",
    })
}

// Generic function: type parameter on the func type.
genFunc : func<T ~ any>(x : T) = {
    @print(x)
}

// Generic function with return type.
identity : func<T ~ any>(x : T) T = {
    return x
}

arenaProc : proc(n : uint64) = {
	xs := @slice<int64>(n)
	ys := @slice<float64>(n)
	@print(xs)
	@print(ys)
}

ASC ~ ascii

definedHere := "defined globally!"

Status ~ enum<utf8> {
    Open := "open",
    Closed := "closed",
}

Planets ~ enum<planet> {
    Earth := {
        radius = 0.5,
        mass = 0.1,
    },
}

planet ~ struct {
    name : ascii

    export pressure : float64

    export (
        radius : float64
        mass : float64
    )
}

// TODO: temporarily disabled, not fully implemented.
// Tuple ~ (utf8, uint64, bool)

Either ~ utf8 ^ uint64

Option ~ utf8?

// Error type for result examples.
DivError ~ error<utf8> {
    DivByZero := "division by zero",
}

// Function returning a result type.
safeDivide : func(a : int64, b : int64) int64 ! DivError = {
    if b == 0 {
        return DivError.DivByZero
    }

    return a / b
}

// Early return pattern: check error, return on failure, use value after.
processResult : func(r : int64 ! DivError) int64 ! DivError = {
    if !r? {
        return r!
    }
    // After early return, value is proven safe.
    return r
}

upper : func(str : utf8, optional? : utf8, alsoOptional? : utf8 = "wassup") utf8 = {
    return @go.strings.ToUpper(str) + optional + alsoOptional
}

val : dyn utf8 = "default"
other : dyn uint64 // valid, will have zero value as default

someFunc : proc(str : utf8) = {
    @print(val) // default
    val = "overwrite"
    @print(val) // overwrite
}

Map ~ map<utf8, uint64>

array : [3]uint64 = {1, 2, 3}
slice : []utf8 = {"foo", "bar", "baz", "qux"}
SliceType ~ []uint64

// any: accepts every type.
List<T ~ any> ~ []T

// number: int + uint + float + complex types.
NumSlice<T ~ number> ~ []T

// ordered: int + uint + float + string — types that support < > <= >=.
SortableSlice<T ~ ordered> ~ []T

// comparable: ordered + complex + bool + struct + array + enum + pointer + tuple + set.
Dict<K ~ comparable, V ~ any> ~ map<K, V>

// Multiple type params with any constraint.
// TODO: temporarily disabled, not fully implemented.
// Pair<A ~ any, B ~ any> ~ (A, B)

// Multi-constraint union: T must be string or int.
TagSlice<T ~ string | int> ~ []T

// Match on a generic type parameter constrained by a union.
// The type switch narrows T to each concrete case type inside the arm.
describe : func<T ~ int64 | utf8>(x : T) = {
    match x {
    case int64:
        @print("int64: ")
        @print(x)
    case utf8:
        @print("utf8: ")
        @print(x)
    }
}

// Match with binding variable — each arm narrows the binding to the case type.
// The binding shadows the subject with the narrowed type.
matchDescribe : func<T ~ float64 | utf8>(x : T) = {
    match val := x {
    case float64:
        @print("float64: ")
        @print(val)
    case utf8:
        @print("utf8: ")
        @print(val)
    }
}

// Match with default case for unhandled types.
// The default branch receives the full union type.
describeDefault : func<T ~ int64 | utf8 | bool>(x : T) = {
    match x {
    case int64:
        @print("int matched")
    default:
        @print("fallback: ")
        @print(x)
    }
}

// Match with tilde case — the ~ prefix means "exact type match" and does
// not propagate the constraint to the case body (for constraint types that
// should not be narrowed).
matchExact : func<T ~ utf8 | int64>(x : T) = {
    match x {
    case ~utf8:
        @print("exact utf8")
    case ~int64:
        @print("exact int64")
    }
}
```
