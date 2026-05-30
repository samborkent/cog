# Ownership model

Ownership is not relevant for immutable values or references. They can be safely
shared, and will always be borrowed, as they are read-only.

Mutable values and references follow the rules below.

## Core ownership rules

1.  Cannot take `&` reference of `&` variable reference.
2.  Taking `&` reference of `var` mutable value will consume the variable.
3.  Taking `*` dereferenced value of `var &` mutable reference will consume the
    variable.
4.  Assigning a `var` mutable value to a new immutable variable consumes the
    variable.
5.  `&` reference cannot be assigned to `var &`, need to take the `*`
    dereferenced value first.
6.  Immutable primitive types can simply be assigned to `var`.
7.  Immutable pointer-like (structs with pointer-like fields, slices, maps,
    sets) values will automatically deep copy when assigned to a mutable value.
8.  `func` cannot have a `var` mutable argument.
9.  `func` cannot be called `async`, and cannot call `async`.
10. `var` mutable variables passed as argument to `func` immutable parameters
    will be borrowed.
11. `proc` can have `var` mutable parameters, argument variable is consumed,
    ownership is transferred.
12. `proc` can be called `async`, and can call `async`.
13. `var` mutable variables passed as argument to `proc` immutable parameters
    will be consumed.
14. Sync closure will borrow `var` mutable variables.
15. `async` closure consumes `var` mutable variables, ownership is transferred.
16. Interfaces and `any` can only be used as generic type constraints, not as
    value type.
17. When a `dyn` variable gets assigned or used in another variable assignment
    or declaration, its value gets copied via `cog.Copy` (deep copy for
    pointer-like types, shallow for primitives).
18. When an immutable or `var` variable gets assigned to `dyn` variables, its
    value gets copied via `cog.Copy`.
19. Reassigning a `var` variable drops the old value.

## Passing `var &` to a `func` (rule 10 detail)

