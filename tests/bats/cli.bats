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
