# Plan: Two Pass Parser

Eliminate the redundant re-lex by merging passes. The first pass builds the full AST but defers procedure bodies (storing a `DeferredBody` node with the byte offset of `{`). The second pass seeks the lexer to each deferred offset and parses the block in-place.

---

## Phase 1: Lexer byte offset tracking

1. Add a `Lexer.Offset() uint32` method that returns `l.s.Pos().Offset` (current scanner position).
2. Add `SeekTo(offset uint32)` on Lexer — seeks src, re-inits scanner, clears ring buffer, calls scanNext()

## Phase 2: AST infrastructure (*parallel with Phase 1*)

4. Create `DeferredBody` node — `{ Token tokens.Token; Offset uint32 }`, implements `Node`
5. Make `AddBlock(*Block) NodeIndex` constructor
6. Add `SetNode(i NodeIndex, n Node)` to AST — `a.nodes[i] = n`. It is fine if it panics when `i` is out of bounds. Compiler programmer should be dilligent in not use this wrong.
7. Change `ProcedureLiteral.Body` from `*Block` to `NodeIndex` — update constructor, `StringTo`, and transpiler access (expression.go lines 718-760)

## Phase 3: Forward Value References

Enable order-independent top-level value declarations. During `globalsPass`, when an identifier cannot be resolved in expression context, create a forward value stub (analogous to `NewForwardAlias` for types). The stub's `ValueType` resolves lazily once the referenced symbol is defined later in the file.

---

### Steps

1. In `primary.go` Identifier case: when `p.globalsPass == true` and `Resolve` fails, instead of emitting "undefined identifier", register a forward value stub:
   - Create `*ast.Identifier` with `Token`, `Qualifier: QualifierImmutable`, `Global: true`, `ValueType: types.None`
   - Call `p.symbols.DefineGlobal(ident)` — registers with `ScanScope`
   - Return an `ast.Identifier` expression node referencing this stub
   - The stub's `ValueType` will be filled in when the real declaration is encountered later

2. In `symbols.go` `DefineGlobal`: when overwriting an existing `ScanScope` symbol (the stub), copy the new identifier's `ValueType` and `Qualifier` into the existing stub so that all references to it see the resolved info. This mirrors how forward type aliases resolve — the `Identifier` pointer is shared, so mutating it updates all referents.

3. Post-pass validation: after `parseFile` completes with `globalsPass=true`, iterate `ScanScope` symbols and error on any that still have `ValueType == types.None` — these are genuinely undefined identifiers.

---

**Relevant files**

- primary.go — forward value stub creation in Identifier case
- symbol_table.go — `DefineGlobal` update for stub resolution
- parser.go — post-pass undefined symbol check

**Verification**

- `task vet`
- `task test`
- Add test: declaration referencing a later-declared global value resolves correctly

---

## Phase 4 — Merge Parser Passes + Body Resolution

Merge `FindGlobals` + `ParseOnly` into a single pass for `Parse()`. During this pass, procedure bodies are skipped (storing `DeferredBody` placeholders with byte offsets). After the pass completes (all symbols registered), `resolveDeferredBodies` seeks the lexer to each deferred offset and parses the block in-place. Multi-file path stays unchanged.

---

**Steps**

### A. Parser State Setup

1. Add `currentReceiver *ast.Identifier` to `Parser` struct
2. In `parseMethod`: set `p.currentReceiver = receiver` before `parseTypedDeclaration`, clear after (defer)
3. Add `Receiver *ast.Identifier` field to `DeferredBody` — needed for scope reconstruction
4. Update `NewDeferredBody` constructor to accept receiver param

### B. Defer Bodies During Globals Pass

5. In primary.go Procedure case, add early-return when `p.globalsPass`:
   - `offset := p.lex.Offset()` (byte offset of `{`)
   - `p.lex.SkipBody()` (lands on token after `}`)
   - Create `DeferredBody` with token + offset + receiver
   - Return `NewProcedureLiteral(procToken, t, deferredIdx)`
   - No scope pushes/pops needed

### C. Restructure Parse()

6. Extract core parsing logic from `ParseOnly` into private `parseFile(ctx, fileName) error`
7. Rewrite `Parse()`: `globalsPass=true` → `parseFile` (no Reset) → `globalsPass=false` → `resolveDeferredBodies(ctx)`
8. Rewrite `ParseOnly()`: Reset + `parseFile` (with `globalsPass=false`, no deferral)

### D. Implement resolveDeferredBodies (*depends on A, B, C*)

9. New file internal/parser/resolve.go:
   - Iterate all exprs (`1..LenExprs()`), find `ProcedureLiteral` with `DeferredBody` body
   - For each: `SeekTo(db.Offset)`, push receiver/typeParam/param scopes, set `currentReturnType`, call `parseBlockStatement`, pop scopes, `SetNode` to replace

### E. Cleanup

10. Remove `FindGlobals()` call from `Parse()` — single-file no longer needs it
11. Keep `FindGlobals()` intact for multi-file external callers

---

**Relevant files**

- parser.go — restructure `Parse()`/`ParseOnly()`, extract `parseFile()`
- primary.go — globalsPass early-return in Procedure case (~line 497)
- method.go — set/clear `currentReceiver` (line ~89)
- deferred_body.go — add `Receiver` field, update constructor
- New: `internal/parser/resolve.go` — `resolveDeferredBodies` implementation

**Verification**

1. `task vet`
2. `task test` — existing tests cover both single-file (`Parse`) and multi-file (`FindGlobals` + `ParseOnly`) paths

**Decisions**

- `DeferredBody` stores receiver (not `ProcedureLiteral`) — only needed during resolution, `DeferredBody` is transient
- Multi-file path unchanged — `FindGlobals` + `ParseOnly` kept for external callers
- Script mode — body deferral fires mechanically (same code path), but scripts don't benefit from it since they enforce definition order (no forward refs). Import scanning happens inline during `parseFile` — imports are parsed as encountered, same timing as current `ParseOnly`.
- Forward alias mechanism handles type forward references during `globalsPass=true`
- Forward value references (Phase 3) handle value forward references during `globalsPass=true` — stubs resolve when the real declaration is encountered