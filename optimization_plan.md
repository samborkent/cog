# Compiler optimization plan

## 1. Get rid of Node interface in AST

Instead of `Node` interface, define:

```go
type NodeKind uint8

const ZeroKind NodeKind = 0

type Node struct {
    Kind NodeKind
    node any
}
```

Where node stores a pointer to a specific node matching Kind. Store AST as `[]Node`, where `Node` is a value.

## 2. Flat AST (follows 1)

Nodes don't store pointers to other nodes, but instead store a `type NodeIndex uint32`. This index is used to index into the flat `[]Node` AST. Nodes can be added with `append`, and removed if needed by replacing index with `Node{}`.

## 3. Text-based transpiler

Instead of transpiling Cog AST to `go/ast`, node will implement `PrintGo(*string.Builder)`.
This method will print the equivalent Go code to the buffer or builder. We can still keep using the `internal/transpiler/component` package to to make use of pre-defined Go AST components for correct printing if needed.

## 4. Parallelized lexer

Each file is lexed in a separate Go routine. Perhaps with a Go routine pool limited by GOMAXPROCS.

## 5. Parallelized transpiler (follows 3)

With text-based transpiler we can easily parallelize printing per file. Perhaps with a Go routine pool limited by GOMAXPROCS.

## 6. Merge two globals passes into ONE
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

## 7. Parallelized parser (follows 6)

After merging passes into single scan, parallelize per-file:

- Globals pass can run in parallel for all files
- AST creation pass can run in parallel for all files
- Use Go routine pool limited by GOMAXPROCS

Note: Must complete globals pass for ALL files before starting AST creation pass, since imports need resolved symbol tables.


