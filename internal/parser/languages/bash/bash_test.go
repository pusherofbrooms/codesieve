package bash

import "testing"

func TestParseBashCapturesTopLevelSignalsAndSkipsNoise(t *testing.T) {
	src := []byte(`#!/usr/bin/env bash

API_TOKEN="x"
project_name="dev"
source ./lib/common.sh

login() {
  local user="$1"
  RETRY_COUNT=3
}
`)
	syms, err := Parse("script.sh", src)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}

	found := map[string]bool{}
	for _, s := range syms {
		found[s.Kind+":"+s.Name] = true
	}
	for _, key := range []string{"script:script:script.sh", "variable:API_TOKEN", "include:source:./lib/common.sh", "function:login"} {
		if !found[key] {
			t.Fatalf("missing expected symbol %q in %+v", key, syms)
		}
	}
	if found["variable:project_name"] || found["variable:RETRY_COUNT"] || found["variable:user"] {
		t.Fatalf("unexpected noisy symbols in %+v", syms)
	}
}
