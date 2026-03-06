# codesieve

`codesieve` is a local code indexer that lets agents fetch precise symbols, outlines, and file slices.

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

## Build without Nix

If you have a recent Go toolchain and a C compiler installed (for Tree-sitter's C code), you can build directly with Go:

```bash
# From the repo root
go build -o codesieve ./cmd/codesieve

# Or install into your GOPATH/bin (if configured)
go install ./cmd/codesieve
```

Then run:

```bash
./codesieve help
```

For development tooling (tests, Bats, jq, etc.), you can either:

- use Nix as described above, or
- install those tools via your OS package manager.

## Usage

For a brief overview of commands and flags, run:

```bash
codesieve help
```

For agent-focused guidance on how to use `codesieve` effectively, see:

- `docs/AGENT_USAGE.md`
- `docs/MANUAL_TESTING.md` (optional real-world smoke tests)
