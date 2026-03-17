# Testing Strategy

Follow this strategy when implementing `codesieve`.

## Rules

- Practice TDD for core logic.
- Write fast Go tests for deterministic behavior.
- Use `bats-core` for CLI contract tests.
- Add agent-facing evaluations after the core CLI is stable.
- Do not rely on manual testing as the primary validation method.

## Test layers

### 1. Go unit tests

Write unit tests first for deterministic components:

- discovery and ignore behavior
- path normalization and safety checks
- parser normalization
- symbol extraction
- per-language parser mapping (language-local `internal/parser/languages/<lang>/*_test.go`)
- source range slicing
- ranking basics
- ID generation
- JSON envelope construction
- diagnostic and error mapping

Prefer table-driven tests.

### 2. Go integration tests

Use Go integration tests for subsystem behavior:

- SQLite schema and migrations
- indexing fixture repositories
- incremental reindex behavior
- query behavior over stored indexes

Keep these tests local, repeatable, and fast.

### 3. CLI contract tests with Bats

Use `bats-core` to test the public CLI.

Cover:

- command success and failure
- exit codes
- stdout/stderr behavior
- `--json` validity and shape
- end-to-end workflows across commands

Use fixture repositories and `jq` for JSON assertions.

Do not push complex business logic assertions into shell when Go tests can express them better.

### 4. Agent evaluations

Add agent evaluations after the CLI is mechanically solid.

Use structured tasks against fixture repositories.

Measure:

- task success
- correct command/tool choice
- number of retrieval steps
- fallback frequency
- bytes or tokens returned

The goal is not only correctness. The goal is reduced retrieval volume without loss of answer quality.

## Fixtures

Maintain small, intentional fixture repositories for supported languages.

Each fixture repo should include:

- top-level and nested symbols
- imports
- duplicate or similar names
- constants and strings for text search
- ignored files
- binary or oversized file cases
- known expected search and outline results

Keep fixtures small enough to understand at a glance.

## Output discipline

Test for compactness, not just correctness.

Assert that:

- outlines stay concise and preserve hierarchy (top-level plus nested symbols)
- search results do not include unnecessary payload
- symbol retrieval returns exact source
- file slices respect requested ranges
- JSON output remains compact and stable

## Delivery order

Implement tests in this order:

1. unit tests
2. integration tests
3. Bats CLI tests
4. agent evaluations

For each feature:

1. add or update a fixture
2. write a failing Go test
3. implement the feature
4. add or update a Bats test for the CLI contract
5. later add an agent evaluation if the feature affects retrieval behavior

## Tooling

Preferred tools:

- Go test framework for unit and integration tests
- `bats-core` for CLI tests
- `jq` for JSON assertions in CLI tests

Run build and test commands through Nix outputs when available.

Use:

- `nix build` for builds
- `nix flake check` for project checks
- `nix develop --command ...` for devShell tools

## Optional manual smoke test (real-world repo)

Manual smoke tests are optional and are **not** wired into `nix flake check`.

### Grafana smoke test

`grafana/grafana` is a good mixed Go + TypeScript/JavaScript repo for exercising indexing, parser behavior, and query quality.

1. Clone Grafana outside this repo:

   ```bash
   git clone --depth 1 https://github.com/grafana/grafana.git ~/src/grafana
   ```

2. From the `codesieve` repo root, run:

   ```bash
   scripts/smoke-test-with-grafana ~/src/grafana
   ```

   The script builds (or uses `$CODESIEVE_BIN`), runs indexing, prints a short summary, and writes JSON outputs inside the Grafana checkout:

   - `.codesieve-grafana-index.json`
   - `.codesieve-grafana-search-login-go.json`
   - `.codesieve-grafana-search-dashboard-ts.json`
   - `.codesieve-grafana-search-auth-header.json`

3. Inspect:

   - indexing runtime and counters (`duration_ms`, `files_indexed`, `symbols_extracted`)
   - DB size at `grafana/.codesieve-grafana.db`
   - whether top symbol/text hits are plausible

## Bottom line

Test the engine first, the CLI second, and the agent workflow third.

Ship only behavior that is covered by automated tests.
