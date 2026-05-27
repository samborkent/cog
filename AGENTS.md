# Cog Language — Transpiler Context

**Module**: `github.com/samborkent/cog` · **Go version**: 1.26.3+ (experimental arenas) · **Branch**: `feature/any`

## Build & Test

```bash
task vet
```

```bash
task test
```

```bash
task bench
```

```bash
task compile
```

```bash
task transpile
```

```bash
task run
```

## Project Structure

- **`cmd/main.go`**: CLI entry point — flags `-file`, `-debug`, `-write`, `-replace-local-cog`
- **`internal/lexer/`**: `lexer.go` — windowed ring-buffer scanner over `text/scanner.Scanner`, `SkipBody()` for deferred parsing
- **`internal/parser/`**: recursive descent parser — `parser.go` (entry), `statement.go`, `declaration.go`, `type_alias.go`, `method.go`, `match.go`, `for_statement.go`, `if_statement.go`, `switch.go`, `import.go`, `expression.go`, `ebnf_parser.go`
- **`internal/ast/`**: flat AST (arena-allocated `[]Node` + `[]Expr` indexed by `NodeIndex`/`ExprIndex`) — `ast.go`, `identifier.go`, `call.go`, `infix.go`, `prefix.go`, `suffix.go`, `block.go`, `procedure_literal.go`, `match_case.go` + 20+ literal nodes (`int8_literal.go`..`complex128_literal.go`)
- **`internal/types/`**: type system — `kind.go`, `basic.go`, `alias.go`, `struct.go`, `enum.go`, `interface.go`, `procedure.go`, `generics.go`, `union.go`, `array.go`, `slice.go`, `map.go`, `set.go`, `option.go`, `result.go`, `either.go`, `tuple.go`, `pointer.go`, `error.go`, `any.go`
- **`internal/transpiler/`**: Go code generation — `transpiler.go`, `declaration.go`, `statement.go`, `expression.go`, `block.go`, `match.go`, `print.go`, `type.go`, `arena.go`, `comment.go`
- **`internal/transpiler/component/`**: reusable Go AST helpers — `node.go`, `literal.go`, `context.go`, `stdlib.go`, `gc.go`, `arena.go`, `convert.go`, `cache.go`, `builtins.go`
- **`builtin/`**: runtime — `print.go`, `if.go`
- **`editors/vscode/`**: VS Code extension — `package.json`, `language-configuration.json`, `syntaxes/cog.tmLanguage.json`

## Language Features (Implemented)

| Feature | Syntax | Status |
|---------|--------|--------|
| Function decl | `name : proc(...) = { }` | ✅ |
| Pure function | `name : func(...) T = { }` | ✅ |
| Typed var | `name : uint64 = 10` | ✅ |
| Type alias | `MyType ~ []int8` | ✅ |
| Generic alias | `List<T ~ any> ~ []T` | ✅ |
| Generic instantiation | `List<int32>` | ✅ |
| Interfaces | `Stringer ~ interface { String : func() utf8 }` | ✅ |
| Methods | `Foo.Method : func() utf8 = { }` | ✅ |
| Enum | `Color ~ enum<uint8> { Red := 1, Blue := 2 }` | ✅ |
| Error type | `MyErr ~ error<utf8> { NotFound := "not found" }` | ✅ |
| Struct | `Point ~ struct { x : int64; export Y : int64 }` | ✅ |
| Option | `x : int64?` with `if x? { }` | ✅ |
| Result | `x : int64 ! MyErr` with `if x? { } else { x! }` | ✅ |
| Either | `x : int64 ^ utf8` | ✅ |
| Match (type) | `match t := expr { case int64: case utf8: }` | ✅ |
| For loops | `for { }` · `for item { }` · `for v, k in map { }` | ✅ |
| Switch | `switch x { case val: }` · `switch { case expr: }` | ✅ |
| Go interop | `@go.strings.ToUpper("hi")` | ✅ |
| Builtins | `@print`, `@if`, `@cast`, `@ref`, `@slice`, `@map`, `@set` | ✅ |
| Dynamic scope | `dyn` keyword with `context.WithValue` | ✅ |
| Arena allocation | automatic via `GOEXPERIMENT=arenas` | ✅ |
| Script mode | `.cogs` files → `package main` | ✅ |
| Multi-file | local `import` with package selector | ✅ |
| Labels | `label: for { break label }` | ✅ |

## Type System

