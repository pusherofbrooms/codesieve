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

codesieve_grammar_discovery_repo_contents_json() {
  local full_name="$1"
  local path="${2:-}"
  local url="https://api.github.com/repos/${full_name}/contents"
  if [ -n "$path" ]; then
    url+="/${path}"
  fi

  local tmp http
  tmp="$(mktemp)"
  http="$(curl -sS -o "$tmp" -w '%{http_code}' "$url")" || {
    rm -f "$tmp"
    return 1
  }

  case "$http" in
    200)
      cat "$tmp"
      rm -f "$tmp"
      return 0
      ;;
    404)
      rm -f "$tmp"
      printf '[]\n'
      return 0
      ;;
    *)
      echo "error: GitHub API ${http} for ${url}" >&2
      cat "$tmp" >&2 || true
      rm -f "$tmp"
      return 1
      ;;
  esac
}

codesieve_grammar_discovery_repo_viability_json() {
  local full_name="$1"
  local root src node_types_meta node_types_json
  local has_tree_sitter_json has_license_file has_parser_c has_scanner_c has_node_types_json
  local symbol_hint_count

  root="$(codesieve_grammar_discovery_repo_contents_json "$full_name" "")" || return 1
  src="$(codesieve_grammar_discovery_repo_contents_json "$full_name" "src")" || return 1

  has_tree_sitter_json="$(printf '%s' "$root" | jq 'if type=="array" then any(.[]; .name=="tree-sitter.json") else false end')"
  has_license_file="$(printf '%s' "$root" | jq 'if type=="array" then any(.[]; (.name | test("^LICENSE"; "i"))) else false end')"
  has_parser_c="$(printf '%s' "$src" | jq 'if type=="array" then any(.[]; .name=="parser.c") else false end')"
  has_scanner_c="$(printf '%s' "$src" | jq 'if type=="array" then any(.[]; .name=="scanner.c") else false end')"
  has_node_types_json="$(printf '%s' "$src" | jq 'if type=="array" then any(.[]; .name=="node-types.json") else false end')"

  symbol_hint_count=0
  if [ "$has_node_types_json" = "true" ]; then
    node_types_meta="$(codesieve_grammar_discovery_repo_contents_json "$full_name" "src/node-types.json")" || return 1
    if [ "$(printf '%s' "$node_types_meta" | jq -r 'type')" = "object" ]; then
      node_types_json="$(printf '%s' "$node_types_meta" | jq -r '.download_url // empty' | xargs -I{} curl -fsSL "{}" 2>/dev/null || true)"
      if [ -n "$node_types_json" ]; then
        symbol_hint_count="$(printf '%s' "$node_types_json" | jq '
          if type=="array" then
            [
              .[]?.type?
              | ascii_downcase
              | select(test("function|method|class|struct|interface|enum|union|trait|module|namespace|variable|const|type"))
            ]
            | unique
            | length
          else
            0
          end
        ' 2>/dev/null || echo 0)"
      fi
    fi
  fi

  jq -cn \
    --argjson has_tree_sitter_json "$has_tree_sitter_json" \
    --argjson has_license_file "$has_license_file" \
    --argjson has_parser_c "$has_parser_c" \
    --argjson has_scanner_c "$has_scanner_c" \
    --argjson has_node_types_json "$has_node_types_json" \
    --argjson symbol_hint_count "$symbol_hint_count" '
      {
        has_tree_sitter_json: $has_tree_sitter_json,
        has_license_file: $has_license_file,
        has_parser_c: $has_parser_c,
        has_scanner_c: $has_scanner_c,
        has_node_types_json: $has_node_types_json,
        symbol_hint_count: $symbol_hint_count,
        verdict: (
          if ($has_tree_sitter_json and $has_parser_c and $has_license_file and ($symbol_hint_count > 0)) then "likely-viable"
          elif ($has_tree_sitter_json and $has_parser_c and $has_license_file) then "viable-basic"
          elif ($has_tree_sitter_json and $has_parser_c) then "missing-license"
          else "incomplete"
          end
        )
      }
    '
}