Passing a `var &` mutable reference to a `func` immutable parameter borrows
the reference; the underlying value becomes borrowed (no mutation through
the caller's reference for the duration of the call).

```go
update : func(point : &Point) = {
    // point is an immutable & reference; read-only
}

main : proc() = {
    p := Point{ ... } // immutable
    r : var &Point = &p   // r owns mutable access to p

    update(r)             // r is *borrowed* — update can read but not mutate
                          // p is still alive, r is not consumed

    // After the call, r can be used for mutation again
    r.x = 10
}
```

Without this rule:
```go
update : func(point : &Point) = { /* read-only */ }

main : proc() = {
    p : var Point = ...
    r : var &Point = &p      // r has write access to p

    update(r)                // If r were not tracked as borrowed,
                             // update could mutate p through a copy.
                             // Meanwhile r still has a live reference
                             // to the same data — aliased mutation.
    r.x = 10                 // r.x and update's `point` would alias
}
```

## Assigning `var` to `var` transfers ownership

```go
main : proc() = {
    a : var []int64 = @slice<int64>(3)
    a[0] = 1

    b : var []int64 = a     // ownership of the underlying slice data
                            // transfers from a to b
    // a is now dead — cannot use a here

    b[0] = 2                // OK
}
```

Without this rule, both `a` and `b` would alias the same slice data.
Mutating through `b` would affect `a`, violating single-writer guarantees.

## Assigning immutable to new `var` declaration copies

```go
main : proc() = {
    a := []int64{1, 2, 3}       // immutable slice
    b : var []int64 = a         // deep copy — a stays alive, b gets its own data

    b[0] = 99                   // only b's copy is mutated

    @print(a[0])                // 1 — a is unchanged
}
```

Without this rule, `b` would alias `a`'s data, meaning mutating `b[0]`
would corrupt the immutable `a` — a soundness hole.

## Reassigning a `var` variable drops the old value

```go
main : proc() = {
    x : var []int64 = @slice<int64>(3)
    x[0] = 1

    x = @slice<int64>(5)        // old slice data is dropped/freed
    x[0] = 2                    // new data is independent
}
```

Without this rule, the old slice memory would leak.

## Option value access (`x!` after `if x?`) consumes the option

```go
main : proc() = {
    x : int64? = 42

    if x? {
        val := x!       // consumes the inner value from x
                        // x is now in "none" state (Set = false)

        @print(val)     // 42
    }

    // x still exists, but its Value is no longer valid
}
```

The transpiler already emits `cog.Option[T]{Value: expr, Set: true}` for
assignments, and `x!` maps to `x.Value`. After consuming the inner value,
the symbol table should mark `x` as having `Set = false` to prevent
double-access.

## Result error access (`x!`) after check

```go
readFile : proc(path utf8) utf8 ! IOError = { ... }

main : proc() = {
    contents : utf8 ! IOError = readFile("foo.txt")

    if contents? { // is contents error?
        @print(contents!)       // error access
    } else {   
        @print(contents)  // value access
    }
}
```

The transpiler's suffix `!` maps to `.Error` for results. After consuming
either branch, the result should be marked consumed.

## `for` iteration borrows the iterable

```go
main : proc() = {
    items : var []int64 = []int64{1, 2, 3}

    for item in items {         // items is borrowed during iteration
        @print(item)
    }

    items[0] = 99               // OK — iteration is done, ownership returned
}
```

```go
main : proc() = {
    items : var []int64 = []int64{1, 2, 3}

    for item in items {
        // items is borrowed — cannot mutate during iteration
        // items[0] = 99  ← compile error: items is borrowed
        @print(item)
    }
}
```

Without this rule, mutating the collection during iteration could cause
use-after-free or iteration over invalidated positions (e.g. map insertion
causing rehash, slice append causing reallocation).

## `async` proc with `var` params — ownership is transferred once

```go
worker : proc(data : var []int64) = {
    // data is owned by this proc
    data[0] = 99
}

main : proc() = {
    buf : var []int64 = @slice<int64>(5)

    async worker(buf)   // buf is consumed — ownership transferred to worker
    // buf is dead here
}
```

When called with `async`, the argument is consumed at the call site.
The async closure does NOT re-capture it (it was already transferred).

## `dyn` variables copy via `cog.Copy`

```go
MyStruct ~ struct {
    data : []utf8
}

main : proc() = {
    dyn s : MyStruct = MyStruct{data: []utf8{"hello"}}

    // Reading dyn copies the value
    local := s
    // `local` is a deep copy of `s` — cog.Copy is invoked under the hood for complex types

    // Assigning to dyn also copies
    s = local
}
```

Since `dyn` uses `context.WithValue` under the hood, values flow through
interface{}, which copies the interface header but not the pointed-to data.
`cog.Copy` ensures pointer-like fields are deep-copied so the original
scope cannot mutate data visible through a `dyn` read elsewhere.

## Tuple ownership (future work)

Tuples are not yet fully worked out. When implemented, tuple element access
should consume individual elements (like destructuring), and assigning a
tuple to a variable should deep-copy the elements.

## Key types and their ownership behavior

| Type | Assignment to `var` | Pass to `func` param | Pass to `proc` param | `dyn` copy |
|------|--------------------|----------------------|----------------------|-----------|
| Primitive (int, float, bool, string) | Shallow copy | Borrow | Consume | Shallow copy |
| Array of primitives | Shallow copy | Borrow | Consume | Shallow copy |
| Array of pointer-like | Deep copy | Borrow | Consume | Deep copy |
| Struct (value fields only) | Shallow copy | Borrow | Consume | Shallow copy |
| Struct (with pointer fields) | Deep copy | Borrow | Consume | Deep copy |
| Slice, Map, Set | Deep copy | Borrow | Consume | Deep copy |
| Option, Result, Either | Deep copy | Borrow | Consume | Deep copy |
| `&` reference | Copied (shared) | Borrow (read-only) | N/A | N/A |
| `var &` reference | Consumed on deref | Borrowed | Consumed | N/A |
