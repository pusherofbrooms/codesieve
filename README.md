# codesieve

`codesieve` is a local code indexer that lets agents fetch precise symbols, outlines, and file slices.

## Quickstart

```bash
# 1) Build and run once (repo root)
go build -o codesieve ./cmd/codesieve

# 2) Index a repository
./codesieve index . --json

# 3) Explore symbols
./codesieve repo outline --json
./codesieve search symbol Login --json
./codesieve show symbol <id> --verify --json
```

For command/flag reference:

```bash
./codesieve --help
```

## Install

### Go install (remote)

Install from GitHub:

```bash
go install github.com/pusherofbrooms/codesieve/cmd/codesieve@latest
```

This places `codesieve` in `$GOBIN` (or `$GOPATH/bin` if `GOBIN` is unset).

### Local build (this repo)

```bash
# From the repo root
go build -o codesieve ./cmd/codesieve

# Or install into your Go bin dir (GOBIN/GOPATH/bin)
go install ./cmd/codesieve
```

For storage and environment variable configuration (including `CODESIEVE_DB_PATH` and repo-local `.gitignore` guidance), see [`docs/CONFIGURATION.md`](docs/CONFIGURATION.md).

### Build/run with Nix

This repo ships a Nix flake.

From GitHub (no local clone required):

```bash
# Run codesieve directly from GitHub
nix run github:pusherofbrooms/codesieve#codesieve -- --help

# Build the binary package from GitHub
nix build github:pusherofbrooms/codesieve#codesieve
```

From a local clone:

```bash
# Build the codesieve binary
nix build .#codesieve

# Run the CLI via the flake app
nix run .#codesieve -- --help

# Open a dev shell with Go, bats, jq, sqlite, clang
nix develop
```

Non-interactive dev-shell commands:

```bash
nix develop --command go test ./...
nix develop --command bats tests/bats
nix flake check
```

Include in your own flake:

```nix
{
  inputs.codesieve.url = "github:pusherofbrooms/codesieve";

  outputs = { self, nixpkgs, codesieve, ... }:
    let
      system = "x86_64-linux";
      pkgs = import nixpkgs { inherit system; };
    in {
      devShells.${system}.default = pkgs.mkShell {
        packages = [
          codesieve.packages.${system}.codesieve
        ];
      };
    };
}
```

## Common retrieval workflow

`outline` returns hierarchical symbols, including nested methods/functions.

```bash
codesieve index . --json
codesieve repo outline --json
codesieve search symbol Login --json
codesieve outline src/auth.go --json
codesieve show symbol <id> --verify --json
codesieve show symbols <id-1> <id-2> --json
```

`repo outline --json` includes `latest_index_run` stats from SQLite (status, duration, and per-run file/symbol counters) so agents can quickly inspect index freshness and recent indexing behavior.

## Supported languages (v1)

- Go
- Python
- TypeScript / JavaScript
- Bash
- YAML (including CloudFormation-oriented symbol extraction)
- JSON (including CloudFormation-oriented symbol extraction)

Go parsing uses the standard library parser. Python, TypeScript/JavaScript, Bash, YAML, and JSON use Tree-sitter with vendored grammars for reproducible builds.

For parser layout, vendoring policy, and extension guidance, see `docs/PARSERS.md`.

## Secret path skipping

`codesieve index` skips common secret-like paths (for example `.env`, key/cert files, and names containing `secret` outside doc extensions) and records `SKIPPED_SECRET` diagnostics.

You can add custom skip globs with:

```bash
CODESIEVE_SECRET_PATH_PATTERNS="*.crt,config/private/*" codesieve index . --json
```

The variable accepts a comma-separated list of glob patterns and is matched case-insensitively against both file basename and relative path.

## Agent integration docs

For agent-focused guidance:

- `SKILL.md` (canonical agent instructions)
- `docs/TESTING.md` (automated testing strategy + optional real-world smoke tests)

To integrate into Claude Code, OpenCode, or Codex, place `SKILL.md` somewhere your coding agent can read.
