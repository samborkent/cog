# Defer Statement Inlining Implementation Plan

## Objective

Add `defer` as a first-class Cog language keyword that transpiles to Go *without* using Go's runtime `defer` statement. Instead, all deferred calls are collected and inlined in reverse order before every return point in the enclosing function.

---

## 1. Token & Keyword Registration

**File:** `internal/tokens/type.go`
- Add `Defer tokens.Type = iota` in the "Control-flow keywords" section (after `Continue`, before `In`)
- Add the string case: `case Defer: return "defer"`

**File:** `internal/tokens/lookup.go`
- Add the keyword entry: `Defer.String(): Defer,` to the `Keywords` map

---

## 2. AST Node

**File:** `internal/ast/defer.go` (new file)

```go
package ast

type Defer struct {
    Token tokens.Token
    Expr  ExprIndex // *Call or *ProcedureLiteral (anonymous closure)
}

func (a *AST) NewDefer(tok tokens.Token, expr ExprIndex) NodeIndex
func (n *Defer) Pos() (uint32, uint16)
func (n *Defer) Hash() uint64
func (n *Defer) StringTo(out *strings.Builder, a *AST)
```

The `Expr` field must be one of:
- A `*ast.Call` — a direct procedure/function call (e.g. `defer fclose(file)`)
- A `*ast.ProcedureLiteral` — an anonymous closure (e.g. `defer { ... }`)

No other expression types are valid; the parser rejects mismatches.

---

## 3. Parser

**File:** `internal/parser/statement.go`

Add a new case in `parseStatement` for `tokens.Defer`:

```go
case tokens.Defer:
    deferToken := p.lex.This()
    p.lex.Step() // consume 'defer'

    // Parse the deferred expression
    exprIdx := p.expression(ctx, types.None)
    if exprIdx == ast.ZeroExprIndex {
        return ast.ZeroNodeIndex
    }

    expr := p.ast.Expr(exprIdx)
    // Validate: must be a Call or ProcedureLiteral
    switch expr.(type) {
    case *ast.Call:
        // OK — direct call
    case *ast.ProcedureLiteral:
        // OK — closure
    default:
        p.error(deferToken, "defer requires a procedure call or closure", "parseStatement")
        return ast.ZeroNodeIndex
    }

    return p.ast.NewDefer(deferToken, exprIdx)
```

**Validated constraint:** `defer` only accepts:
1. `defer someProc(args...)` — a direct call expression
2. `defer { stmts }` — an inline closure/block

Invalid: `defer x + 1`, `defer someVar`, `defer ident.field`.

---

## 4. Transpiler — Core Inlining Algorithm

### 4a. New types on `Transpiler`

**File:** `internal/transpiler/transpiler.go`

```go
type deferInfo struct {
    callStmts []goast.Stmt // the Go statements representing the deferred call
}
```

Add field to `Transpiler` struct:
```go
deferStack []deferInfo
```

Reset in `resetFileState`:
```go
t.deferStack = nil
```

### 4b. convertStmt handler for *ast.Defer

**File:** `internal/transpiler/statement.go`

In the `convertStmt` switch, add:

```go
case *ast.Defer:
    expr := t.Expr(n.Expr)
    
    // Convert the expression to Go statements.
    // - For *ast.Call: produce an ExprStmt (the call)
    // - For *ast.ProcedureLiteral: produce all body statements
    stmts, err := t.convertDeferExpr(expr)
    if err != nil {
        return nil, err
    }
    
    // Instead of emitting, push onto the defer stack (FILO order).
    t.deferStack = append(t.deferStack, deferInfo{callStmts: stmts})
    
    // Emit a no-op placeholder statement for line directive tracking
    return []goast.Stmt{&goast.EmptyStmt{}}, nil
```

### 4c. convertDeferExpr helper

```go
func (t *Transpiler) convertDeferExpr(expr ast.Expr) ([]goast.Stmt, error) {
    switch n := expr.(type) {
    case *ast.Call:
        // Convert the call expression
        goExpr, err := t.convertExpr(n)
        if err != nil {
            return nil, err
        }
        return []goast.Stmt{&goast.ExprStmt{X: goExpr}}, nil

    case *ast.ProcedureLiteral:
        body := t.Node(n.Body).(*ast.Block)
        stmts := make([]goast.Stmt, 0, len(body.Statements))
        
        prevInFunc := t.inFunc
        prevUsesDyn := t.usesDyn
        t.usesDyn = false
        if procType, ok := n.ProcedureType.(*types.Procedure); ok {
            t.inFunc = procType.Function
        } else {
            t.inFunc = false
        }

        for _, s := range body.Statements {
            stmt, err := t.convertStmt(t.Node(s))
            if err != nil {
                return nil, err
            }
            stmts = append(stmts, stmt...)
        }

        bodyUsesDyn := t.usesDyn
        t.usesDyn = prevUsesDyn || bodyUsesDyn
        t.inFunc = prevInFunc

        return stmts, nil

    default:
        return nil, fmt.Errorf("defer requires a procedure call or closure, got %T", n)
    }
}
```

