# codesieve v1 Spec

## Purpose

`codesieve` is a local CLI for coding agents.

Its job is to reduce token usage while preserving accuracy by letting agents retrieve:

- compact structural summaries of code
- exact symbol bodies on demand
- small file slices when needed
- text matches as a fallback

The tool is intentionally narrow.

It is not designed first for humans, editors, or servers.

---

## Product thesis

Coding agents waste tokens when they read whole files too early.

`codesieve` should help agents follow a smaller retrieval loop:

1. index the repository once
2. search for relevant symbols or text
3. inspect a compact outline
4. fetch only the exact symbol or file slice needed

If this loop works well, agents can understand code with less context and fewer full-file reads.

---

## Primary user

The primary user is a coding agent operating in a local repository.

The CLI should be:

- simple
- predictable
- machine-friendly
- hard to misuse
- stable under `--json`

Human-readable output is useful, but secondary.

---

## Non-goals

Version 1 is **not**:

- an editor backend
- an MCP server
- an HTTP service
- a TUI application
- a human-first code navigation tool
- a full semantic code intelligence platform
- a remote repository indexing service
- an AI summarization system

Do not add architecture for these unless needed for the v1 CLI.

---

## Companion integration

A thin `pi` integration can be a deliverable.

The preferred form is a small `pi` extension that exposes `codesieve` through a few focused tools or commands.

This integration should:

- wrap the local `codesieve` CLI
- keep the command mapping simple and predictable
- avoid reimplementing indexing logic inside the extension
- stay optional and outside the core architecture

Recommended scope for a `pi` extension:

- `codesieve_index`
- `codesieve_search_symbol`
- `codesieve_search_text`
- `codesieve_outline`
- `codesieve_show_symbol`
- `codesieve_show_file`

The extension should be treated as a thin convenience layer for `pi`, not as a second product surface.

---

## Success criteria

`codesieve` v1 succeeds if an agent can:

- index a local repository
- get a compact repo-level outline before deep reads
- find likely symbols quickly
- inspect file structure without reading the whole file
- retrieve exact symbol source by ID (single and batch)
- retrieve a precise file slice by line range
- fall back to text search when symbol search is insufficient
- do all of the above through a small, consistent CLI

---

## Nix deliverables

The repository must ship a `flake.nix` early in the project.

Required flake outputs:

- `packages.<system>.default` to build the `codesieve` binary
- an app so `nix run` works for the CLI
- a `devShell` for project development
- checks for automated validation

The development shell should include the tools needed to work on the project, including:

- Go toolchain
- test tooling
- `bats-core`
- `jq`
- any additional project-specific build or lint tools

Expected workflows:

- `nix build` builds the project
- `nix run` runs the CLI
- `nix develop --command ...` runs development tools
- `nix flake check` runs project checks

Nix support is part of the project deliverable, not optional polish.

---

## Core commands

The command surface should stay small.

### `codesieve index <path>`

Index a local repository or folder.

Examples:

```bash
codesieve index .
codesieve index ~/src/project
```

Minimal flags:

- `--json`
- `--force`
- `--no-gitignore`
- `--max-files`
- `--max-size`

Output should include:

- files indexed
- files skipped by reason
- symbols extracted
- duration
- warnings

### `codesieve search symbol <query>`

Search indexed symbols.

Examples:

```bash
codesieve search symbol authenticate
codesieve search symbol login
```

Minimal optional filters:

- `--lang`
- `--kind`
- `--limit`
- `--json`

Result fields:

- symbol id
- name
- qualified name if available
- kind
- file path
- line
- short signature if available
- score

### `codesieve outline <file>`

Return a compact structural outline for one file.

Examples:

```bash
codesieve outline src/auth.py
codesieve outline internal/api/client.go
```

Minimal flags:

- `--json`

Output should include:

- file path
- language
- hierarchical symbols (top-level and nested)
- line ranges
- symbol ids

### `codesieve show symbol <id>`

Return the exact source and metadata for one symbol.

Examples:

```bash
codesieve show symbol src/auth.py::Auth.login#method
codesieve show symbol <symbol-id>
```

Minimal flags:

- `--context N`
- `--content-only`
- `--verify`
- `--json`

Output should include:

- symbol id
- name
- kind
- file path
- line range
- source content
- signature if available
- optional verification result when `--verify` is used

### `codesieve show symbols <id...>`

Return exact source and metadata for multiple symbols in one call.

Examples:

```bash
codesieve show symbols <id-1> <id-2>
```

Minimal flags:

- `--content-only`
- `--json`

Output should include:

