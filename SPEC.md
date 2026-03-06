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
- find likely symbols quickly
- inspect file structure without reading the whole file
- retrieve exact symbol source by ID
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

### `codesieve search text <query>`

Fallback text search across indexed content.

Examples:

```bash
codesieve search text "jwt secret"
codesieve search text "AUTH_HEADER"
```

Minimal flags:

- `--lang`
- `--limit`
- `--json`

Result fields:

- file path
- line
- snippet
- match range if available

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
- top-level symbols
- nested symbols when cheap to extract
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
- `--json`

Output should include:

- symbol id
- name
- kind
- file path
- line range
- source content
- signature if available

### `codesieve show file <path>`

Return a file slice.

Examples:

```bash
codesieve show file src/auth.py --start-line 10 --end-line 40
```

Minimal flags:

- `--start-line`
- `--end-line`
- `--json`

Output should include:

- file path
- requested line range
- source content

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
2. `codesieve search symbol <query>`
3. `codesieve outline <file>` if more structure is needed
4. `codesieve show symbol <id>` for exact source
5. `codesieve search text <query>` only when structural search is not enough
6. `codesieve show file <path> --start-line --end-line` as a final fallback

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
- support text search over indexed content

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

Language support only needs to cover:

- file detection
- structural symbol extraction
- line and byte ranges
- outline generation

More languages can be added later.

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
- search text
- build file outlines
- retrieve exact symbol content
- retrieve file slices

---

## Storage

Use SQLite as the local store.

Reasons:

- simple deployment
- good point lookups
- supports FTS
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
- FTS tables for symbol/text search

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
- secret files skipped when reasonable

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

- `search text`
- `show file` with line ranges
- incremental reindexing via file hashing
- diagnostics for skipped and failed files
- improved ranking quality

### Milestone 3

Deliver only if clearly needed:

- broader language coverage
- improved parsing fidelity
- limited semantic enrichment if it measurably improves agent retrieval

---

## Future directions

Keep these out of v1 unless they become necessary:

- semantic providers and LSP integration
- references/callers/callees
- type relationships
- remote repository indexing
- editor integrations
- HTTP services
- MCP adapters
- TUI
- AI summaries

These belong in a separate future directions document.

---

## Testing

Follow `docs/TESTING.md` when implementing and extending the project.

---

## Bottom line

`codesieve` v1 should be the simplest useful local retrieval CLI for coding agents.

It should do a few things well:

- index code
- search symbols
- search text
- outline files
- return exact source slices

If it helps agents stop reading entire files unnecessarily, it is succeeding.
