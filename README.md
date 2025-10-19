# cog
cog is a Go-based programming language that brings some additional features.

# Operators

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

- Allow package-less files (scripts)
    - These files cannot be imported, and will be excuted as if wrapped in a main function.
    - Should `ctx` be predefined in a script?
- Disallow `main` in declarations besides `main : proc(ctx: context)`.
- Implement `@cast[A, B any](x A) B` to cast types instead of `float32()`, etc. It must panic if casting cannot be done without overflow or precision loss.
    - Also implement `@convert[A, B any](x A) B`, which will perform best-effort conversion, allowing some precision loss and handling overflows.
