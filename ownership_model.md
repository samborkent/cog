# Ownership model

Ownership is not relevant for immutable values or references. They can be safely
shared, and will always be borrowed, as they are read-only.

Mutable values and references follow the rules below.

## Rules

### 1. Cannot take `&` reference of `&` variable reference.

An immutable reference cannot be re-referenced — there is no double-indirection.

```go
r : &Point = ...
q := &r  // compile error
```

### 2. Taking `&` reference of `var` mutable value consumes the variable.

```go
p : var Point = Point{...}
r := &p        // p is consumed; r is an immutable & reference to its data
// p is dead here
_ = p // compile error
```

### 3. Taking `*` dereferenced value of `var &` mutable reference consumes the variable.

```go
p : var Point = Point{...}
r : var &Point = &p     // r owns mutable access
// p is dead here
val := *r               // r is consumed; val is a copy of the pointed-to value
// r is dead here
```

### 4. Assigning a `var` mutable value to a new immutable variable consumes the variable.

```go
a : var []int64 = @slice<int64>(3)
b := a         // ownership transfers from a to b; b is immutable
// a is dead here
```

### 5. Immutable primitive types can simply be assigned to `var`.

Primitives (int, float, bool, string, arrays of primitives, structs of only
primitives) are value types. Assigning to `var` is a trivial copy.

```go
a : int64 = 42
b : var int64 = a    // fine — simple copy, a stays alive
```

### 6. Immutable pointer-like values deep-copy when assigned to `var`.

Pointer-like types are: slices, maps, sets, structs with pointer-like fields.
Assigning an immutable value of such a type to a `var` variable performs an
automatic `cog.Copy` deep copy.

```go
a := []int64{1, 2, 3}      // immutable slice
b : var []int64 = a         // deep copy — a stays alive, b gets independent data
b[0] = 99                   // only b's copy changes
@print(a[0])                // 1 — unchanged
```

Without this rule, mutating `b` would corrupt the immutable `a`, breaking
read-only guarantees.

### 7. `func` cannot have `var` mutable parameters.

A `func` is a pure function — it cannot mutate its arguments.

```go
// compile error:
bad : func(x : var int64) = { x = 1 }

// correct:
good : func(x : int64) int64 = { x + 1 }
```

### 8. `func` cannot be called `async` and cannot call `async`.

A `func` is synchronous and pure. It must not launch concurrent work.

```go
// compile error:
f : func() = {
    async otherProc()
}

// also compile error:
async f()
```

### 9. `var` mutable variables passed as argument to `func` immutable parameters are borrowed.

The variable is available again after the call completes. The callee cannot
mutate it.

```go
update : func(p : Point) = { /* read-only */ }

main : proc() = {
    pt : var Point = Point{...}
    pt.x = 1

    update(pt)       // pt is borrowed during the call
    pt.x = 2         // OK — borrow is released
}
```

### 10. `proc` can have `var` mutable parameters. The argument variable is consumed — ownership transfers.

```go
worker : proc(data : var []int64) = {
    data[0] = 99     // worker owns data
}

main : proc() = {
    buf : var []int64 = @slice<int64>(3)
    worker(buf)      // buf is consumed; ownership transferred to worker
    // buf is dead here
}
```

### 11. `proc` can be called `async` and can call `async`.

```go
worker : proc() = {
    async otherWorker() // OK
}

main : proc() = {
    async worker()    // OK
}
```

### 12. A `proc` always takes ownership — even immutable `proc` parameters consume the argument.

A `proc` can outlive its caller (via `async`), so the caller cannot retain
access to the argument after the call returns. The argument is consumed
regardless of the parameter's mutability.

```go
logger : proc(msg : []utf8) = {
    // msg is immutable inside logger, but owned
}

main : proc() = {
    m : var []utf8 = "hello"

    logger(m)        // m is consumed — logger might be async, so
    // m is dead here // the caller cannot assume m survives
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

### 13. A block expression borrows `var` mutable variables.

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

### 14. An `async` closure consumes `var` mutable variables.

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

### 15. Interfaces and `any` can only be used as generic type constraints, not as value types.

```go
// OK — constraint only:
Box ~ struct[T ~ any] { value : T }

// compile error — any used as value type:
x : any = 42
```

### 16. `dyn` variables copy on read and write.

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

### 17. `for` iteration borrows the iterable.

```go
items : var []int64 = []int64{1, 2, 3}

for item in items {     // items is borrowed during iteration
    // items[0] = 99   ← compile error: items is borrowed
    @print(item)
}

items[0] = 99           // OK — iteration done, borrow released
```

Without this rule, mutating the collection during iteration would cause
use-after-free or iteration over invalidated positions (map rehash, slice
reallocation).

### 18. Elements captured into async closures during iteration must be owned.

If an async closure is spawned inside a for-loop body, the iterable is
borrow-live for the synchronous body — but any elements captured by the async
closure must be owned, not borrowed from the iterable.

```go
items : var []utf8 = ["a", "b", "c"]

for item in items {         // items is borrowed
    async {                 // item is borrowed from items
        @print(item)        // unsafe: item refers into items, which may
                            // be freed or mutated after the loop ends
    }
}
// compile error: cannot borrow item into async closure
```

To safely capture individual elements, the element must be copied first:

```go
for item in items {
    local := item           // copy the element
    async {
        @print(local)       // OK — local is independent
    }
}
```

### 19. Assigning `var` to `var` transfers ownership.

```go
a : var []int64 = @slice<int64>(3)
a[0] = 1

b : var []int64 = a     // ownership transfers from a to b
// a is dead here

b[0] = 2                // OK
```

Without this rule, both would alias the same data — mutating `b` would
unexpectedly affect `a`.

### 20. Passing a `var &` mutable reference to a `func` immutable parameter borrows the reference.

The underlying value is borrowed — the caller cannot mutate through its
reference for the duration of the call.

```go
read : func(p : &Point) = { /* read-only */ }

main : proc() = {
    pt : var Point = Point{...}
    r : var &Point = &pt        // r has mutable access
    // pt is dead.

    read(r)                     // r is borrowed
    // r is alive again after the call
    r.x = 10                    // OK
}
```

Without this rule, the callee could hold a reference aliasing the same data
that the caller continues to mutate through `r` — unsound.

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