- **Basic kinds enum** in `internal/types/kind.go` (`Kind`, `Lookup` map)
- **22 builtin basics** in `internal/types/basic.go`: `int8`..`int128`, `uint8`..`uint128`, `float16`..`float64`, `complex32`..`complex128`, `bool`, `ascii`, `utf8`
- **10 builtin constraints** in `internal/types/generics.go`: `any`, `comparable`, `ordered`, `number`, `int`, `uint`, `float`, `complex`, `string`, `signed`, `summable` — each a `types.Union`
- **Helper predicates** in `internal/types/is.go`: `IsInt`, `IsUint`, `IsFloat`, `IsComplex`, `IsNumber`, `IsString`, `IsSigned`, `IsSummable`, `IsBool`, `IsComparable`, `IsIndexable`, `IsIterator`, `IsPointer`, `IsFixed`
- **Generic instantiation** in `internal/types/alias.go`: `Instantiate(typeArgs map[string]Type) Type` with recursive `SubstituteType`
- **Constraint checking** in `internal/types/helpers.go`: `Satisfies(concrete, constraint Type) bool`, `AssignableTo`, `Equal`
- **Composite types**: `Array`, `Slice`, `Map`, `Set`, `Tuple`, `Union`, `Either`, `Option`, `Result`, `Reference` (pointer), `Procedure`, `Interface`, `Struct`

## Important Conventions

- **Never use `go build .`** → use `go vet .` for compilation checks, `go run .` or `go run ./cmd/main.go` for execution
- **Always use `GOEXPERIMENT=arenas`** when running any Go command (`go vet`, `go test`, `go run`)
- **Use `rtk` prefix** for `go vet`, `go test`, and `go run` (e.g. `GOEXPERIMENT=arenas rtk go test ./...`)
- **Test files mirror source**: `assignment_test.go` ⇔ `assignment.go`, `block_test.go` ⇔ `block.go` — add test cases to existing test files, don't create new ones
- **Benchmark tracking**: run `task stats` to record lines-of-code and coverage to `stats.txt`
- **Transpiler helpers**: use `go/printer` for Go output testing — helpers are in `internal/transpiler/helpers_test.go` (`transpile`, `transpileWithPrint`, `mustContain`, `mustNotContain`, `mustFailTranspile`)
- **Parser helpers**: use `internal/parser/helpers_test.go` (`parse`, `parseShouldError`, `stmtAs`)
- **Must-check analysis**: options/results require `?` check before value/error access — enforced in `internal/parser/symbol_table.go` via `checkState` (`checkValue`, `checkError`)

<!-- caliber:managed:pre-commit -->
## Before Committing

**IMPORTANT:** Before every git commit, you MUST ensure Caliber syncs agent configs with the latest code changes.

First, check if the pre-commit hook is already installed:
```bash
grep -q "caliber" .git/hooks/pre-commit 2>/dev/null && echo "hook-active" || echo "no-hook"
```

- If **hook-active**: the hook handles sync automatically — just commit normally. Tell the user: "Caliber will sync your agent configs automatically via the pre-commit hook."
- If **no-hook**: run Caliber manually before committing:
  1. Tell the user: "Caliber: Syncing agent configs with your latest changes..."
  2. Run: `caliber refresh && git add CLAUDE.md .claude/ .cursor/ .cursorrules .github/copilot-instructions.md .github/instructions/ AGENTS.md CALIBER_LEARNINGS.md .agents/ .opencode/ 2>/dev/null`
  3. After it completes, briefly tell the user what Caliber updated. Then proceed with the commit.

**Valid `caliber refresh` options:** `--quiet` (suppress output) and `--dry-run` (preview without writing). Do not pass any other flags — options like `--auto-approve`, `--debug`, or `--force` do not exist and will cause errors.

**`caliber config`** takes no flags — it runs an interactive provider setup. Do not pass `--provider`, `--api-key`, or `--endpoint`.

If `caliber` is not found, read `.agents/skills/setup-caliber/SKILL.md` and follow its instructions to install Caliber.
<!-- /caliber:managed:pre-commit -->

<!-- caliber:managed:learnings -->
## Session Learnings

Read `CALIBER_LEARNINGS.md` for patterns and anti-patterns learned from previous sessions.
These are auto-extracted from real tool usage — treat them as project-specific rules.
<!-- /caliber:managed:learnings -->

<!-- caliber:managed:model-config -->
## Model Configuration

Recommended default: `claude-sonnet-4-6` with high effort (stronger reasoning; higher cost and latency than smaller models).
Smaller/faster models trade quality for speed and cost — pick what fits the task.
Pin your choice (`/model` in Claude Code, or `CALIBER_MODEL` when using Caliber with an API provider) so upstream default changes do not silently change behavior.

<!-- /caliber:managed:model-config -->

<!-- caliber:managed:sync -->
## Context Sync

This project uses [Caliber](https://github.com/caliber-ai-org/ai-setup) to keep AI agent configs in sync across Claude Code, Cursor, Copilot, and Codex.
Configs update automatically before each commit via `caliber refresh`.
If the pre-commit hook is not set up, read `.agents/skills/setup-caliber/SKILL.md` and follow the setup instructions.
<!-- /caliber:managed:sync -->
