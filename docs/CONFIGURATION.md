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

## Secret path skip patterns

`codesieve index` skips built-in secret-like paths and supports custom globs via:

- `CODESIEVE_SECRET_PATH_PATTERNS`

The value is a comma-separated list of glob patterns matched case-insensitively against basename and relative path.

Example:

```bash
CODESIEVE_SECRET_PATH_PATTERNS="*.crt,config/private/*" codesieve index . --json
```
