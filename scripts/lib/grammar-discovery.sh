#!/usr/bin/env bash

# Shared helpers for grammar discovery scripts.

codesieve_grammar_discovery_require_tools() {
  local missing=()
  command -v curl >/dev/null 2>&1 || missing+=(curl)
  command -v jq >/dev/null 2>&1 || missing+=(jq)
  if [ "${#missing[@]}" -gt 0 ]; then
    echo "error: missing required tool(s): ${missing[*]}" >&2
    echo "hint: run scripts via 'nix develop --command ...'" >&2
    return 1
  fi
}

codesieve_grammar_discovery_orgs() {
  printf '%s\n' "tree-sitter" "tree-sitter-grammars"
}

codesieve_grammar_discovery_collect_json() {
  local page response count org
  local -a rows=()

  while IFS= read -r org; do
    page=1
    while true; do
      response="$(curl -fsSL "https://api.github.com/orgs/${org}/repos?type=public&per_page=100&page=${page}")" || return 1
      if [ "$(printf '%s' "$response" | jq -r 'type')" != "array" ]; then
        echo "error: unexpected GitHub API response for org '${org}'" >&2
        printf '%s\n' "$response" | jq -r '.message? // "unknown error"' >&2 || true
        return 1
      fi

      count="$(printf '%s' "$response" | jq 'length')"
      if [ "$count" -eq 0 ]; then
        break
      fi

      while IFS= read -r row; do
        rows+=("$row")
      done < <(
        printf '%s' "$response" | jq -c --arg org "$org" '
          .[]
          | select(.name | startswith("tree-sitter-"))
          | {
              org: $org,
              name,
              full_name,
              html_url,
              description: (.description // ""),
              pushed_at,
              stargazers_count,
              archived,
              fork,
              license: (.license.spdx_id // "")
            }
        '
      )

      if [ "$count" -lt 100 ]; then
        break
      fi
      page=$((page + 1))
    done
  done < <(codesieve_grammar_discovery_orgs)

  if [ "${#rows[@]}" -eq 0 ]; then
    printf '[]\n'
    return 0
  fi

  printf '%s\n' "${rows[@]}" | jq -s '
    unique_by(.html_url)
    | map(select(.archived | not))
    | map(select(.fork | not))
  '
}
