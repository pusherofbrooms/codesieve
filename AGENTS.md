# General instructions.

## Conversational behavior

- Questions should be answered in the shortest reasonable way.
- Be concise. Brevity is wit.

## Nix Discipline
This computer runs Nix package manager with flakes.

Use:
- `nix run` for one-off commands
- `nix shell` with the --command option if the package doesn't have a default app.
- `nix develop` with the --command option to interact with tools defined for a devShell
- `nix build` to build the current project.
- If a new dependency is needed, propose a change to flake.nix.

You don't have access to a PTY, so no interactive nix shell.

- No old-style Nix commands (`nix-env`, `nix-shell`, `nix-channel`, etc.)
- No `nix profile`
- No global installs
- No imperative package management

The system must remain reproducible and declarative.

## Follow this development workflow:

1. Define a minimal, explicit spec (inputs, outputs, constraints, edge cases).
2. Implement a complete, coherent first version that reasonably satisfies the spec (avoid placeholder or stub implementations).
3. Write tests to validate real behavior, including edge cases and failure modes.
4. Execute tests and analyze failures.
5. Apply targeted fixes only to the failing logic or incorrect tests; do not rewrite unrelated code.
6. Ensure fixes generalize beyond the specific test cases (avoid overfitting).
7. Expand test coverage after core functionality passes.

Tests are for validation and refinement. Do not generate tests before a meaningful implementation exists.

## Version control

- After tests pass, commit.
- After documentation changes, commit.

## Repository cleanup helpers

If you need to remove rebuildable local directories or vendored grammar trees, prefer the repo scripts so the operation is easy to audit:

- `scripts/clean-local-artifacts`
  - removes `.bats-codesieve-bin`, `.codesieve-test.db`, and `vendor/`
- `scripts/reset-vendor`
  - removes `vendor/`
- `scripts/remove-vendored-grammar <python|javascript|typescript|bash|hcl|yaml|json|java|rust|csharp>`
  - removes one vendored grammar tree under `third_party/`
- `scripts/prune-vendored-grammars [all|python|javascript|typescript|bash|hcl|yaml|json|java|rust]`
  - keeps only build-required files in vendored grammar trees

