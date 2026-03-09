---
name: codesieve
description: Token-efficient local code indexing and retrieval for coding agents.
---

# codesieve skill

## Purpose

`codesieve` lets agents explore a local codebase without loading entire files.
It indexes the repository once, then retrieves only the symbols or file slices needed.

Use it to:

- index the current project
- search for likely symbols
- inspect file structure
- fetch exact symbol bodies
- fetch small file slices
- fall back to text search when needed

## Environment

- The `codesieve` CLI is available on the system PATH.
- Commands are run in the repository root.
- `CODESIEVE_DB_PATH` may be set to choose a specific SQLite index file; if unset, the default is used.

All commands should be executed with `--json` and their JSON output parsed programmatically.

## Commands

Use these commands via the host's shell/tool mechanism:

- **Index repository**

  ```bash
  codesieve index . --json
  ```

  Optional flags:
  - `--force` (reindex all files)
  - `--no-gitignore`
  - `--max-files N`
  - `--max-size BYTES`

- **Search symbols**

  ```bash
  codesieve search symbol "<query>" --json
  ```

  Optional filters:
  - `--lang <language>`
  - `--kind <kind>` (e.g. function, method, class)
  - `--limit N`
  - `--path-substr <substring>`

- **Search text**

  ```bash
  codesieve search text "<query>" --json
  ```

  Optional filters:
  - `--lang <language>`
  - `--limit N`
  - `--path-substr <substring>`

- **Outline a file**

  ```bash
  codesieve outline <path/to/file> --json
  ```

- **Show a symbol by ID**

  ```bash
  codesieve show symbol <symbol-id> --json
  ```

  Optional:
  - `--context N` (lines of surrounding context)

- **Show a file slice**

  ```bash
  codesieve show file <path/to/file> --start-line N --end-line M --json
  ```

## Retrieval discipline

When reasoning about code, follow this order:

1. **Index first** (once per repo or after changes):

   - If the project is not yet indexed, run `codesieve index . --json`.
   - Note that indexing can take a long time on large code bases.

2. **Prefer symbol search:**

   - Use `codesieve search symbol` to find functions, methods, classes, and types by name.
   - Use filters (`--lang`, `--kind`, `--path-substr`) to narrow results.

3. **Use outlines before reading large files:**

   - Use `codesieve outline <file>` to understand structure without loading the whole file.

4. **Fetch exact symbol bodies:**

   - Use `codesieve show symbol <id>` to retrieve only the symbol's source (with optional context), not the entire file.

5. **Fallback to text search and file slices:**

   - If symbol search is insufficient, use `codesieve search text`.
   - Use `codesieve show file` with a narrow line range instead of reading entire files.

Avoid reading whole files through other mechanisms when `codesieve` can provide a smaller, more relevant slice.
