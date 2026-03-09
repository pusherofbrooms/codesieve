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

If you have a recent Go toolchain and a C compiler installed (for Tree-sitter's C code), you can build directly with Go.

### Local build from this repo

```bash
# From the repo root
go build -o codesieve ./cmd/codesieve

# Or install into your Go bin dir (GOBIN/GOPATH/bin)
go install ./cmd/codesieve
```

Then run:

```bash
./codesieve help
```

### Remote install via `go install`

Once the module path and upstream repository are aligned, you will be able to install
`codesieve` directly from GitHub:

```bash
go install github.com/pusherofbrooms/codesieve/cmd/codesieve@latest
```

This builds the CLI and places the `codesieve` binary in `$GOBIN` (or `$GOPATH/bin` if
`GOBIN` is not set). Ensure that directory is on your `PATH` so agents and shells can run
`codesieve` without a local build.

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
