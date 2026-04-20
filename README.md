# cog

cog is a Go-based hobby programming language that brings some additional features. It is wholly incomplete and a work-in-progress.

The following basic features are missing that need to be implemented before Cog can be used to write useful programs:

- Go-to-Cog type conversions

## TODO:

### Bugs
- When declaring type alias in script mode, the type gets placed in global scope, instead of inside of main.
    This is required for method declaration, so we need to manually disallow using a type which is only defined later in the file in script mode.
- Syntax ambiguity `&` method reference receiver vs bitwise AND

### Features
- Remove `@ref` allocator.
- Change `@cast` signature to `@cast<B, A any>(x A) B?`. Return type will only be set if lossless cast is possible.
- Define builtin functions as `cog` functions.
- Design how iterators should work.
    - Range over int (or other literal) should not be possible.
    - Instead we should range over an iterator function which takes literal as argument.

### Improvements
- Get rid of symbol table in transpiler if possible.
- Defining receiver vars should use regular symbol table, instead of custom fields (e.g. t.inMethod, p.currentReceiver)

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
    - `@cast<B, A ~ any>(x A) B` bitwise type cast (target must be same size or larger)
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
    - `switch var { case val: ... }`
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
    - Declaration: `var r : int64 ! MyError`
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
    - Mutable receiver: `(var f : &Foo).Set : proc(v : utf8) = { f.value = v }`
    - Exported methods: `export Foo.String : func() utf8 = { ... }` or `export (f : Foo).String : func() utf8 = { ... }`
    - Methods can be declared in any order relative to the struct definition
    - Method names are scoped to their receiver type (no conflict with global names)
    - `func` methods cannot have a `var` receiver (pure functions cannot mutate state)
    - Duplicate method names on the same type are rejected
    - Selector assignment (`f.value = x`) requires a `var` receiver

### Partly implemented

- Canonical syntax highlighting

### Planned

- Result type `T ! E` with typed error handling
    - Also allow `interface{ String() string }` and `interface{ Error() string }` as error types
- Type qualifiers
    - `comp` for compile time constants. Similar to Zig' `comptime`. When used on variables, like C++ `constexpr`, when used for functions like C++ `consteval`.
- Variables need to be passed to scope explicitely (no catch all closures)
    - `(foo, bar) { // foo & bar are available in this scope }
- Additional safety regarding mutability and ownership.
- Type switch
    - `switch t { type uint64: ... }`
    - For `t ~ any | interface | union`
- Select statement
- Conversion builtins:
    - `@convert<A, B any>(x A) B` to cast types instead of `float32()`, etc.
        - Will perform best-effort conversion, allowing some precision loss and handling overflows.
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

a : int64 = 0

export isExported := true
NotExported := true

String ~ utf8
export notExported ~ uint64
export ExportedString ~ String

main : proc() = {
    _ = 10 // inline comment stays inline

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

    collection : set<string> = { "hello1", "hello2" }

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
    
    // Must-check: cannot use utf without checking first.
    // @print(utf)  // ERROR
    
    if utf? {
        @print(utf)  // OK: proven set
    }
    @print(utf)  // OK: direct check persists
    
    var option : uint64?
    if option? {
        @print("do not print")
    }
    option = 10
    if !option? {
        @print("not set")
    } else {
        @print(option)  // OK: proven set in else
    }

    // Result type: T ! E
    DivError ~ error<utf8> {
        DivByZero := "division by zero",
    }
    var divResult : int64 ! DivError = safeDivide(10, 2)
    if divResult? {
        @print(divResult)   // OK: proven no error
    }
    @print(divResult)       // OK: direct check persists

    var earlyReturn : int64 ! DivError = safeDivide(10, 0)
    if !earlyReturn? {
        @print(earlyReturn!)  // OK: error extraction
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

    var localSlice : []utf8
    localSlice = {"hello", "world"}
    localCopy := localSlice[0]
    mapVal := otherMap[10]
    @print(mapVal)

    typedLiteralA := []int8{
		5, 4, 3, 2, 1,
	}
	typedLiteralB := [5]int8{
		5, 4, 3, 2, 1,
	}
	typedLiteralC := map<ascii, int8>{
		"hello": 5,
	}
	typedLiteralD := set<ascii>{
		"hello",
	}
	typedLiteralE := set<ASC>{
		"hello",
	}

    var index := 0

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

	what := @if<uint64, bool>(5 != 6, 10, 6)

	ref := @ref<utf8>()
	_ = ref

	arg : uint64 = 10
	_ = @slice<int32>(arg)
	_ = @slice<int32, uint8>(10)

	arenaProc(arg)

    // float16: backed by x448/float16 package, arithmetic promotes to float32.
    half : float16 = 1.5
    halfNeg := -half
    halfSum := half + half
    halfCmp := half < halfNeg

    // complex32: two float16 parts, arithmetic promotes to complex64.
    comp : complex32 = {1.0, 2.0}
    compNeg := -comp
    compSum := comp + comp
    compEq := comp == compNeg

    // uint128: backed by lukechampine.com/uint128, ops via methods.
    big : uint128 = 42
    bigSum := big + big
    bigMul := big * big
    bigCmp := big < bigSum

    // int128: backed by ryanavella/wide, ops via methods.
    signed : int128 = 42
    signedNeg := -signed
    signedSum := signed + signed
    signedCmp := signed < signedNeg

    // cross-file: Coordinate type and formatCoord function defined in other.cog.
    loc : Coordinate = {
        lat = 52.37,
        lon = 4.89,
    }
    @print(formatCoord(loc))

    // Generic function: type argument inferred from argument.
    genFunc("hello generics")
    genFunc(42)

    // Explicit type argument.
    genFunc<utf8>("explicit type arg")

    // Generic function with return type.
    idResult := identity("identity")
    @print(idResult)
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

Tuple ~ (utf8, uint64, bool)

Either ~ utf8 ^ uint64

Option ~ utf8?

// Error type for result examples.
DivError ~ int32
divByZero : DivError = 1

safeDivide : func(a : int64, b : int64) int64 ! DivError = {
    if b == 0 {
        return divByZero
    }
    return a / b
}

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

Map ~ map<utf8, uint64>

array : [3]uint64 = {1, 2, 3}
slice : []utf8 = {"foo", "bar", "baz", "qux"}
SliceType ~ []uint64

// Interfaces and methods.
Stringer ~ interface {
    String : func() utf8
}

Print : func<T ~ Stringer>(x : T) = {
    @print(x.String())
}

export Foo ~ struct {
    value : utf8
}

// Shorthand method (no receiver variable).
export Foo.String : func() utf8 = {
    return "value"
}

// Reference shorthand (pointer receiver in Go output).
&Foo.Mutate : proc() = {}

// Explicit receiver: access fields via receiver variable.
(f : Foo).GetValue : func() utf8 = {
    return f.value
}

// Mutable reference receiver: can assign to fields.
(var f : &Foo).SetValue : proc(v : utf8) = {
    f.value = v
}
```
