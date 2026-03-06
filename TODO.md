# TODO

## V1 foundation

- [x] Add `flake.nix` with package, app, devShell, and checks
- [x] Scaffold Go CLI and SQLite-backed storage
- [x] Implement initial commands: `index`, `search symbol`, `search text`, `outline`, `show symbol`, `show file`
- [x] Add Go parsing and tree-sitter-backed Python and TS/JS parsing
- [x] Improve `.gitignore` handling edge cases and diagnostics coverage
- [ ] Add richer ranking and search filters
- [x] Add incremental reindex cleanup and smarter stale-file handling
- [x] Add Bats CLI contract tests and fixture repositories
- [ ] Add integration tests for SQLite indexing/query behavior
- [ ] Add thin `pi` integration wrapper
