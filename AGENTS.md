- The following MCP servers are available:
    - context7: MUST be used for documentation retrieval
    - gopls: MUST be used to interact with Go language server and verify code.
    - github: MUST be used to interact with GitHub.
- There are also multiple Go skills available that SHOULD be used when appriopriate. MUST check documentation before use.

- When running one of the following commands, you MUST prefix with `rtk`: ls, tree, read, git, gh, find, diff, grep, wget, wc, curl,go, golangci-lint
- You SHOULD use the following two `rtk` subcommands for more efficient processing:
    - smart: generate summary
    - json: how JSON structure without values
    - deps: summarize project dependencies
    - summary: run command and show heuristic summary
- You MUST add a test case to verify changes or new code.
- If a `Makefile`, or `Taskfile.yml` is present, read it, and use those commands.

# Golang
- INDENTATION DOES NOT MATTER
- Use of `rtk go build` or `go build` is FORBIDDEN.
- Use `rtk go vet .` instead of `rtk go build .` to check for parser issues.
- Use `rtk go run .` or `rtk go run ./cmd/foo` instead of `rtk go build` to run programs, to avoid leaving behind build artifacts.
- To see source files from a dependency, or to answer questions about a dependency, run `rtk go mod download -json MODULE` and use the returned `Dir` path to read the files.
- Use `rtk go doc foo.Bar` or `rtk go doc -all foo` to read documentation for packages, types, functions, etc.
- Verify if latest training data used lower Go version than locally installed `rtk go version`. If so, get latest Go docs if needed.
- Use `rtk go test` to verify all changes.
- If a `.golang-ci.yml` file is present, run `rtk golangci-lint run` to verify changes are compliant.
