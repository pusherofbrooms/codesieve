# Parser and Tree-sitter Notes

`codesieve` uses structural parsers for symbol extraction.

Current parser strategy:

- Go uses the standard library `go/parser`
- Python uses the official Tree-sitter Go runtime plus a vendored grammar
- JavaScript uses the official Tree-sitter Go runtime plus a vendored grammar
- TypeScript / TSX use the official Tree-sitter Go runtime plus vendored grammars
- Java uses the official Tree-sitter Go runtime plus a vendored grammar
- Rust uses the official Tree-sitter Go runtime plus a vendored grammar
- Bash uses the official Tree-sitter Go runtime plus a vendored grammar
- Terraform/OpenTofu (HCL) uses the official Tree-sitter Go runtime plus a vendored grammar
- YAML uses the official Tree-sitter Go runtime plus a vendored grammar
- JSON uses the official Tree-sitter Go runtime plus a vendored grammar

## Why grammar sources are vendored

Non-Go parsing uses the official Tree-sitter Go runtime:

- `github.com/tree-sitter/go-tree-sitter`

However, the upstream grammar Go bindings rely on relative C includes into grammar source trees. In this repository's Nix packaging flow, direct use of those upstream bindings is not reliable under `buildGoModule`.

To keep builds reproducible and hermetic under Nix, grammar sources are vendored into this repository under `third_party/`, and tiny local wrapper packages expose the language handles used by the parser.

Only the minimal build-required subset is kept per grammar tree (C parser/scanner sources, required headers, `tree-sitter.json`, `LICENSE*`, and `.codesieve-vendor-meta`).

This is intentional.

Do not replace vendored grammars with direct upstream grammar Go imports unless the Nix packaging story has been revalidated.

## Layout

Vendored grammars live under:

- `third_party/tree-sitter-python`
- `third_party/tree-sitter-javascript`
- `third_party/tree-sitter-typescript`
- `third_party/tree-sitter-java`
- `third_party/tree-sitter-rust`
- `third_party/tree-sitter-bash`
- `third_party/tree-sitter-hcl`
- `third_party/tree-sitter-yaml`
- `third_party/tree-sitter-json`

Local wrappers live under:

- `internal/tslang/python`
- `internal/tslang/javascript`
- `internal/tslang/typescript`
- `internal/tslang/java`
- `internal/tslang/rust`
- `internal/tslang/bash`
- `internal/tslang/hcl`
- `internal/tslang/yaml`
- `internal/tslang/json`

Application parser code lives under `internal/parser/`.

For Terraform/OpenTofu, `codesieve` recognizes `.tf`, `.tfvars`, `.hcl`, `.tf.json`, and `.tfvars.json`.

Shared Tree-sitter extraction helpers (for common traversal and symbol-shape patterns) live under `internal/parser/core/`.

For TypeScript and JavaScript, shared extraction logic lives under `internal/parser/languages/tsjs/` to avoid duplicated walkers.

## Adding a new Tree-sitter-backed language

1. Pick the official upstream grammar repository and version.
2. Vendor the required grammar sources under `third_party/`.
3. Add a local wrapper in `internal/tslang/<language>/` that exposes a small `Language()` function or variant functions.
4. Implement symbol extraction in `internal/parser/languages/<language>/`.
   - Reuse helpers in `internal/parser/core/` for node traversal, signature extraction, and container/member symbol shaping where applicable.
5. Register the language in `internal/parser/registry.go`:
   - language name
   - supported file extensions
   - parser function
6. Add parser tests and, if needed, fixture coverage.
7. Validate the full build and test flow.

## Updating an existing vendored grammar

1. Select the new upstream version.
2. Replace the vendored source tree under `third_party/` (prefer `scripts/vendor-grammar`, which auto-prunes to required files).
3. Confirm wrapper include paths still match the vendored layout.
4. Re-run formatting and tests.
5. Recompute `vendorHash` if Nix reports a mismatch.

## Validation commands

Use the project's Nix workflows:

```bash
nix develop --command go test ./...
nix develop --command bats tests/bats
nix flake check
```

For build validation:

```bash
nix build
```

## Nix note

`flake.nix` uses `proxyVendor = true;` because the official Tree-sitter Go runtime includes C sources and headers that must remain available during the Go build.
