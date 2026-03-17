package parser

import "testing"

func TestSupportedLanguagesIncludesBuiltins(t *testing.T) {
	names := SupportedLanguages()
	want := map[string]bool{
		"go":         false,
		"python":     false,
		"typescript": false,
		"javascript": false,
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
		{path: "file.ts", want: "typescript"},
		{path: "file.tsx", want: "typescript"},
		{path: "file.js", want: "javascript"},
		{path: "file.jsx", want: "javascript"},
		{path: "main.tf", want: "hcl"},
		{path: "inputs.tfvars", want: "hcl"},
		{path: "terragrunt.hcl", want: "hcl"},
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
