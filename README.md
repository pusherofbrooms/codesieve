# codesieve

`codesieve` is a local code indexer that lets agents fetch precise symbols, outlines, and file slices.


## Usage

For a brief overview of commands and flags, run:

```bash
codesieve --help
```

For agent-focused guidance, use:

- `SKILL.md` (canonical agent instructions)
- `docs/TESTING.md` (automated testing strategy + optional real-world smoke tests)

Typical retrieval flow:

```bash
codesieve index . --json
codesieve repo outline --json
codesieve search symbol Login --json
codesieve outline src/auth.go --json
codesieve show symbol <id> --json
```

## Secret path skipping

`codesieve index` skips common secret-like paths (for example `.env`, key/cert files, and names containing `secret` outside doc extensions) and records `SKIPPED_SECRET` diagnostics.

You can add custom skip globs with:

```bash
CODESIEVE_SECRET_PATH_PATTERNS="*.crt,config/private/*" codesieve index . --json
```

The variable accepts a comma-separated list of glob patterns and is matched case-insensitively against both file basename and relative path.

## Integration into Claude Code, OpenCode, and Codex

Place the SKILL.md into a path where your coding agent can see it.

## Build and Install

If you have a recent Go toolchain and a C compiler installed (for Tree-sitter's C code), you can build directly with Go.

### Remote install via `go install`

Once the module path and upstream repository are aligned, you will be able to install
`codesieve` directly from GitHub:

```bash
go install github.com/pusherofbrooms/codesieve/cmd/codesieve@latest
```

This builds the CLI and places the `codesieve` binary in `$GOBIN` (or `$GOPATH/bin` if
`GOBIN` is not set). Ensure that directory is on your `PATH` so agents and shells can run
`codesieve` without a local build.

### Local build from this repo

```bash
# From the repo root
go build -o codesieve ./cmd/codesieve

# Or install into your Go bin dir (GOBIN/GOPATH/bin)
go install ./cmd/codesieve
```

Then run:

```bash
./codesieve --help
```

## Build and run (Nix)

This repo ships a Nix flake.

Common commands:

```bash
# Build the codesieve binary
nix build

# Run the CLI via the flake app
nix run

# Open a dev shell with Go, bats, jq, sqlite, clang
nix develop
```

You can also run tools inside the dev shell non-interactively:

```bash
nix develop --command go test ./...
nix develop --command bats tests/bats
nix flake check
```

