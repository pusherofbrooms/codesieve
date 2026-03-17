package parser

import (
	"strings"
	"testing"
)

func TestSupportedLanguagesIncludesBuiltins(t *testing.T) {
	names := SupportedLanguages()
	want := map[string]bool{
		"go":         false,
		"python":     false,
		"rust":       false,
		"typescript": false,
		"javascript": false,
		"java":       false,
		"hcl":        false,
		"json":       false,
		"bash":       false,
		"yaml":       false,
	}

	for _, name := range names {
		if _, ok := want[name]; ok {
			want[name] = true
		}
	}

	for name, found := range want {
		if !found {
			t.Fatalf("missing language %q in SupportedLanguages: %v", name, names)
		}
	}
}

func TestDetectLanguageByExtension(t *testing.T) {
	tests := []struct {
		path string
		want string
	}{
		{path: "main.go", want: "go"},
		{path: "script.py", want: "python"},
		{path: "main.rs", want: "rust"},
		{path: "file.ts", want: "typescript"},
		{path: "file.tsx", want: "typescript"},
		{path: "file.js", want: "javascript"},
		{path: "file.jsx", want: "javascript"},
		{path: "AuthService.java", want: "java"},
		{path: "AuthService.cs", want: "csharp"},
		{path: "Program.csx", want: "csharp"},
		{path: "main.tf", want: "hcl"},
		{path: "inputs.tfvars", want: "hcl"},
		{path: "terragrunt.hcl", want: "hcl"},
		{path: "main.tf.json", want: "hcl"},
		{path: "env.tfvars.json", want: "hcl"},
		{path: "template.json", want: "json"},
		{path: "script.sh", want: "bash"},
		{path: "script.bash", want: "bash"},
		{path: "template.yaml", want: "yaml"},
		{path: "template.yml", want: "yaml"},
		{path: "README.md", want: ""},
	}

	for _, tc := range tests {
		t.Run(tc.path, func(t *testing.T) {
			if got := DetectLanguage(tc.path); got != tc.want {
				t.Fatalf("DetectLanguage(%q) = %q, want %q", tc.path, got, tc.want)
			}
		})
	}
}

func TestDetectLanguageWithContentSupportsBashShebang(t *testing.T) {
	content := []byte("#!/usr/bin/env bash\necho hi\n")
	if got := DetectLanguageWithContent("scripts/deploy", content); got != "bash" {
		t.Fatalf("DetectLanguageWithContent shebang = %q, want bash", got)
	}
	if got := DetectLanguageWithContent("README", []byte("# docs\n")); got != "" {
		t.Fatalf("DetectLanguageWithContent non-shebang = %q, want empty", got)
	}
}

func TestBuildRegistryRejectsDuplicateLanguageNames(t *testing.T) {
	_, err := buildRegistry([]Spec{
		{Name: "go", Extensions: []string{".go"}, Parse: noopParse},
		{Name: "go", Extensions: []string{".gox"}, Parse: noopParse},
	})
	if err == nil || !strings.Contains(err.Error(), "duplicate parser language name") {
		t.Fatalf("expected duplicate language name error, got %v", err)
	}
}

func TestBuildRegistryRejectsDuplicateExtensionsAcrossLanguages(t *testing.T) {
	_, err := buildRegistry([]Spec{
		{Name: "lang-a", Extensions: []string{".x"}, Parse: noopParse},
		{Name: "lang-b", Extensions: []string{".x"}, Parse: noopParse},
	})
	if err == nil || !strings.Contains(err.Error(), "duplicate parser extension") {
		t.Fatalf("expected duplicate extension error, got %v", err)
	}
}

func TestBuildRegistryNormalizesAndDeduplicatesExtensions(t *testing.T) {
	reg, err := buildRegistry([]Spec{{
		Name:       "shell",
		Extensions: []string{"SH", ".sh", " .SH ", ""},
		Parse:      noopParse,
	}})
	if err != nil {
		t.Fatalf("buildRegistry error: %v", err)
	}
	if len(reg.specs) != 1 {
		t.Fatalf("expected 1 spec, got %d", len(reg.specs))
	}
	if reg.specs[0].Version != "1" {
		t.Fatalf("expected default parser version 1, got %q", reg.specs[0].Version)
	}
	ext := reg.specs[0].Extensions
	if len(ext) != 1 || ext[0] != ".sh" {
		t.Fatalf("unexpected normalized extensions: %v", ext)
	}
	if got := reg.byExt[".sh"]; got == nil || got.Name != "shell" {
		t.Fatalf("byExt lookup failed: %+v", got)
	}
}

func TestLanguageVersionReturnsRegisteredVersion(t *testing.T) {
	if got := LanguageVersion("go"); strings.TrimSpace(got) == "" {
		t.Fatalf("expected non-empty language version for go")
	}
	if got := LanguageVersion("does-not-exist"); got != "" {
		t.Fatalf("expected empty version for unknown language, got %q", got)
	}
}

func noopParse(string, []byte) ([]Symbol, error) { return nil, nil }