### 4d. Inject deferred statements before returns — the central algorithm

**Approach:** After converting a function/procedure body's statements (both in `convertDecl` for proc declarations and `convertExpr` for `*ast.ProcedureLiteral`), walk the generated Go AST to find all `*goast.ReturnStmt` nodes and insert the reversed deferred statements before them. Also append deferred statements before any implicit return (end of function body).

#### Walk-and-inject function

```go
// injectDeferred prepends the current deferred statements (in reverse order)
// before every return statement in the given block, and appends them at the
// end of the block for implicit returns.
func (t *Transpiler) injectDeferred(body *goast.BlockStmt) {
    if len(t.deferStack) == 0 {
        return
    }

    // Build the deferred preamble from the stack (reverse order = Go defer semantics)
    deferredStmts := t.buildReversedDeferredStmts()

    // 1. Inject before every explicit return statement
    injectBeforeReturns(body, deferredStmts)

    // 2. Append at end for implicit returns (fall-through end of function)
    body.List = append(body.List, deferredStmts...)
}

// buildReversedDeferredStmts flattens the defer stack in reverse order
// so the first deferred call executes last (matching Go's LIFO semantics).
func (t *Transpiler) buildReversedDeferredStmts() []goast.Stmt {
    total := 0
    for _, d := range t.deferStack {
        total += len(d.callStmts)
    }
    stmts := make([]goast.Stmt, 0, total)
    for i := len(t.deferStack) - 1; i >= 0; i-- {
        stmts = append(stmts, t.deferStack[i].callStmts...)
    }
    return stmts
}
```

#### Recursive injection helper

```go
// injectBeforeReturns recursively walks a block's statements and prepends
// deferredStmts before each *goast.ReturnStmt found at any nesting level.
func injectBeforeReturns(block *goast.BlockStmt, deferredStmts []goast.Stmt) {
    for i, stmt := range block.List {
        switch s := stmt.(type) {
        case *goast.ReturnStmt:
            // Insert deferred statements before this return
            preamble := make([]goast.Stmt, len(deferredStmts))
            copy(preamble, deferredStmts)
            block.List = append(block.List[:i], append(preamble, block.List[i:]...)...)
            i += len(preamble) // skip past the injected statements

        case *goast.IfStmt:
            injectBeforeReturns(s.Body, deferredStmts)
            if s.Else != nil {
                if elseBlock, ok := s.Else.(*goast.BlockStmt); ok {
                    injectBeforeReturns(elseBlock, deferredStmts)
                }
            }

        case *goast.ForStmt:
            injectBeforeReturns(s.Body, deferredStmts)

        case *goast.RangeStmt:
            injectBeforeReturns(s.Body, deferredStmts)

        case *goast.SwitchStmt:
            if s.Body != nil {
                for _, cc := range s.Body.List {
                    if caseClause, ok := cc.(*goast.CaseClause); ok {
                        injectBeforeReturnsInCase(caseClause, deferredStmts)
                    }
                }
            }

        case *goast.LabeledStmt:
            if blockStmt, ok := s.Stmt.(*goast.BlockStmt); ok {
                injectBeforeReturns(blockStmt, deferredStmts)
            }
        }
    }
}
```

**Critical:** The switch/case walk must iterate `CaseClause.Body` recursively. The if-else walk must handle both branches. Return statements inside closures (nested `*ast.FuncLit`) must ***not*** be touched — they belong to a different function scope.

### 4e. Where injection is called

**In `convertDecl`** — at the end of procedure/function declaration handling (around `declaration.go:177`):

```go
// After building funcDecl.Body, inject deferred statements
t.injectDeferred(funcDecl.Body)
```

**In `convertExpr`** — at the end of `*ast.ProcedureLiteral` handling (around `expression.go:723`):

```go
// After building the FuncLit body, inject deferred statements
// BUT: save/restore deferStack so inner closures don't leak into outer scope
savedStack := t.deferStack
t.deferStack = nil
// ... (body conversion happens here, which pushes onto the new empty stack)
// After converting body:
t.injectDeferred(funcLit.Body)
t.deferStack = savedStack  // restore outer defer stack
```

### 4f. Script mode handling

