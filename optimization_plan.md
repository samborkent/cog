# Compiler optimization plan

## 1. Flat AST

Nodes don't store pointers to other nodes, but instead store a `type NodeIndex uint32`. This index is used to index into two flat AST slices: `[]Node` & `[]Expr` AST. Nodes can be added with `append`, and removed if needed by replacing index with `nil`.
`Nodes[0]` & `Exprs[0]` must be an empty `nil` node, so unset nodes can use `NodeIndex == 0`.

## 2. Arena based parser allocation

Allocate all AST nodes on an arena, and deallocate once parsing is done. Need an arena per file.

## 3. Store kinds in index

Store `NodeKind` in `NodeIndex` & `ExprIndex`, and store `TypeKind` in `ExprIndex`.

## 4. Text-based transpiler

Instead of transpiling Cog AST to `go/ast`, node will implement `PrintGo(*string.Builder)`.
This method will print the equivalent Go code to the buffer or builder. We can still keep using the `internal/transpiler/component` package to to make use of pre-defined Go AST components for correct printing if needed.

## 5. Parallelized lexer

Each file is lexed in a separate Go routine. Perhaps with a Go routine pool limited by GOMAXPROCS.

## 6. Parallelized transpiler (follows 3)

With text-based transpiler we can easily parallelize printing per file. Perhaps with a Go routine pool limited by GOMAXPROCS.

## 7. Merge two globals passes into ONE
Current state: Parser has 2 globals passes:

1. `preRegisterTypeNames` - scans and pre-registers only type names with empty ValueType
2. Main `FindGlobals` loop - parses full definitions

Why two passes exist: Type names must be in symbol table before parsing expressions that reference them, but full type definitions can appear anywhere in file (including after usage).

Solution: Extend `preRegisterTypeNames` to also pre-register:

- Procedure/function names with placeholder `ValueType` (e.g., `types.None`)
- Global variable names
- Method names on receiver types
- Struct fields

Then the main `FindGlobals` loop fills in full definitions for all symbols, not just types.

Expected gain: ~15-25% reduction in parsing time by eliminating redundant scan

## 8. Parallelized parser (follows 6)

After merging passes into single scan, parallelize per-file:

- Globals pass can run in parallel for all files
- AST creation pass can run in parallel for all files
- Use Go routine pool limited by GOMAXPROCS

Note: Must complete globals pass for ALL files before starting AST creation pass, since imports need resolved symbol tables.
