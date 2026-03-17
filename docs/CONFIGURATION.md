# Configuration

This page collects runtime configuration for `codesieve`.

## Database location

`codesieve` stores index data in SQLite.

- Default path: `~/.codesieve/index.db`
- Override path with `CODESIEVE_DB_PATH`

Examples:

```bash
CODESIEVE_DB_PATH=/tmp/codesieve.db codesieve index . --json
CODESIEVE_DB_PATH=.codesieve.db codesieve index . --json
```

If you use a repo-local DB path, add an ignore rule:

```gitignore
.codesieve*.db
*.codesieve.db
```

## Schema migrations and compatibility

`codesieve` tracks DB schema state with:

- `PRAGMA user_version` (current schema version)
- `schema_migrations` table (applied migration records)

On startup, `codesieve` runs forward migrations automatically.

This improves long-term DB stability when index metadata evolves, and avoids requiring users to wipe their DB for routine upgrades.

## Secret path skip controls

`codesieve index` skips built-in secret-like paths and supports custom controls:

- `CODESIEVE_SECRET_PATH_MODE`
  - `balanced` (default): skip high-confidence secret paths; avoid skipping common source files only because their names contain `secret`
  - `strict`: also skip non-doc files whose basename contains `secret`
- `CODESIEVE_SECRET_PATH_PATTERNS`
  - comma-separated denylist globs (case-insensitive) matched against basename and relative path
- `CODESIEVE_SECRET_PATH_ALLOW_PATTERNS`
  - comma-separated allowlist globs (case-insensitive) matched against basename and relative path
  - only affects soft basename-contains-`secret` heuristics; it does not override hard secret checks (for example `.pem`, `.env`, key/cert files)

Examples:

```bash
# Add extra deny patterns
CODESIEVE_SECRET_PATH_PATTERNS="*.crt,config/private/*" codesieve index . --json

# Keep a strict default heuristic
CODESIEVE_SECRET_PATH_MODE=strict codesieve index . --json

# Allow known-safe config naming while retaining hard secret checks
CODESIEVE_SECRET_PATH_ALLOW_PATTERNS="config/*secret*.json" codesieve index . --json
```

## Indexing controls (CLI flags)

`codesieve index` also supports runtime controls:

- `--no-gitignore`
- `--max-files=<n>`
- `--max-size=<bytes>`

Example:

```bash
codesieve index . --json --no-gitignore --max-files=20000 --max-size=2097152
```
