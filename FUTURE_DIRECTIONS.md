# Future Directions

This file is the backlog for ideas intentionally deferred from `SPEC.md`.

Items here are out of v1 scope by default. Promote only when they clearly improve the core retrieval loop while preserving local, deterministic behavior.

## Deferred areas

### Semantic analysis

- LSP integration
- references
- callers/callees
- implementations
- type hierarchy
- semantic ranking signals
- provider diagnostics and capability reporting

### Integrations

- editor integrations
- HTTP service
- MCP adapter
- shell integrations beyond the core CLI
- TUI browsing

Note: a thin `pi` extension is explicitly allowed as a companion deliverable and does not belong in this deferred bucket if it simply wraps the core CLI.

### Repository sources

- remote git indexing
- cached clones
- multi-repo search

### Retrieval enhancements

- graph navigation beyond lightweight importer lookup
- impact analysis beyond lightweight importer lookup
- rename tracking
- advanced stable ID aliasing
- embeddings or rerankers
- AI-generated summaries

### Expanded UX

- richer human-readable output
- interactive browsing
- team/org workflows
- collaborative cache sharing

## Conditionally eligible in Milestone 3 (`SPEC.md`)

These are not default v1 scope, but may be promoted in Milestone 3 when they stay local, deterministic, and measurably useful for agent retrieval:

- lightweight `find importers` (parser-driven import relationships)
- package/module summaries derived from indexed symbols (non-AI)

## Promotion criteria

A deferred feature should move into the main spec only if it clearly improves the core agent workflow:

1. find likely code
2. inspect compact structure
3. retrieve exact source
4. avoid unnecessary full-file reads
