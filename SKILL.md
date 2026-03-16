---
name: codesieve
description: Token-efficient local code indexing and retrieval for coding agents.
---

# codesieve skill

`codesieve` is a local retrieval CLI for code. Use it to avoid full-file reads.

## Operating mode

- Run from repo root.
- Prefer `--json` and parse output.
- If unsure about commands, run `codesieve --help`.

## Core retrieval loop

1. Index once:

   ```bash
   codesieve index . --json
   ```

2. Search symbols first:

   ```bash
   codesieve search symbol "<query>" --json
   ```

3. Inspect structure before large reads:

   ```bash
   codesieve outline <path/to/file> --json
   ```

4. Fetch exact symbol source:

   ```bash
   codesieve show symbol <symbol-id> --json
   ```

5. Fallback only when needed:

   ```bash
   codesieve search text "<query>" --json
   codesieve show file <path/to/file> --start-line N --end-line M --json
   ```

## Common narrowing flags

- `--limit=<n>`
- `--lang=<language>`
- `--path-substr=<substring>`
- `--kind=<kind>` (symbol search only)
- `--case-sensitive`

## Less-common flags (discover as needed)

Use `codesieve --help` to discover uncommon flags and command forms.

## Goal

Default behavior should be:

- search first
- outline second
- exact symbol retrieval third
- text search and file slices last
