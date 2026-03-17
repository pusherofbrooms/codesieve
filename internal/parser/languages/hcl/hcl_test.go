package hcl

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/pusherofbrooms/codesieve/internal/parser/core"
)

func TestParseNestedRepeatedBlocksHaveStableUniqueQualifiedNames(t *testing.T) {
	content := mustReadFixture(t, "tests/testdata/languages/hcl/basic.tf")
	syms, err := Parse("basic.tf", content)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}

	ingressQualified := map[string]struct{}{}
	for _, sym := range syms {
		if sym.Name == "ingress" && sym.Kind == "block" {
			ingressQualified[sym.QualifiedName] = struct{}{}
		}
	}
	if len(ingressQualified) != 2 {
		t.Fatalf("expected two uniquely-qualified ingress blocks, got %d: %+v", len(ingressQualified), ingressQualified)
	}

	foundSecondIngressFromPort := false
	for _, sym := range syms {
		if sym.Name == "from_port" && sym.Kind == "argument" && sym.QualifiedName == "resource.aws_security_group.web.ingress[2].from_port" {
			foundSecondIngressFromPort = true
			break
		}
	}
	if !foundSecondIngressFromPort {
		t.Fatalf("expected from_port argument under second ingress block")
	}
}

func TestParseTerraformJSONExtractsTopLevelTerraformSymbols(t *testing.T) {
	content := mustReadFixture(t, "tests/testdata/languages/hcl_cases/main.tf.json")
	syms, err := Parse("main.tf.json", content)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}

	want := map[string]string{
		"resource.aws_s3_bucket.app": "resource",
		"variable.aws_region":        "variable",
		"output.bucket_name":         "output",
	}
	for qualified, kind := range want {
		if !hasSymbol(syms, qualified, kind) {
			t.Fatalf("missing symbol %s/%s", qualified, kind)
		}
	}
}

func hasSymbol(symbols []core.Symbol, qualified, kind string) bool {
	for _, sym := range symbols {
		if sym.QualifiedName == qualified && sym.Kind == kind {
			return true
		}
	}
	return false
}

func mustReadFixture(t *testing.T, rel string) []byte {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd: %v", err)
	}
	repoRoot := filepath.Clean(filepath.Join(wd, "..", "..", "..", ".."))
	full := filepath.Join(repoRoot, rel)
	content, err := os.ReadFile(full)
	if err != nil {
		t.Fatalf("ReadFile(%s): %v", full, err)
	}
	return content
}