- symbols list with per-symbol source
- per-id errors for unknown IDs

### `codesieve repo outline`

Return a compact repository overview.

Examples:

```bash
codesieve repo outline
```

Minimal flags:

- `--json`

Output should include:

- language breakdown
- top-level directory counts
- symbol kind counts
- index age or staleness hint when cheap to compute

### Milestone 3 optional commands

If promoted by the milestone rule, keep additions narrow:

- `codesieve find importers <file>`
- `codesieve module summary <path-or-module>`

Both should remain local, deterministic, and parser-driven (no AI dependency).

---

## Command design rules

The CLI should follow these rules:

1. Prefer a small number of commands over many specialized commands.
2. Use explicit nouns like `search symbol` and `show symbol`.
3. Keep flags minimal.
4. Make JSON output stable and easy to parse.
5. Return structured failures instead of ambiguous text.
6. Avoid hidden network behavior.

---

## Retrieval model

The intended agent workflow is:

1. `codesieve index <path>`
2. `codesieve repo outline` for a cheap high-level map
3. `codesieve search symbol <query>`
4. `codesieve outline <file>` if more structure is needed
5. `codesieve show symbol <id>` or `codesieve show symbols <id...>` for exact source
6. native `rg` only when structural search is not enough
7. native `read` only when live file verification is needed

This workflow should be documented for agents and supported cleanly by the CLI.

---

## Indexing model

Version 1 should use structural parsing, not deep semantic analysis.

### Required behavior

- discover files in a local repository
- respect `.gitignore` by default
- allow explicit ignore overrides
- skip binaries
- skip oversized files
- hash files for incremental reindexing
- parse supported languages into a normalized symbol model
- store symbol ranges for exact source retrieval

### Not required for v1

- LSP integration
- references
- callers/callees
- implementations
- type hierarchy
- cross-file semantic graph construction

---

## Language support

Version 1 should support a small number of languages well enough to be useful.

Recommended initial languages:

- Go
- Python
- TypeScript / JavaScript

Non-Go parsing uses the official `github.com/tree-sitter/go-tree-sitter` runtime with vendored grammar sources for reproducible Nix builds. See `docs/PARSERS.md` for parser layout, vendoring policy, and extension guidance.

Language support only needs to cover:

- file detection
- structural symbol extraction
- line and byte ranges
- outline generation

More languages can be added later.

### Language extension ergonomics

To keep language additions cheap and consistent, the implementation should include a small extension pattern:

1. **Per-language parser packages and registration** *(done)*
   - Keep a thin central registry.
   - Let each language expose a small spec (language name, extensions, parser function).
   - Avoid one large monolithic parser registration file over time.

2. **Shared parser contract** *(done)*
   - Use one consistent parser function contract across languages.
   - Keep normalization and symbol finalization in shared code instead of per-language duplication.

3. **Reusable Tree-sitter extraction helpers** *(done)*
   - Provide shared traversal/extraction helpers for common node patterns (named declarations, container/member relationships, signature extraction).
   - Keep language files focused on language-specific mapping, not repeated tree walking boilerplate.

4. **Standard fixture convention per language** *(done)*
   - Maintain a consistent fixture layout for language test data.
   - Require the same baseline assertions for each new language: indexing counts, outline hierarchy quality, and exact `show symbol` source retrieval.

### Next language priorities (post-v1)

Recommended order for near-term additions, balancing team usage, parser availability, and implementation risk:

1. Terraform / OpenTofu (HCL)
2. Bash
3. PHP
4. C / C++

Keep additions parser-driven and local-only. Each language should ship with fixture coverage for symbol extraction and outline quality before promotion.

---

## Architecture

Keep the architecture simple:

```text
Discovery -> Parse -> Normalize -> Store -> Query
```

### Components

#### Discovery

Responsibilities:

- walk local files safely
- apply ignore rules
- reject unsupported or oversized files
- produce file metadata

#### Parse

Responsibilities:

- parse files with tree-sitter or equivalent structural parsers
- extract symbols and containers
- extract signatures and doc comments when cheap
- compute line and byte ranges

#### Store

Responsibilities:

- persist repository metadata
- persist file metadata
- persist symbols and ranges
- support text search
- support incremental updates

#### Query

Responsibilities:

- search symbols
- build file outlines
- build repo outline summaries
- retrieve exact symbol content
- retrieve multiple symbols in one call
- optionally resolve lightweight importer relationships

---

## Storage

Use SQLite as the local store.

Reasons:

- simple deployment
- good point lookups
- easier incremental updates than flat files
- good fit for a local CLI

