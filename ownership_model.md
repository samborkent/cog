# Ownership model

Ownership is not relevant for immutable values or references. They can be safely
shared, and will always be borrowed, as they are read-only.

Mutable values and references follow the rules below.

## Core principles

The model is built on two fundamental distinctions:

1. **`func` (pure function) vs `proc` (procedure)**: A `func` is synchronous and
   read-only — it borrows arguments. A `proc` may be async — it always consumes
   arguments.
2. **Borrow vs consume**: A borrow is temporary (call duration, block duration).
   A consume is permanent (ownership transfers, source is dead).

The type system determines whether assignment to `var` is a shallow copy
(primitives, structs of primitives, fixed arrays of primitives) or a deep copy
(pointer-like types: slices, maps, sets, structs with pointer-like fields).
See the [reference table](#type-behavior-reference) below.

## Rules

### 1. Immutable values are always borrowed.

Immutable values and references are read-only. They can be freely shared,
copied, and passed — no ownership tracking needed.

```go
r : &Point = ...
q := &r   // compile error — no double-indirection on immutable & reference
```

### 2. Taking `&` reference of `var` mutable value consumes the variable.

```go
p : var Point = Point{...}
r := &p        // p is consumed; r is an immutable & reference to its data
// p is dead here
```

### 3. Dereferencing `var &` mutable reference consumes the variable.

```go
p : var Point = Point{...}
r : var &Point = &p     // r owns mutable access
// p is dead here
val := *r               // r is consumed; val is a copy of the pointed-to value
// r is dead here
```

### 4. Assigning `var` to `var` transfers ownership.

```go
a : var []int64 = @slice<int64>(3)
b : var []int64 = a     // ownership transfers from a to b
// a is dead here
b[0] = 2                // OK — b owns the data
```

Without this rule, both would alias the same data — mutating `b` would
unexpectedly affect `a`.

### 5. Assigning immutable to `var` copies: shallow for primitives, deep for pointer-like types.

Primitives (int, float, bool, string, arrays of primitives, structs of only
primitives) are value types. Assigning to `var` is a trivial copy:

```go
a : int64 = 42
b : var int64 = a    // fine — simple copy, a stays alive
```

Pointer-like types (slices, maps, sets, structs with pointer-like fields)
require a deep copy. Assigning an immutable value of such a type to a `var`
variable performs an automatic `cog.Copy`:

```go
a := []int64{1, 2, 3}      // immutable slice
b : var []int64 = a         // deep copy — a stays alive, b gets independent data
b[0] = 99                   // only b's copy changes
@print(a[0])                // 1 — unchanged
```

The compiler resolves pointer-like vs. primitive transitively through type
aliases and generic instantiations. See the [reference table](#type-behavior-reference)
and [rule 21](#21-the-compiler-resolves-pointer-like-vs-primitive-transitively-including-through-type-aliases-and-generics) for details.

### 6. `func` is pure: no `var` params, no `async`, no mutation.

```go
// compile errors:
bad  : func(x : var int64) = { x = 1 }
bad2 : func() = { async otherProc() }
bad3 : async someFunc()

// correct:
good : func(x : int64) int64 = { x + 1 }
```

### 7. `func` borrows arguments.

A `var` argument passed to a `func` is borrowed for the duration of the call.
The callee cannot mutate it, and the caller can use the variable again after
the call returns.

```go
update : func(p : Point) = { /* read-only */ }

main : proc() = {
    pt : var Point = Point{...}
    pt.x = 1
    update(pt)       // pt is borrowed during the call
    pt.x = 2         // OK — borrow is released
}
```

This includes `var &` mutable references passed to `func` — the reference is
borrowed, and the caller can mutate through it again after the call:

```go
read : func(p : &Point) = { /* read-only */ }

main : proc() = {
    pt : var Point = Point{...}
    r : var &Point = &pt        // r has mutable access; pt is dead.

    read(r)                     // r is borrowed
    // r is alive again after the call
    r.x = 10                    // OK
}
```

Without this rule, the callee could hold a reference aliasing the same data
that the caller continues to mutate through `r` — unsound.

### 8. `proc` always consumes arguments.

A `proc` can outlive its caller (via `async`), so the caller cannot retain
access to the argument after the call returns. The argument is consumed
regardless of the parameter's mutability.

```go
logger : proc(msg : []utf8) = {
    // msg is immutable inside logger, but owned
}

main : proc() = {
    m : var []utf8 = "hello"
    logger(m)        // m is consumed — logger might be async
    // m is dead here
}
```

This applies transitively: passing a proc parameter to another proc also
consumes it:

```go
middleman : proc(data : var []int64) = {
    worker(data)            // data consumed by worker
    // data is dead here
}
```

To see why consumption is necessary even for immutable parameters:

```go
logger : proc(msg : []utf8) = {
    async transmit(msg)     // msg used after caller returns
}

main : proc() = {
    buf : var []utf8 = @utf8("hello")
    logger(buf)
    // buf was not consumed → caller could reuse buf's memory
    // while transmit(buf) in logger is still reading it
}
```

### 9. `proc` can be async; `func` cannot.

```go
worker : proc() = {
    async otherWorker() // OK
}

main : proc() = {
    async worker()    // OK
    // someFunc()    // compile error: cannot call func as async
}
```

### 10. Returning a value from a `proc` transfers ownership to the caller.

A `proc` that returns a `var` type transfers ownership of the returned value to
the caller. The caller receives an owned value.

```go
maker : proc() var []int64 = {
    result : var []int64 = @slice<int64>(3)
    result                          // ownership of result transfers to caller
}

main : proc() = {
    data : var []int64 = maker()    // data owns the returned slice
    data[0] = 99                    // OK
}
```

A `proc` cannot return a `&` reference to a local variable — that would create
a dangling pointer. `&` return types are only valid when the reference comes
from a parameter.

```go
// compile error — returning reference to local:
bad : func() &Point = {
    p : var Point = Point{1, 2}
    @ref(p)
}
```

### 11. A block expression borrows `var` mutable variables.

A block expression (`{ stmts; expr }`) evaluates its statements and final
expression, then yields the result. Block expressions are not first-class
values — they cannot be assigned or returned.

```go
main : proc() = {
    x : var int64 = 1
    y := {              // x is borrowed during the block
        @print(x)
        42
    }                   // x is alive again after the block
    x = y               // OK
}
```

### 12. An `async` closure consumes `var` mutable variables.

An async closure runs in a separate goroutine and may outlive the caller.
`var` variables referenced inside are moved into the closure — ownership
transfers permanently. Async closures are not first-class values and cannot
be assigned or returned.

```go
main : proc() = {
    x : var int64 = 1
    async { @print(x) } // x is consumed, moved into async closure
    // x is dead here
}
```

### 13. `defer` reserves a `var` variable — the variable must survive intact to scope exit.

A `defer` statement registers a call to be executed in reverse order when the
enclosing scope exits. The deferred call will consume the variable when it
executes. Since the deferred call runs *last* (at scope exit), the variable
must survive all intervening code without being consumed or moved.

The compiler checks this at the `defer` declaration point: the variable must
be alive and remain alive through the rest of the scope. Any subsequent
consumption, re-assignment, or move of the variable (or its reserved field)
is a compile error — it would leave the variable dead before the defer runs.

The compiler checks forward from the `defer` declaration: if any subsequent
statement consumes or moves the variable (or its reserved field), the `defer`
is a compile error — the deferred call would find the variable dead at scope
exit.

```go
main : proc() = {
    f : var File = open("file.txt")
    defer close(f)        // OK — f is never consumed after this point
    f.Read()              // OK — borrowing is fine
}
```

```go
main : proc() = {
    c : var Container = Container{data: @slice<int64>(3), label: "hello"}
    defer cleanup(c.data)    // compile error — c.data is consumed below
    c.data = @slice<int64>(5)
}
```

If the variable was already consumed before the `defer`, the `defer` itself is
a compile error:

```go
main : proc() = {
    c : var Container = Container{data: @slice<int64>(3), label: "hello"}
    d := c.data                // c.data consumed
    defer cleanup(c.data)      // compile error — c.data is already dead
}
```

Multiple defers capturing the same variable are rejected. The first-to-execute
defer (last in source order) reserves the variable; the other defer (earlier in
source order) sees it already reserved:

```go
main : proc() = {
    c : var Container = Container{data: @slice<int64>(3), label: "hello"}
    defer cleanup(c.data)   // compile error — c.data is reserved by the defer below
    defer cleanup(c.data)   // reserves c.data (first-to-execute at scope exit)
}
```

Field-level reservation follows partial-move semantics (rule 19): reserving
`c.data` for a defer does not reserve `c.label`.

```go
main : proc() = {
    c : var Container = Container{data: @slice<int64>(3), label: "hello"}
    defer cleanup(c.data)   // OK — c.label can still be used, no consumption of c.data after this
    c.label = "bye"         // OK — c.label is not reserved
}
```

### 14. An `async` closure inside a borrow-scope overrides the borrow with consumption.

If a borrow-scope (block expression, for-loop body) contains an `async` closure
that captures a `var` outer variable, the variable is consumed (not borrowed).
The borrow is never released because the async closure outlives the borrow-scope.

```go
main : proc() = {
    x : var int64 = 1
    y := {
        async { @print(x) }   // x consumed by async closure
        42
    }
    // x is dead here (consumed, not borrowed)
}
```

This overrides rules 11 (block borrows) and 15 (for borrows) when an async
capture is present. The compiler must scan all nested statements in the
borrow-scope for async closures before deciding borrow vs. consume.

### 15. `for` iteration borrows the iterable.

```go
items : var []int64 = []int64{1, 2, 3}

for item in items {     // items is borrowed during iteration
    // items[0] = 99   ← compile error: items is borrowed
    @print(item)
}

items[0] = 99           // OK — iteration done, borrow released
```

The borrow spans the entire loop construct, including nested loops over the
same variable (read-only borrows are shareable):

```go
items : var []int64 = []int64{1, 2, 3}
for outer in items {
    for inner in items {   // OK — both are read-only borrows
        @print(inner)
    }
}
items[0] = 99              // OK — borrow released
```

The borrow count must be a counter (not a boolean flag) to support nesting.

Elements captured into async closures during iteration must be owned — copied
before capture:

```go
for item in items {
    local := item           // copy the element
    async {
        @print(local)       // OK — local is independent
    }
}
```

### 16. Consumption is permanent across conditional branches.

If a `var` variable is consumed in any branch of `if`/`else` or `match`, it is
considered dead after the entire conditional — even if other branches only
borrow it.

```go
main : proc() = {
    data : var []int64 = @slice<int64>(3)
    if cond {
        worker(data)        // data consumed
    } else {
        @print(data[0])     // data borrowed
    }
    // data is dead here    // conservative: consumed in some branch
}
```

Consumption is also permanent across loop iterations. A `var` consumed inside
a loop body is dead for subsequent iterations:

```go
main : proc() = {
    data : var []int64 = @slice<int64>(3)
    for i in 0..10 {
        worker(data)        // consumed on first iteration
        // compile error: data is dead on second iteration
    }
}
```

To re-initialize each iteration, declare a fresh variable inside the loop:

```go
for i in 0..10 {
    data : var []int64 = @slice<int64>(3)  // fresh each iteration
    worker(data)
}
```

### 17. Interfaces and `any` can only be used as generic type constraints, not as value types.

```go
// OK — constraint only:
Box ~ struct[T ~ any] { value : T }

// compile error — any used as value type:
x : any = 42
```

### 18. `dyn` variables copy on read and write.

Since `dyn` is implemented with Go's `context.WithValue`, values flow through
`any`, which copies the interface header but not the pointed-to data.

For basic types (primitives, structs of primitives), the copy is trivial:

```go
dyn x : int64 = 42
y := x              // simple value copy
y = 99              // x is unaffected
```

For pointer-like types (slices, maps, structs with pointer fields), `cog.Copy`
is used to deep-copy:

```go
dyn s : []utf8 = ["hello"]
local := s              // deep copy via cog.Copy
local[0] = "bye"        // s is unaffected

s = local               // also deep-copied on write
```

### 19. Struct fields inherit the `var`/`dyn`/immutable property of the enclosing variable and support partial moves.

Struct fields do not carry independent `var`/`dyn` annotations — those
qualifiers are inherited from the enclosing variable. Ownership is tracked
per-field. Moving a field out of a `var` struct consumes only that field; the
struct variable and its other fields remain alive.

```go
Container ~ struct { data : []int64; label : utf8 }

main : proc() = {
    c : var Container = Container{data: @slice<int64>(3), label: "hello"}
    c.data[0] = 1            // OK — c is var, so c.data is also var

    d := c.data              // c.data is consumed (var → immutable)
    // c.data is dead here   // but c itself is still alive
    // c.label is still alive

    @print(c.label)          // OK — other fields accessible
}
```

Partial moves interact with borrowing:

- **Whole-struct borrow** (passing to `func`): no field can be moved out,
  and field-level mutation is blocked for the call duration.
- **Per-field borrow** (taking `&c.data`): only that specific field is
  borrowed. Other fields can be read and mutated freely.
- **Move-out after whole-struct borrow**: `"cannot move c.data: c is borrowed"`
- **Whole-struct use after field moved out**: `"cannot move c: field 'data' is already moved out"`

When a moved-out field is re-assigned (`c.data = @slice<int64>(5)`), the field
becomes alive again. The struct as a whole can then be borrowed, passed, or
moved normally.

```go
main : proc() = {
    c : var Container = Container{data: @slice<int64>(3), label: "hello"}
    d := c.data                     // c.data consumed
    // worker(c)                    // compile error — c.data is dead
    c.data = @slice<int64>(5)       // re-assign — c.data alive again
    worker(c)                       // OK — all fields alive
}
```

When all fields are consumed, the struct variable itself becomes dead:

```go
main : proc() = {
    c : var Container = Container{data: @slice<int64>(3), label: "hello"}
    d := c.data              // c.data consumed
    s := c.label             // c.label consumed (shallow copy — primitive)
    // c is dead here        // all fields consumed
}
```

Passing a `var` struct to a `proc` consumes the entire struct — all fields are
transferred as a unit. A `func` call borrows the entire struct — no field can
be mutated during the call, and all fields are alive afterward.

Error messages for common violations:

| Violation | Error message |
|-----------|---------------|
| Use of a moved-out field | `c.data: field 'data' moved out here` |
| Pass whole struct after partial move | `cannot move c: field 'data' is already moved out` |
| Borrow whole struct after partial move | `cannot borrow c: field 'data' is moved out` |
| Move field while struct is borrowed | `cannot move c.data: c is borrowed` |

### 20. Enum variant payloads follow struct-like ownership.

When destructuring an enum variant via `match`, the payload value inherits the
ownership qualifier of the matched value. A `var` enum transfers payload
ownership to the match arm.

```go
Result ~ enum<int8> {
    Ok := 0 with value : utf8;
    Err := 1 with msg : utf8
}

main : proc() = {
    r : var Result = Result.Ok("hello")
    match r {
        case Ok(val) => {
            // r is consumed; val owns the utf8 payload
        }
        case Err(msg) => {
            // r is consumed; msg owns the utf8 payload
        }
    }
}
```

Since only one match arm executes, the value is consumed in the matching arm
and dropped in the others.

If the match binds are immutable (the match arm does not declare `var`), the
enum value is borrowed (not consumed). The enum variable is alive after the
match. This follows the same `func`/`proc` distinction (rules 7, 8): an
immutable binding borrows; a `var` binding consumes.

```go
match r {
    case Ok(val) => {    // val is immutable — r is borrowed
        @print(val)
    }
    case Err(msg) => {   // msg is immutable — r is borrowed
        @print(msg)
    }
}
// r is alive here
```

### 21. The compiler resolves pointer-like vs. primitive transitively, including through type aliases and generics.

"Pointer-like" is determined by examining the underlying Go representation:

| Cog Type | Go Representation | Pointer-like? |
|----------|------------------|---------------|
| `int64`, `float32`, `bool`, `ascii`, `utf8` | Primitive value | No |
| `int64?` (Option) | Struct with value field | No, if inner type is primitive; Yes, if inner type is pointer-like (determined at instantiation) |
| `[]int64` | Slice header (ptr, len, cap) | Yes |
| `map[utf8]int64` | Map header | Yes |
| `Set[int64]` | Map header | Yes |
| `[3]int64` | Fixed array | No (primitives); Yes (pointer-like elements) |
| `struct { x, y int64 }` | Struct of values | No |
| `struct { data []int64 }` | Struct with slice field | Yes |
| `MySlice ~ []int64` | Slice (after alias resolution) | Yes |
| `Box<[]int64>` | Struct with pointer-like generic arg | Yes |
| `&Point` | Pointer | Yes (but copied by pointer-copy, not deep copy — it is an immutable reference) |

The compiler must resolve through type aliases and generic instantiations. For
generic types, the determination is made at instantiation time.

At transpile time, if the compiler can statically determine that a type is
primitive (all fields/elements are basic Go types), it generates a simple
assignment instead of `cog.Copy`.

### 22. Index expressions borrow the container; `&items[i]` extends the borrow for the reference's lifetime.

Reading `items[i]` copies the element out (for value types) and borrows `items`
for the duration of the read. Taking a reference `&items[i]` borrows `items`
for the lifetime of the resulting reference.

```go
main : proc() = {
    items : var []Point = [Point{1,2}, Point{3,4}]
    p := items[0]               // items borrowed, p is a copy
    p.x = 99                    // OK — p is independent
    items[1].x = 7              // OK — items no longer borrowed

    r := &items[0]              // items borrowed for r's lifetime
    r.x = 99                    // OK — mutation through reference
    // items is alive again when r is no longer used
}
```

This is a natural extension of partial borrows (rule 19) — only the indexed
element is borrowed, not the entire container.

### 23. Method receivers follow the same ownership rules as regular parameters.

Methods are syntactic sugar for functions where the receiver is the first
parameter. A `proc` method with a `var` receiver consumes the receiver value:

```go
Foo ~ struct { val : []int64 }

Foo.Process : proc(self : var &Foo) = { ... }

main : proc() = {
    f : var Foo = Foo{...}
    f.Process()             // f consumed (self is var &Foo)
    // f is dead here
}
```

A `func` method borrows the receiver for the duration of the call.

### 24. Map/set insertion consumes the key/value.

When inserting into a `var` map or set, the key and value are consumed (the
collection owns its contents).

```go
main : proc() = {
    m : var map[utf8]var []int64 = @map<utf8, []int64>()
    key : utf8 = "items"
    val : var []int64 = @slice<int64>(3)

    m[key] = val        // both key and val consumed — map owns them
    // key is dead here
    // val is dead here
}
```

### 25. Chained method calls borrow the receiver through the chain.

When methods are chained (e.g., `x.foo().bar().baz()`), the receiver `x` is
borrowed for the duration of the entire chain if any intermediate method returns
a borrowed reference.

```go
items : var []int64 = []int64{3, 1, 2}
result := items.sort().reverse()   // items borrowed for entire chain
items[0] = 99                       // OK after chain completes
```

## Type behavior reference

| Type | Assign to `var` | Pass to `func` | Pass to `proc` | `dyn` copy |
|------|----------------|----------------|----------------|-----------|
| Primitive (int, float, bool, string) | Shallow copy | Borrow | Consume | Shallow |
| Array of primitives | Shallow copy | Borrow | Consume | Shallow |
| Array of pointer-like elements | Deep copy | Borrow | Consume | Deep |
| Struct (value fields only) | Shallow copy | Borrow | Consume | Shallow |
| Struct (pointer-like fields) | Deep copy | Borrow | Consume | Deep |
| Slice, Map, Set | Deep copy | Borrow | Consume | Deep |
| `&` reference | Copied (shared) | Borrowed (read-only) | N/A | N/A |
| `var &` reference | Consumed on deref | Borrowed | Consumed | N/A |
