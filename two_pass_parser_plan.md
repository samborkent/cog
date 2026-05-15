## Plan: Deferred Body Parsing Optimization

Eliminate the redundant re-lex by merging passes. The first pass builds the full AST but defers procedure bodies (storing a `DeferredBody` node with the byte offset of `{`). The second pass seeks the lexer to each deferred offset and parses the block in-place.

---

### Phase 1: Lexer byte offset tracking

1. Add `Offset uint32` to `tokens.Pos` struct
2. Populate it in `lexer.scanNext()` from `s.Pos().Offset` (text/scanner.Position already has it)
3. Add `SeekTo(offset uint32)` on Lexer — seeks `src`, re-inits scanner, clears ring buffer, calls `Step()`

### Phase 2: AST infrastructure (*parallel with Phase 1*)

4. Create `DeferredBody` node — `{ Token tokens.Token; Offset uint32 }`, implements `Node`
5. Make `Block` implement `Node` — add `Pos()`, `Hash()`, `StringTo()`, plus `AddBlock(*Block) NodeIndex` constructor
6. Add `SetNode(i NodeIndex, n Node)` to AST — `a.nodes[i] = n`
7. Change `ProcedureLiteral.Body` from `*Block` to `NodeIndex` — update constructor, `StringTo`, and transpiler access (expression.go lines 718-760)

### Phase 3: Merge parser passes

8. Restructure `Parse()`: `globalsPass=true` → run today's `ParseOnly` logic (no Reset) → `globalsPass=false` → `resolveDeferredBodies()`
9. Remove `FindGlobals()` call — forward reference mechanism (current `NewForwardAlias` stubs) handles type resolution inline
10. In `primary.go` Procedure case: when `globalsPass`, capture `offset := p.lex.This().Offset`, `skipScope(ctx)`, return `ProcedureLiteral` with `DeferredBody` NodeIndex

### Phase 4: Body resolution

11. `resolveDeferredBodies(ctx)` — iterate AST exprs, find `ProcedureLiteral` with `DeferredBody`, seek lexer, push param/type-param scopes from `ProcedureType`, `parseBlockStatement`, `SetNode` to replace in-place
12. Remove `p.lex.Reset()` — no longer needed

---

**Key files**: token.go, lexer.go, block.go, ast.go, procedure_literal.go, primary.go, parser.go, expression.go + new `deferred_body.go`

**Verification**: `task vet` after each phase, `task test`

---

**Open items to validate:**

1. **`scanner.Pos().Offset` semantics** — need to verify it gives the byte offset of the token *start* (not end). If end, we'd subtract `len(tokenText)`.

2. **Method receiver scope** — during `resolveDeferredBodies`, method bodies need receiver context. May need to store parent declaration info on `DeferredBody` or `ProcedureLiteral` to know which receiver scope to push.

3. **Multi-file `ParseOnly`** — currently used when `FindGlobals` runs across multiple files first. The new design should still support this mode (call `parseMain` after cross-file symbol table is populated).