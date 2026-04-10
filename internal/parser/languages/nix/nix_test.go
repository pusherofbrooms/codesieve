package nix

import "testing"

func TestParseNixFlakeLikeBindings(t *testing.T) {
	src := []byte(`{ pkgs, system, ... }:
let
  lib = pkgs.lib;
in {
  packages.${system}.default = pkgs.hello;
  devShells.${system}.default = pkgs.mkShell {
    buildInputs = [ pkgs.go ];
  };
  overlays.default = final: prev: {
    hello = prev.hello;
  };
  inherit lib;
}
`)
	syms, err := Parse("flake.nix", src)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}

	found := map[string]bool{}
	for _, s := range syms {
		found[s.Kind+":"+s.QualifiedName] = true
	}
	for _, key := range []string{
		"file:file:flake.nix",
		"binding:lib",
		"attrset:packages",
		"attrset:packages.${system}",
		"binding:packages.${system}.default",
		"binding:devShells.${system}.default",
		"binding:devShells.${system}.default.buildInputs",
		"function:overlays.default",
		"parameter:overlays.default.final",
		"parameter:overlays.default.prev",
		"binding:overlays.default.hello",
		"inherit:lib",
	} {
		if !found[key] {
			t.Fatalf("missing expected symbol %q in %+v", key, syms)
		}
	}
}
