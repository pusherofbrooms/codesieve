# Agent Usage Guide

`codesieve` is meant to help agents read less code.

Use it to narrow context before opening full files.

## Preferred workflow

1. Run `codesieve index <repo>` once.
2. Use `codesieve search symbol <query>` first.
3. If needed, inspect `codesieve outline <file>`.
4. Fetch exact source with `codesieve show symbol <id>`.
5. Use `codesieve search text <query>` only when symbol search is not enough.
6. Use `codesieve show file <path> --start-line --end-line` as a last fallback.

## Retrieval rules

### Prefer symbol search over file reading

Good:

```bash
codesieve search symbol authenticate --json
```

Avoid reading full files before checking whether the needed symbol already exists.

When you get too many matches, you can narrow symbol search with:

- `--lang=<lang>` (language filter)
- `--kind=<kind>` (e.g. `function`, `method`, `class`)
- `--path-substr=<path>` (only matches in paths containing this substring)
- `--limit=<n>` (maximum results to return)
- `--case-sensitive` (require exact case match on the symbol name)

### Prefer outline before large reads

If multiple candidate symbols are in one file, inspect the outline:

```bash
codesieve outline src/auth.py --json
```

This is usually much cheaper than reading the whole file.

### Prefer exact symbol retrieval

Once you have a symbol ID, fetch only that symbol:

```bash
codesieve show symbol <id> --json
```

### Use text search as fallback

Use text search when:

- the symbol name is unknown
- the target is a constant, string, config key, or comment
- structural parsing did not capture the needed code

Example:

```bash
codesieve search text "AUTH_HEADER" --json
```

You can also narrow text search with:

- `--lang=<lang>` (language filter)
- `--path-substr=<path>` (only matches in paths containing this substring)
- `--limit=<n>` (maximum results to return)
- `--case-sensitive` (treat the query as case-sensitive)

### Read file slices, not whole files

If symbol lookup is insufficient, request only the needed lines:

```bash
codesieve show file src/auth.py --start-line 120 --end-line 180 --json
```

## Recommended decision tree

### If looking for a function, method, class, type, or module entry point

1. `search symbol`
2. `outline`
3. `show symbol`

### If looking for a string, constant usage, config key, SQL fragment, or log line

1. `search text`
2. `show file` for a narrow range

### If many search results are returned

- narrow by `--lang` or `--kind`
- inspect the most likely file outline
- then fetch one symbol at a time

## Goal

The goal is to minimize token use without lowering code understanding quality.

Default behavior should be:

- search first
- outline second
- exact symbol retrieval third
- file slices only when necessary
