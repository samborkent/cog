# Ownership model

Ownership is not relevant for immutable values or references. They can be safely shared, and will always be borrowed, as they are read-only.

Mutable values and references follow the following rules:

- Cannot take `&` reference of `&` variable reference.
- Taking `&` reference of `var` mutable value will consume the variable.
- Taking `*` dereferenced value of `var &` mutable reference will consume the variable.
- Assigning a `var` mutable value to a new immutable variable consumes the variable.
- `&` reference cannot be assigned to `var &`, need to take the `*` dereferenced value first.
- Immutable primitive types can simple be assigned to `var`.
- Immutable pointer-like (structs with point-like fields, slices, maps, sets, function pointers) values will automatically deep copy when assigned to a mutable value.
- `func` cannot have a `var` mutable argument.
- `func` cannot be called `async`, and cannot call `async`.
- `var` mutable variables passed as argument to `func` immutable parameters will be borrowed.
- `proc` can have `var` mutable parameters, argument variable is consumed, ownership is transferred.
- `proc` can be called `async`, and can call `async`.
- `var` mutable variables passed as argument to `proc` immutable parameters will be consumed.
- Sync closure will borrow `var` mutable variables.
- `async` closure consumes `var` mutable variables, ownership is transferred.
- Interfaces and `any` can only be used as generic type constraints, not as value type.
- When a `dyn` variable gets assigned or used in another variable assignment or declaration, its value gets copied.
- When an immutable or `var` variable gets assigned to `dyn` variables, its value gets copied.
