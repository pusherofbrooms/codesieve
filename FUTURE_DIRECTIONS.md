# Future Directions

This file lists ideas intentionally deferred from `SPEC.md`.

They are not part of the v1 scope unless they become necessary to achieve the core goal of reducing agent token use.

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

## Promoted into conditional scope in `SPEC.md`

These are no longer fully deferred. They are allowed in Milestone 3 when they remain local, deterministic, and measurably useful for agent retrieval:

- lightweight `find importers` (parser-driven import relationships)
- package/module summaries derived from indexed symbols (non-AI)

## Rule for promoting an item into scope

A deferred feature should only move into the main spec if it clearly improves the core agent workflow:

1. find likely code
2. inspect compact structure
3. retrieve exact source
4. avoid unnecessary full-file reads