In `TranspileScript`, the main body is constructed as a flat list of statements in a `func main()`. The same `injectDeferred` call applies after `convertStmt` loop.

---

## 5. Scoping and Variable Capture

### 5a. Direct call (`defer proc(args...)`)

When `defer proc(file)` is transpiled, the call expression `proc(file)` is evaluated as a Go expression statement. Since the arguments are already evaluated and the call is emitted as statements, Go's normal variable scoping applies. The key insight: the transpiler converts the arguments immediately, and the resulting `goast.CallExpr` is stored in the defer stack. When injected before a return, the same `goast.Expr` is reused with whatever variable names were in scope at the `defer` point.

**Important caveat:** If a deferred call references a variable that is later reassigned, the inlined call sees the *latest* value, not the value at the `defer` site. This is different from Go's actual `defer` behavior (which captures by reference at defer time).

**Resolution strategy:** For variable arguments in deferred calls, generate a copy at the defer point:

```go
defer fclose(file)
// transpiles to:
// (at defer point)
__defer_file := file
// (before return)
fclose(__defer_file)
```

This is handled during `convertDeferExpr` for `*ast.Call` by:
1. Analyzing arguments for identifier references
2. For each argument that is a simple identifier (not a literal, not a constant), generate a `__defer_<name>_<counter>` copy assignment
3. Replace the identifier in the saved call expression with the copy variable
4. Prepend the copy assignments to `deferInfo.callStmts`

**Implementation:** A new helper function `captureDeferArgs(call *ast.Call) (preamble []goast.Stmt, newArgs []goast.Expr)` that:
- Iterates `call.Arguments`
- For each `*ast.Identifier` argument, creates `__defer_<name>_<counter>` and generates an assign-and-copy
- For composite expressions (struct field access, index expressions), captures the entire base expression as a temp variable if it contains variable references

This handles the most common case (variable arguments) correctly. Complex expressions with side effects (e.g., `defer f(g())`) are already safe since the argument `g()` evaluates immediately in Go.

### 5b. Closure (`defer { stmts }`)

