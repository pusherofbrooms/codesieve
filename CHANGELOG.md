# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project follows [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- C# language support (`.cs`, `.csx`) via vendored `tree-sitter-c-sharp` grammar, including symbol extraction for namespaces, imports, types, members, and overload-aware methods.
- PHP language support (`.php`) via vendored `tree-sitter-php` grammar, including symbol extraction for namespaces, imports, classes/interfaces/traits/enums, methods/constructors, properties, constants, enum cases, and top-level functions.

## [0.1.0] - 2026-03-17

### Added
- Initial `codesieve` CLI with core commands:
  - `index`
  - `search symbol`
  - `search text`
  - `outline`
  - `repo outline`
  - `show symbol`, `show symbols`, `show file`
- SQLite-backed local index with incremental reindexing and parser-version-aware invalidation.
- Symbol extraction for:
  - Go
  - Python
  - TypeScript / JavaScript
  - Java
  - Rust
  - Bash
  - HCL (Terraform/OpenTofu)
  - YAML
  - JSON
- CloudFormation-oriented symbol extraction for YAML and JSON templates.
- Index diagnostics and skip behavior for ignored files, binary files, oversized files, secret-like paths, and Terraform artifacts.
- Nix flake support for build, dev shell, and checks.
- Test suites for Go packages and Bats CLI integration.

[Unreleased]: https://github.com/pusherofbrooms/codesieve/compare/v0.1.0...main
[0.1.0]: https://github.com/pusherofbrooms/codesieve/releases/tag/v0.1.0
