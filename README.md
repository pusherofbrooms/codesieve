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

## Usage

For a brief overview of commands and flags, run:

```bash
codesieve help
```

For agent-focused guidance on how to use `codesieve` effectively, see:

- `docs/AGENT_USAGE.md`
- `docs/MANUAL_TESTING.md` (optional real-world smoke tests)
