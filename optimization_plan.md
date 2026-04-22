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

Where node stores a pointer to a specific node matching `Kind`. Store AST as `[]Node`, where `Node` is a value.

To be determined if the distinction between statement and expression is still necessary. If still necessary, need to find out how to implement this without interfaces.

## 2. Flat AST (follows 2)

Nodes don't store pointers to other nodes, but instead store a `type NodeIndex uint32`. This index is used to index into the flat `[]Node` AST. Nodes can be added with `append`, and removed if needed by replacing index with `Node{}`.

## 3. Text-based transpiler

Instead of transpiling Cog AST to `go/ast`, nodes will implement `PrintGo(*bytes.Buffer)` or `PrintGo(*string.Builder)`.
This method will print the equivalent Go code to the buffer or builder. We can still keep using the `internal/transpiler/component` package to to make use of pre-defined Go AST components for correct printing if needed.

At the end, the buffer or buildercan write all contents to a file. This can easily be parallelized per file.
