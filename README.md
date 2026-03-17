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

To print the installed version:

```bash
./codesieve version
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

## Command and flag highlights

Use `codesieve <command> --help` for full details. Commonly useful flags:

### Indexing

```bash
codesieve index . --json --no-gitignore --max-files=20000 --max-size=2097152
```

- `--no-gitignore`: ignore `.gitignore` rules during indexing
- `--max-files=<n>`: cap discovered file count
- `--max-size=<bytes>`: cap per-file size for indexing

Incremental indexing is parser-version aware: grammar/parser upgrades can trigger selective reparsing without requiring a full DB wipe.

### Text search

```bash
codesieve search text "token.*expires" --regex --context-lines=2 --json
```

- `--regex`: treat query as regular expression
- `--context-lines=<n>`: include surrounding lines in text results

### Symbol search ranking behavior

`search symbol` ranking prefers exact and qualified-name matches and de-prioritizes common non-primary paths (for example tests, vendored code, and generated output directories).

### Show commands

```bash
codesieve show symbol <id> --content-only
codesieve show symbol <id> --context=3 --json
codesieve show file path/to/file --start-line=20 --end-line=80 --content-only
```

- `show symbol --content-only`: print only symbol source
- `show symbol --context=<n>`: include surrounding lines for symbol retrieval
- `show file --start-line/--end-line`: fetch a precise file slice
- `show file --content-only`: print only file content slice

## Supported languages (v1)

- Go
- Python
- TypeScript / JavaScript
- Java
- Rust
- Bash
- Terraform / OpenTofu (HCL)
- YAML (including CloudFormation-oriented symbol extraction)
- JSON (including CloudFormation-oriented symbol extraction)

Go parsing uses the standard library parser. Python, TypeScript/JavaScript, Java, Rust, C#, Bash, Terraform/OpenTofu (HCL), YAML, and JSON use Tree-sitter with vendored grammars for reproducible builds.

For parser layout, vendoring policy, and extension guidance, see `docs/PARSERS.md`.

## Secret path and artifact skipping

`codesieve index` skips common secret-like paths and records `SKIPPED_SECRET` diagnostics.

By default (`CODESIEVE_SECRET_PATH_MODE=balanced`), high-confidence secret files are skipped (for example `.env`, key/cert material, and explicit secret patterns), while common source files are not skipped only because their basename contains `secret`.

For stricter behavior, set `CODESIEVE_SECRET_PATH_MODE=strict` to also skip non-doc files whose basename contains `secret`.

It also skips common Terraform/OpenTofu generated artifacts (`.terraform/`, `*.tfstate`, `*.tfstate.backup`) and records `SKIPPED_ARTIFACT` diagnostics.

Other common skip diagnostics during indexing include:

- `SKIPPED_IGNORED` (matched by `.gitignore`, unless `--no-gitignore`)
- `SKIPPED_TOO_LARGE` (exceeded `--max-size`)
- `SKIPPED_BINARY` (binary file detection)

You can add custom deny/allow globs with:

```bash
CODESIEVE_SECRET_PATH_PATTERNS="*.crt,config/private/*" codesieve index . --json
CODESIEVE_SECRET_PATH_ALLOW_PATTERNS="config/*secret*.json" codesieve index . --json
```

These variables accept comma-separated glob patterns and are matched case-insensitively against both file basename and relative path.

## Agent integration docs

For agent-focused guidance:

- `SKILL.md` (canonical agent instructions)
- `docs/TESTING.md` (automated testing strategy + optional real-world smoke tests)

To integrate into Claude Code, OpenCode, or Codex, place `SKILL.md` somewhere your coding agent can read.

