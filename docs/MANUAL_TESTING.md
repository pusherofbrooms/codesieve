# Manual / Real-world Testing

These steps are **optional** and meant for humans running heavier, real-world checks. They are **not** wired into `nix flake check`.

All commands assume you are in the `codesieve` repo root.

## Grafana smoke test

`grafana/grafana` is a good mixed Go + TypeScript/JavaScript repo for exercising indexing, tree-sitter parsing, and search behavior.

### 1. Clone Grafana (outside this repo)

```bash
git clone --depth 1 https://github.com/grafana/grafana.git ~/src/grafana
```

### 2. Run the smoke test script

From the `codesieve` repo root:

```bash
scripts/smoke-test-with-grafana ~/src/grafana
```

What this does:

- builds a local `codesieve` binary (or uses `$CODESIEVE_BIN` if set)
- runs `codesieve index . --json` inside the Grafana repo
  - stores the raw JSON as `.codesieve-grafana-index.json`
  - prints a short summary: files indexed, symbols extracted, duration
- runs a few representative queries:
  - Go auth-related symbols: `search symbol Login --lang=go`
  - TypeScript dashboard symbols: `search symbol Dashboard --lang=typescript --path-substr=public/app/features/dashboard`
  - text search for `AUTH_HEADER`
- writes per-query JSON results under `grafana` as:
  - `.codesieve-grafana-search-login-go.json`
  - `.codesieve-grafana-search-dashboard-ts.json`
  - `.codesieve-grafana-search-auth-header.json`

### 3. What to look for

- **Indexing**
  - runtime (via `duration_ms`)
  - `files_indexed`, `files_updated`, `symbols_extracted`
  - sanity-check DB size at `grafana/.codesieve-grafana.db`
- **Symbol search**
  - are top results for `Login` / `Dashboard` plausible and in expected paths?
  - do scores and ordering look reasonable given the ranking rules?
- **Text search**
  - does `AUTH_HEADER` (or similar) find relevant locations without excessive noise?

You can repeat the test after editing/deleting a file in Grafana to manually inspect incremental indexing behavior.
