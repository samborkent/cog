Cog Project Review — Biggest Problems
I've read the entire codebase. Here are the most significant issues, ordered by impact:

1. Parser panics instead of recovering gracefully
advance() in parser.go:130 calls panic("reached end of token stream") when it hits the token boundary. The main.go entry point runs the parser in a goroutine with a recover() to catch this, but this is a fragile pattern. Any unexpected token sequence (e.g. unterminated block, missing closing paren) can trigger an unrecoverable crash. A real error recovery strategy — like synchronizing to the next statement boundary and continuing — would let more errors be reported per parse and make the tool far more useful (especially once you build an LSP).

2. Package-level mutable state in the transpiler
ifLabelCounter in block.go:15 is a var at package scope. This means:

Transpiling multiple files in sequence produces non-deterministic label names.
Concurrent transpilation (e.g. for multi-file projects) will race.
Tests that exercise if-block labels will be order-dependent.
Move it to a field on the Transpiler struct.

3. No error recovery / single-error-abort in many paths
Beyond the panic issue, many parse functions return nil on the first error (e.g. parseCallArguments, parseForStatement, parseSwitch). This means the parser usually reports only one error per compilation. Languages with good developer UX report many errors at once. This compounds the panic problem — a misplaced token early in the file means zero information about the rest.

4. Go AST coupling makes multi-backend extremely difficult
The transpiler emits go/ast nodes directly. Your stated goal of eventually targeting Zig or LLVM means you'd need to write entirely new backends from scratch. There's no intermediate representation (IR) between the Cog AST and Go's AST. Introducing a lightweight Cog IR that captures semantics (lowered types, resolved identifiers, control flow) would let you share analysis/optimization between backends and keep the Go backend as just one code generator.

5. Lexer limitation: Go's text/scanner
The lexer wraps Go's text/scanner, which was designed for Go-like tokens. This constrains your syntax to things text/scanner can handle. For example:

Custom operators or multi-character tokens require special-case handling (you already do this for ==, !=, etc.)
The scanner's built-in number parsing, string escaping, and comment handling may not match Cog's intended semantics
Switching to a hand-written lexer would give you full control and isn't much more code than the current wrapper

6. Type cache keyed on String() — collision risk
In transpiler/type.go, the typeCache is keyed on the result of Type.String(). If two structurally different types have the same string representation (which is plausible for complex generic types or aliased types), you'll silently return the wrong cached Go AST expression. Using the type's identity (pointer or a more discriminating key) would be safer.

7. Missing bounds check in parseCallArguments
In parser/call.go, the loop for i := 0; ... indexes into procType.Parameters[i] without checking i < len(procType.Parameters). Passing too many arguments to a typed function will panic with an index-out-of-range.

8. Dynamic variable implementation via context.WithValue
Dynamic variables are transpiled into context.WithValue / context.Value calls with type-asserted keys. This works, but:

Each dynamic variable access does a linear scan through the context chain
The runtime cost scales with the number of dynamic variables in scope
It conflates Cog's scoping semantics with Go's context, which has different cancellation and lifetime semantics
This is fine for now, but for performance-comparable code, a closure-captured-variable or explicit parameter-passing approach would be cheaper.

9. No source position tracking through transpilation
The /​/line directives appended in attachLineDecl provide some mapping, but they're text comments appended after AST construction. There's no structured source map. When the user gets a Go compile error, they'll see Cog line numbers (good), but for runtime errors (stack traces), the mapping may be incomplete or misleading. This becomes critical for debugging.

10. Test coverage is integration-only
All tests are end-to-end integration tests (lex → parse → transpile → compile → run). There are no unit tests for individual parser rules, transpiler conversions, or type system operations. This means:

Diagnosing test failures requires tracing through the entire pipeline
Edge cases in individual components are hard to verify
Refactoring any component is risky without fine-grained regression tests

Smaller issues worth noting
- parseBuiltinSet error messages say "parseBuiltinMap" — copy-paste error in parser/builtins.go
- @go call ignores return types — GoCallExpression sets ValueType: types.None, so the type system can't check how the result is used
- No else if support — the parser only handles else { block }, not else if expr { block }
- p.error scope parameter is inconsistent — some calls pass scope strings, others don't, making error messages inconsistent
- parseForStatement error messages say "parseIfStatement" — copy-paste in for_statement.go:105

What's working well
- The three-pass parser design (pre-register types → find globals → parse) handles forward references cleanly
- The func vs proc distinction with automatic context injection is a nice safety feature
- The stringTo(*strings.Builder) pattern on AST nodes is clean and allocation-efficient
- The integration test suite covers a good range of error cases
- The transpiler/component package keeps Go AST construction DRY
