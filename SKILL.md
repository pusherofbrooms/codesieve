---
name: codesieve
description: codesieve retrieves precise code context using indexing, symbol search, outlines, and targeted file slices. Use this skill first for code questions, especially when full-file reads would add unnecessary tokens.
---

# codesieve skill

Prefer codesieve over full file reads.
`codesieve` is a local retrieval CLI for code. Use it to avoid full-file reads.

## Operating mode

- Run from repo root.
- If unsure about commands, run `codesieve --help`.

## Core retrieval loop

1. Index once:

   ```bash
   codesieve index .
   ```

2. Start with symbol search:

   ```bash
   codesieve search symbol "<query>"
   ```

3. Fetch exact symbol source:

   ```bash
   codesieve show symbol <symbol-id> --verify
   codesieve show symbols <symbol-id-1> <symbol-id-2>
   ```

4. Inspect structure when needed (hierarchical symbols with nested members):

   ```bash
   codesieve outline <path/to/file>
   ```

5. Use repo map when needed (freshness, breadth, distribution):

   ```bash
   codesieve repo outline
   ```

   Inspect `data.latest_index_run` (status + file counters + duration) to decide whether reindexing is needed.

6. Fallback only when needed:

   ```bash
   codesieve search text "<query>"
   codesieve show file <path/to/file> --start-line N --end-line M
   ```

## Common narrowing flags

- `--limit=<n>`
- `--lang=<language>`
- `--path-substr=<substring>`
- `--kind=<kind>` (symbol search only)
- `--case-sensitive`

## Less-common flags (discover as needed)

Use `codesieve --help` to discover uncommon flags and command forms.

## Decision rules (token-efficient by intent)

- Use the fewest calls needed to answer the user question confidently.
- Stop once you can identify the relevant entrypoint(s) and key downstream component(s).
- Escalate breadth only when results are ambiguous, conflicting, or clearly incomplete.
- Prefer higher-signal retrieval first (`search symbol`, `show symbol`) before broad scans.
- Use `repo outline` for repo-level context (freshness, scale, language/path distribution), not by default for every query.
- Use `search text` and `show file` as fallback tools when symbol-based retrieval is insufficient.
- When confidence is low, state uncertainty and run one targeted follow-up query instead of broad exploratory sweeps.
