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

## Phase 4 — Two-Pass Body Resolution

**TL;DR**: Replace `FindGlobals` + `ParseOnly` with `ParseGlobals` (full parse, defers bodies and unresolvable `{}`-expressions) + `ParseBodies` (seeks to deferred offsets, parses in-place). Eliminates the redundant re-lex. Shared symbol table constraint preserved — ParseGlobals runs for ALL files before ParseBodies starts for ANY file.

---

**Pipeline**
```
ParseGlobals (all files, shared symbol table) →
ValidateGlobals (after all files) →
[compile imports, wire exports] →
ParseBodies (all files) →
Transpile (all files)
```

---

**Steps**

### A. Body & Expression Deferral

1. In `primary.go` Procedure case: when `globalsPass == true`, capture `Offset()`, call `SkipBody()`, store `DeferredBody` node as the body. Don't push param/typeParam scopes.
2. Create `DeferredExpr` expression node — `{ Token tokens.Token; Offset uint32; TypeHint types.Type }`, implements `Expr`. `TypeHint` stores the expected type context (forward alias) so `ParseBodies` can re-pass it to expression parsing once it's resolved.
3. Add `SetExpr(i ExprIndex, e Expr)` to AST — `a.exprs[i] = e`. Same contract as `SetNode`.
4. In `primary.go` LBrace case: when `globalsPass == true` and the expected type is unresolved (forward alias whose `Derived` is `types.None`), capture `Offset()`, call `SkipBody()`, return `DeferredExpr` with the type hint. Covers struct literals, enum value expressions, and any other `{}`-expression whose type isn't yet concrete.

### B. Global Registration During ParseGlobals

5. `parseDeclaration`: when `globalsPass`, call `DefineGlobal` instead of `Define`
6. `parseTypeAlias`: when `globalsPass`, call `DefineGlobal` instead of `Define`. Register early for recursive types.
7. `parseMethod`: when `globalsPass`, call `DefineMethod` + attach to receiver struct. Body defers naturally via step A.1.

### C. Restructure API

8. Rename `ParseOnly` → `ParseGlobals` — sets `globalsPass=true`, runs full parse, returns AST with `DeferredBody` / `DeferredExpr` placeholders
9. Extract validation into `ValidateGlobals()` — checks unresolved forward stubs. Called by orchestrator after ALL files.
10. New `ParseBodies(ctx) error`:
    - Walk nodes: find `DeferredBody`, `SeekTo` offset, reconstruct scopes (type params → receiver → params), parse block, `SetNode` to replace
    - Walk exprs: find `DeferredExpr`, `SeekTo` offset, call `p.expression(ctx, deferredExpr.TypeHint)`, `SetExpr` to replace. By this point the forward alias has resolved, so the type hint now points to the concrete type.
11. Remove `FindGlobals()` — subsumed by ParseGlobals

### D. DeferredBody Metadata

12. Add `Receiver *ast.Identifier` to `DeferredBody` — needed to reconstruct receiver scope during ParseBodies
13. Type params + params accessible from `ProcedureLiteral.ProcedureType` (no extra storage needed)

### E. Orchestrator (cmd/main.go)

14. Replace `findGlobals()` + `ParseOnly()` with `ParseGlobals()` per file → `ValidateGlobals()` → import compilation → `ParseBodies()` per file
15. `Parse()` convenience (tests): does ParseGlobals + ValidateGlobals + ParseBodies sequentially

### F. Cleanup

15. Make scripts work with new pipeline.
17. Remove `FindGlobals`, `findGlobalDecl`, `findGlobalType`, `findGlobalMethod`, `findScriptImports`
18. Remove/repurpose `globals.go` (keep only `ValidateGlobals`)

---

**Relevant files**
- primary.go — body deferral in `case *types.Procedure` (~line 497), expression deferral in `case tokens.LBrace`
- deferred_body.go — add `Receiver` field
- deferred_expr.go — new file: `DeferredExpr` expression node
- ast.go — add `SetExpr(i ExprIndex, e Expr)`
- declaration.go — `DefineGlobal` when `globalsPass`
- method.go — `DefineMethod` + struct attachment during `globalsPass`
- parser.go — rename `ParseOnly`→`ParseGlobals`, new `ParseBodies` (walks both nodes and exprs), update `Parse`
- globals.go — extract `ValidateGlobals`, delete rest
- main.go — orchestrator restructure

**Verification**
1. `task vet`
2. `task test` — covers single-file (`Parse`) and multi-file paths
3. Manually verify forward refs across files still resolve
4. Verify method bodies can reference other methods (forward + cross-file)
5. Verify enum with forward-referenced struct value type (e.g. `enum<planet>` where `planet` defined later)
6. Verify global struct literal with forward-referenced type

**Decisions**
- `ParseGlobals` replaces BOTH `FindGlobals` and `ParseOnly` (single lex pass per file)
- Validation after ALL files (not per-file) — cross-file forward stubs resolve during other files' ParseGlobals
- `DeferredBody` is transient — replaced by `SetNode` during `ParseBodies`
- `DeferredExpr` is transient — replaced by `SetExpr` during `ParseBodies`. The forward alias in `TypeHint` resolves lazily, so by the time `ParseBodies` runs, it points to the concrete type.
- Script mode unchanged — bodies still deferred mechanically, definition order still enforced
- `Parse()` kept as convenience for tests (all 3 steps on single parser)
