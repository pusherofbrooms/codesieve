# General instructions.

## Nix Discipline
This computer runs Nix package manager with flakes.

Use:
- `nix run` for one-off commands
- `nix shell` with the --command option if the package doesn't have a default app.
- `nix develop` with the --command option to interact with tools defined for a devShell
- `nix build` to build the current project.
- If a new dependency is needed, propose a change to flake.nix.

Sadly, you don't have access to a PTY, so no interactive nix shell.

- No old-style Nix commands (`nix-env`, `nix-shell`, `nix-channel`, etc.)
- No `nix profile`
- No global installs
- No imperative package management

The system must remain reproducible and declarative.