### Suggested storage layout

- `~/.codesieve/index.db`

Optional later:

- content caches
- per-repo metadata directories

### Minimum tables

- `repos`
- `files`
- `symbols`
- `index_runs`
- `diagnostics`

### Minimum file fields

```text
File {
  id
  repo_id
  path
  language
  hash
  size_bytes
  indexed_at
  parse_status
}
```

### Minimum symbol fields

```text
Symbol {
  id
  repo_id
  file_id
  name
  qualified_name
  kind
  parent_symbol_id
  signature
  documentation
  start_line
  end_line
  start_byte
  end_byte
}
```

---

## Symbol IDs

Version 1 needs IDs that are easy for agents to pass back into the CLI.

Use two forms:

1. **Human-readable ID** for display and agent reuse
2. **Canonical internal ID** for storage

The exact scheme can be simple in v1 as long as:

- IDs are unique within a repo
- `show symbol <id>` is reliable
- JSON output always returns the canonical ID field

Do not over-engineer rename tracking in v1.

---

## Ranking

Keep ranking simple and explainable.

Recommended symbol ranking inputs:

- exact name match
- prefix match
- fuzzy name match
- path match
- kind match

Avoid advanced graph or semantic scoring in v1.

---

## Output contracts

Machine-readable output matters most.

### JSON success envelope

```json
{
  "ok": true,
  "data": {},
  "meta": {
    "repo": "/path/to/repo",
    "timing_ms": 12
  }
}
```

### JSON error envelope

```json
{
  "ok": false,
  "error": {
    "code": "SYMBOL_NOT_FOUND",
    "message": "No symbol matched the provided id"
  },
  "meta": {
    "timing_ms": 4
  }
}
```

### Output principles

- stable field names
- compact payloads
- predictable shapes
- no unnecessary prose in JSON mode

---

## Diagnostics

Version 1 should report skipped or degraded behavior clearly.

Suggested diagnostic codes:

- `SKIPPED_BINARY`
- `SKIPPED_TOO_LARGE`
- `SKIPPED_IGNORED`
- `SKIPPED_SECRET`
- `PARSE_FAILED`
- `FILE_NOT_INDEXED`
- `SYMBOL_NOT_FOUND`
- `INVALID_RANGE`

Diagnostics should be available in command output and stored for later inspection.

---

## Security and safety

Version 1 should be safe by default.

Requirements:

- local-only operation
- no network calls during normal indexing/querying
- path traversal prevention
- safe file path normalization
- binary file skipping
- file size limits
- symlinks disabled by default
- secret files skipped by default (priority)

Secret-file skipping is a v1 safety priority, not optional polish.

Default secret-path detection should be path-based (no required full-content secret scanning in the indexing path). Implementations may allow additional user-defined secret path globs via environment variable configuration.

---

## Agent usage guide

The project should include a concise guide for agents.

The guide should teach this retrieval discipline:

1. search symbols first
2. inspect outlines before reading large files
3. fetch exact symbol bodies instead of whole files
4. use text search as fallback
5. read file slices only when symbol-based retrieval is insufficient

This guide is part of the product, not an afterthought.

---

## Implementation plan

### Milestone 1

Deliver:

- CLI skeleton
- local discovery
- ignore handling
- secret-file exclusion with `SKIPPED_SECRET` diagnostics (priority)
- SQLite setup
- structural parsing for Go, Python, TS/JS
- normalized symbol extraction
- `index`
- `search symbol`
- `outline`
- `show symbol`
- stable JSON envelopes

### Milestone 2

Deliver:

- `show symbols` batch symbol retrieval
- hierarchical `outline` JSON output
- `repo outline`
- incremental reindexing via file hashing
- diagnostics for skipped and failed files
- improved ranking quality

### Milestone 3

Deliver only if clearly needed:

- `find importers` based on lightweight import extraction
- package/module summaries derived from indexed symbols (non-AI)
- broader language coverage
- improved parsing fidelity
- limited semantic enrichment if it measurably improves agent retrieval

---

## Future directions

For deferred and out-of-scope ideas, see `FUTURE_DIRECTIONS.md`.

`SPEC.md` should stay focused on the executable v1 scope and conditional Milestone 3 additions.

---

## Testing

Follow `docs/TESTING.md` when implementing and extending the project.

---

## Bottom line

`codesieve` v1 should be the simplest useful local retrieval CLI for coding agents.

It should do a few things well:

- index code
- search symbols
- outline files
- return exact symbol source

If it helps agents stop reading entire files unnecessarily, it is succeeding.
