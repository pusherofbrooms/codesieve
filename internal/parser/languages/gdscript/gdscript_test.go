package gdscript

import "testing"

func TestParseGDScriptExtractsScriptClassMembersAndInnerClass(t *testing.T) {
	src := []byte(`class_name PlayerController
extends Node

signal logged_in(user)

const MAX_RETRIES := 3
@export var endpoint: String = "https://example.invalid"
var token := ""

func login(user: String) -> bool:
	var local_only := user
	return local_only != ""

class Session:
	var active := false

	func close() -> void:
		active = false
`)

	syms, err := Parse("player_controller.gd", src)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}

	found := map[string]bool{}
	for _, s := range syms {
		found[s.Kind+":"+s.QualifiedName] = true
	}
	for _, key := range []string{
		"class:PlayerController",
		"signal:PlayerController.logged_in",
		"const:PlayerController.MAX_RETRIES",
		"field:PlayerController.endpoint",
		"field:PlayerController.token",
		"method:PlayerController.login",
		"class:PlayerController.Session",
		"field:PlayerController.Session.active",
		"method:PlayerController.Session.close",
	} {
		if !found[key] {
			t.Fatalf("missing expected symbol %q in %+v", key, syms)
		}
	}
	if found["field:PlayerController.local_only"] || found["var:local_only"] {
		t.Fatalf("unexpected local variable symbol in %+v", syms)
	}
}
