# Configuration

This page collects runtime configuration for `codesieve`.

## Database location

`codesieve` stores index data in SQLite.

- Default path: `~/.codesieve/index.db`
- Override with: `CODESIEVE_DB_PATH`

Example:

```bash
CODESIEVE_DB_PATH=/tmp/codesieve.db codesieve index . --json
```

### Shared user-level DB (default)

By default, one user-level DB stores many repositories.

- each repository is tracked by normalized absolute path
- indexing updates only the matching repository records
- queries are scoped to your current working repository

This is the simplest setup and usually needs no `.gitignore` changes.

### Repo-local DBs

If you set `CODESIEVE_DB_PATH` to a file inside a repository (common in CI or per-project isolation), add it to `.gitignore`.

Suggested patterns:

```gitignore
# codesieve local indexes
.codesieve*.db
*.codesieve.db
```

Example repo-local usage:

```bash
CODESIEVE_DB_PATH=.codesieve.db codesieve index . --json
```

## Secret path skip patterns

`codesieve index` skips built-in secret-like paths and supports custom globs via:

- `CODESIEVE_SECRET_PATH_PATTERNS`

The value is a comma-separated list of glob patterns matched case-insensitively against basename and relative path.

Example:

```bash
CODESIEVE_SECRET_PATH_PATTERNS="*.crt,config/private/*" codesieve index . --json
```
