#!/usr/bin/env bats

setup_file() {
  export PROJECT_ROOT="$(cd "$BATS_TEST_DIRNAME/../.." && pwd)"
  if [ -n "${CODESIEVE_BIN:-}" ]; then
    export TEST_BIN="$CODESIEVE_BIN"
  else
    export TEST_BIN="$PROJECT_ROOT/.bats-codesieve-bin"
    (cd "$PROJECT_ROOT" && go build -o "$TEST_BIN" ./cmd/codesieve)
  fi
}

setup() {
  export FIXTURE="$PROJECT_ROOT/tests/testdata/basicrepo"
  export DB_PATH="$BATS_TEST_TMPDIR/test-$$.db"
}

@test "index returns stable json and respects gitignore" {
  run env CODESIEVE_DB_PATH="$DB_PATH" "$TEST_BIN" index "$FIXTURE" --json
  [ "$status" -eq 0 ]

  echo "$output" | jq -e '.ok == true' >/dev/null
  echo "$output" | jq -e '.data.files_indexed == 3' >/dev/null
  echo "$output" | jq -e '.data.symbols_extracted >= 6' >/dev/null

  pushd "$FIXTURE" >/dev/null
  run env CODESIEVE_DB_PATH="$DB_PATH" "$TEST_BIN" search symbol Hidden --json
  popd >/dev/null

  [ "$status" -eq 0 ]
  echo "$output" | jq -e '.data.results | length == 0' >/dev/null
}

@test "search symbol finds likely symbols" {
  env CODESIEVE_DB_PATH="$DB_PATH" "$TEST_BIN" index "$FIXTURE" --json >/dev/null

  pushd "$FIXTURE" >/dev/null
  run env CODESIEVE_DB_PATH="$DB_PATH" "$TEST_BIN" search symbol Login --kind=function --json
  popd >/dev/null

  [ "$status" -eq 0 ]
  echo "$output" | jq -e '.ok == true' >/dev/null
  echo "$output" | jq -e '.data.results | length >= 1' >/dev/null
  echo "$output" | jq -e '.data.results[] | select(.name == "Login" and .file_path == "src/auth.go")' >/dev/null
}

@test "outline and show symbol work together" {
  env CODESIEVE_DB_PATH="$DB_PATH" "$TEST_BIN" index "$FIXTURE" --json >/dev/null

  pushd "$FIXTURE" >/dev/null
  run env CODESIEVE_DB_PATH="$DB_PATH" "$TEST_BIN" outline src/auth.go --json
  [ "$status" -eq 0 ]
  symbol_id="$(echo "$output" | jq -r '.data.symbols[] | select(.name == "Login") | .id')"
  [ -n "$symbol_id" ]

  run env CODESIEVE_DB_PATH="$DB_PATH" "$TEST_BIN" show symbol "$symbol_id" --json
  popd >/dev/null

  [ "$status" -eq 0 ]
  echo "$output" | jq -e '.ok == true' >/dev/null
  echo "$output" | jq -e '.data.name == "Login"' >/dev/null
  echo "$output" | jq -e '.data.file_path == "src/auth.go"' >/dev/null
  echo "$output" | jq -e '.data.content | contains("func Login(user string) error")' >/dev/null
}

@test "search text and show file return narrow content" {
  env CODESIEVE_DB_PATH="$DB_PATH" "$TEST_BIN" index "$FIXTURE" --json >/dev/null

  pushd "$FIXTURE" >/dev/null
  run env CODESIEVE_DB_PATH="$DB_PATH" "$TEST_BIN" search text AUTH_HEADER --json
  [ "$status" -eq 0 ]
  echo "$output" | jq -e '.data.results[] | select(.file_path == "src/client.ts")' >/dev/null

  run env CODESIEVE_DB_PATH="$DB_PATH" "$TEST_BIN" show file src/client.ts --start-line=7 --end-line=11 --json
  popd >/dev/null

  [ "$status" -eq 0 ]
  echo "$output" | jq -e '.data.file_path == "src/client.ts"' >/dev/null
  echo "$output" | jq -e '.data.start_line == 7 and .data.end_line == 11' >/dev/null
  echo "$output" | jq -e '.data.content | contains("AUTH_HEADER")' >/dev/null
}

@test "search text supports regex/context and uses indexed content" {
  worktree="$BATS_TEST_TMPDIR/text-worktree"
  cp -R "$FIXTURE" "$worktree"

  env CODESIEVE_DB_PATH="$DB_PATH" "$TEST_BIN" index "$worktree" --json >/dev/null

  pushd "$worktree" >/dev/null
  run env CODESIEVE_DB_PATH="$DB_PATH" "$TEST_BIN" search text 'return\s+id;' --regex --context-lines=1 --json
  [ "$status" -eq 0 ]
  echo "$output" | jq -e '.ok == true' >/dev/null
  echo "$output" | jq -e '.data.results[] | select(.file_path == "src/client.ts" and (.context_before | length) >= 1)' >/dev/null

  # Mutate file after indexing; results should still come from indexed content.
  cat > src/client.ts <<'EOF'
export class Client {
  login(token: string) {
    return token;
  }
}
EOF

  run env CODESIEVE_DB_PATH="$DB_PATH" "$TEST_BIN" search text AUTH_HEADER --json
  popd >/dev/null

  [ "$status" -eq 0 ]
  echo "$output" | jq -e '.data.results[] | select(.file_path == "src/client.ts")' >/dev/null
}