When an inline closure is deferred, all statements are collected as Go statements. Variable references within the closure use normal Go lexical scoping. Since the closure body is not a Go func literal (it's just statement blocks), it shares the enclosing function's scope. The same copy-capture strategy applies for variables that might be reassigned.

### 5c. Dynamic variables (`dyn`)

Dynamic variables accessed within `defer` work naturally since the transpiler already rewrites dyn reads/writes via `cogDynRead`/`cogDynWrite`. The defer stack stores these rewritten Go expressions, so they're correctly evaluated at inlining time (before return).

---

## 6. Edge Cases

### 6a. Nested functions/closures

Each new function scope (`*ast.ProcedureLiteral`, procedure declaration) has its own `deferStack`. Before converting a nested function body:
1. Save current `deferStack`
2. Create new empty `deferStack`
3. After body conversion + injection, restore saved `deferStack`

This ensures deferred calls in an inner function are injected only into that function's return points, not into the outer function's return points.

### 6b. Multiple return points

The recursive `injectBeforeReturns` walk handles all returns at any nesting depth: `if`/`else` branches, `switch` cases, `for` loops (via `break`-like returns). Each return gets the full deferred preamble.

### 6c. No return value (void procedures)

When a procedure has no return value, `*goast.ReturnStmt{}` (empty) is emitted. The injection algorithm still works — it injects before any `*goast.ReturnStmt{}`. For implicit returns at end of function body, the deferred statements are appended at the end of the block.

### 6d. Panic recovery

The transpiler does **not** handle panic recovery. True `defer` runs even during panics. If panic recovery is needed, users must use Go interop (`@go.defer`). This is an explicit design tradeoff to keep the inlining approach simple. A future enhancement could wrap function bodies in a `recover()` block, but that adds overhead and complexity.

### 6e. Multiple defers are stacked LIFO

Three defers:
```cog
defer a()
defer b()
defer c()
```
The `deferStack` collects `[a(), b(), c()]`.
`buildReversedDeferredStmts()` iterates in reverse: `[c(), b(), a()]`.
This matches Go's LIFO semantics.

### 6f. `defer` in if/for/switch blocks

Since `defer` is parsed as a statement and added to the block's statement list, it naturally appears inside `if`, `for`, `switch` bodies. The transpiler's recursive walk visits these inner blocks and injects deferred statements before returns within them. The key rule: deferred statements from an inner scope are only injected before returns in that same scope's return points. For example:

```cog
if condition {
    defer cleanup()
    return
}
```

The `defer cleanup()` is pushed onto the shared `deferStack` and injected before the `return` inside the `if` block. This is correct because the `if` block is within the same function body.

### 6g. `defer` inside loops

```cog
for item in collection {
    defer cleanup(item)
}
// after loop body — deferred calls are injected before any return
```

This works but has an important semantic difference from Go: in Go, each loop iteration pushes a new defer; in inlined mode, only the last iteration's deferred call is in the stack (each iteration replaces the previous). 

**Correct solution:** When within a loop body, the deferred statements must be accumulated per-iteration. After the loop body, emit the accumulated deferred statements for that iteration.

**Implementation:** Track whether we're inside a loop body with a counter. When converting `defer` inside a loop:
- Don't add to the top-level deferStack
- Instead, add to a per-loop "loop defer stack"
- After the loop body's last statement (before the closing `}`), inject the loop's deferred statements in reverse order

Actually, a simpler correct approach: when a `defer` is inside a loop body, the transpiler wraps the loop iteration in a closure and calls it immediately — or uses an intermediate variable. The simplest correct approach:

For `defer` inside a loop, generate an actual Go `defer` statement (fall back to Go runtime defer). This handles the per-iteration semantics correctly without complex analysis. Tag these defers specially so the transpiler knows not to inline them.

**Simpler and fully correct approach:** Track a `loopDepth int` on the transpiler. When `loopDepth > 0` and a `defer` is encountered, emit it as a real `goast.DeferStmt` instead of inlining. This is a pragmatic hybrid:

```go
if t.loopDepth > 0 {
    // Inside a loop — emit real Go defer for correct per-iteration semantics
    return []goast.Stmt{&goast.DeferStmt{Call: callExpr}}, nil
}
```

---

## 7. Test Plan

### 7a. File: `internal/transpiler/defer_test.go`

Test cases:

| # | Test | Input | Expected Go Output |
|---|------|-------|-------------------|
| 1 | Basic defer with proc call | `defer close(f)` inside proc | Inlined `close(f)` before every return |
| 2 | Multiple defers LIFO | `defer a(); defer b(); defer c(); return` | `c(); b(); a(); return` |
| 3 | Defer before return with value | `defer cleanup(); return result` | `cleanup(); return result` |
| 4 | Nested function | `outer fn { defer a(); inner fn { defer b(); return }; return }` | Inner return gets `b()`, outer gets `a()` before returns |
| 5 | Defer in if/else branches | `if cond { defer a(); return } else { defer b(); return }` | Each branch gets its deferred stmt before return |
| 6 | Defer closure | `defer { x = x + 1 }` | Inlined assignment |  
| 7 | Multiple return points | `defer close(f); if cond { return }; return` | `close(f)` before both returns |
| 8 | Variable capture | `defer close(file)` where file is reassigned | Copy to `__defer_file` at defer point |
| 9 | Defer with no return (void) | `defer a(); stmt; stmt` | `a()` appended at end of block |
| 10 | Defer inside for loop | `for { defer cleanup(x) }` | Real Go `defer cleanup(x)` emitted |
| 11 | Switch case with defer | `defer a(); switch { case 1: return }` | `a()` before case 1's return |
| 12 | Error: invalid defer expr | `defer x + 1` | Parser error |
| 13 | Script mode | `.cogs` with `defer close(f); return` | Inlined in `func main()` |
| 14 | Defer in method | `(var r T).Method { defer r.cleanup(); return }` | Inlined before return |
| 15 | Zero defers | function with no defer | No changes to output |

### 7b. Integration test

Create a `.cogs` script file that uses defer and verify the generated Go compiles and runs correctly:
```cog
// defer_test.cogs
main : proc() = {
    x := 1
    defer { assert(x == 2) }
    x = 2
}
```
Expected: assertion passes because `defer { assert(x == 2) }` is inlined before the implicit return at end of `main`.

---

## 8. Implementation Order

1. **Add token** — `internal/tokens/type.go` + `internal/tokens/lookup.go`
2. **Add AST node** — `internal/ast/defer.go`
3. **Parser integration** — `internal/parser/statement.go` (single case)
4. **Transpiler scaffolding** — `deferStack` field, helper types, `convertDeferExpr`
5. **Statement conversion** — `case *ast.Defer` in `convertStmt`
6. **Inject deferred** — `injectDeferred`, `injectBeforeReturns`, `buildReversedDeferredStmts`
7. **Closure body injection points** — `convertDecl` for procs, `convertExpr` for `ProcedureLiteral`
8. **Variable capture** — argument copy generation for direct calls
9. **Loop detection** — `loopDepth` tracking and fallback to real Go `defer`
10. **Tests** — all test cases from §7