@test "reindex skips unchanged files" {
  run env CODESIEVE_DB_PATH="$DB_PATH" "$TEST_BIN" index "$FIXTURE" --json
  [ "$status" -eq 0 ]
  echo "$output" | jq -e '.data.files_indexed == 3 and .data.files_updated == 3 and .data.files_unchanged == 0' >/dev/null

  run env CODESIEVE_DB_PATH="$DB_PATH" "$TEST_BIN" index "$FIXTURE" --json
  [ "$status" -eq 0 ]
  echo "$output" | jq -e '.data.files_indexed == 0 and .data.files_updated == 0 and .data.files_unchanged == 3 and .data.files_deleted == 0' >/dev/null
}

@test "reindex removes deleted files from the index" {
  worktree="$BATS_TEST_TMPDIR/worktree"
  cp -R "$FIXTURE" "$worktree"

  run env CODESIEVE_DB_PATH="$DB_PATH" "$TEST_BIN" index "$worktree" --json
  [ "$status" -eq 0 ]

  rm -f "$worktree/src/helpers.py"

  run env CODESIEVE_DB_PATH="$DB_PATH" "$TEST_BIN" index "$worktree" --json
  [ "$status" -eq 0 ]
  echo "$output" | jq -e '.data.files_deleted == 1' >/dev/null

  pushd "$worktree" >/dev/null
  run env CODESIEVE_DB_PATH="$DB_PATH" "$TEST_BIN" search symbol auth_helper --json
  popd >/dev/null

  [ "$status" -eq 0 ]
  echo "$output" | jq -e '.data.results | length == 0' >/dev/null
}

@test "index handles duplicate one-line javascript methods" {
  worktree="$BATS_TEST_TMPDIR/dup-js"
  mkdir -p "$worktree"
  cat > "$worktree/dup.js" <<'EOF'
class A { foo(){} foo(){} }
EOF

  run env CODESIEVE_DB_PATH="$DB_PATH" "$TEST_BIN" index "$worktree" --json
  [ "$status" -eq 0 ]
  echo "$output" | jq -e '.ok == true' >/dev/null
  echo "$output" | jq -e '.data.files_indexed == 1' >/dev/null
  echo "$output" | jq -e '.data.symbols_extracted == 3' >/dev/null

  pushd "$worktree" >/dev/null
  run env CODESIEVE_DB_PATH="$DB_PATH" "$TEST_BIN" search symbol foo --json
  popd >/dev/null

  [ "$status" -eq 0 ]
  echo "$output" | jq -e '[.data.results[] | select(.qualified_name == "A.foo")] | length == 2' >/dev/null
}

@test "index skips secret paths without indexing their symbols" {
  worktree="$BATS_TEST_TMPDIR/secret-skip"
  cp -R "$FIXTURE" "$worktree"

  cat > "$worktree/src/secrets.py" <<'EOF'
def leaked_key():
    return "AKIA_SHOULD_NOT_BE_INDEXED"
EOF

  cat > "$worktree/.env" <<'EOF'
API_KEY=AKIA_ENV_SHOULD_NOT_BE_INDEXED
EOF

  run env CODESIEVE_DB_PATH="$DB_PATH" "$TEST_BIN" index "$worktree" --json
  [ "$status" -eq 0 ]
  echo "$output" | jq -e '.ok == true' >/dev/null
  echo "$output" | jq -e '[.data.files_skipped[] | select(.code == "SKIPPED_SECRET")] | length >= 2' >/dev/null

  pushd "$worktree" >/dev/null
  run env CODESIEVE_DB_PATH="$DB_PATH" "$TEST_BIN" search symbol leaked_key --json
  popd >/dev/null

  [ "$status" -eq 0 ]
  echo "$output" | jq -e '.data.results | length == 0' >/dev/null
}

@test "repo outline returns repository summary" {
  env CODESIEVE_DB_PATH="$DB_PATH" "$TEST_BIN" index "$FIXTURE" --json >/dev/null

  pushd "$FIXTURE" >/dev/null
  run env CODESIEVE_DB_PATH="$DB_PATH" "$TEST_BIN" repo outline --json
  popd >/dev/null

  [ "$status" -eq 0 ]
  echo "$output" | jq -e '.ok == true' >/dev/null
  echo "$output" | jq -e '.data.repo_path | length > 0' >/dev/null
  echo "$output" | jq -e '.data.total_files == 3' >/dev/null
  echo "$output" | jq -e '.data.total_symbols >= 6' >/dev/null
  echo "$output" | jq -e '.data.language_breakdown.go == 1' >/dev/null
  echo "$output" | jq -e '.data.top_level_directory_counts.src == 3' >/dev/null
  echo "$output" | jq -e '.data.symbol_kind_counts | length > 0' >/dev/null
  echo "$output" | jq -e '.data.index_age_seconds >= 0' >/dev/null
  echo "$output" | jq -e '.data.stale == false' >/dev/null
}

@test "top-level help aliases exit successfully" {
  run "$TEST_BIN" --help
  [ "$status" -eq 0 ]
  [[ "$output" == *"codesieve <command>"* ]]

  run "$TEST_BIN" -h
  [ "$status" -eq 0 ]
  [[ "$output" == *"codesieve <command>"* ]]

  run "$TEST_BIN" help
  [ "$status" -eq 0 ]
  [[ "$output" == *"codesieve <command>"* ]]
}

@test "subcommand help exits successfully" {
  run "$TEST_BIN" search --help
  [ "$status" -eq 0 ]
  [[ "$output" == *"Usage: codesieve search <symbol|text> <query> [flags]"* ]]

  run "$TEST_BIN" show symbol --help
  [ "$status" -eq 0 ]
  [[ "$output" == *"Usage: codesieve show symbol <id> [flags]"* ]]
}
