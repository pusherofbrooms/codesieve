package languages

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGeneratedSupportedLanguagesDocIsUpToDate(t *testing.T) {
	repoRoot := repoRoot(t)
	path := filepath.Join(repoRoot, "docs", "supported_languages.md")
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%s): %v", path, err)
	}
	want := RenderSupportedLanguagesMarkdown()
	if string(got) != want {
		t.Fatalf("%s is out of date; run: nix develop --command go run ./cmd/gen-language-artifacts", path)
	}
}

func TestGeneratedLanguageMapShellIsUpToDate(t *testing.T) {
	repoRoot := repoRoot(t)
	path := filepath.Join(repoRoot, "scripts", "language-map.sh")
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%s): %v", path, err)
	}
	want := RenderLanguageMapShell()
	if string(got) != want {
		t.Fatalf("%s is out of date; run: nix develop --command go run ./cmd/gen-language-artifacts", path)
	}
}

func TestRenderSupportedLanguagesSummary(t *testing.T) {
	got := RenderSupportedLanguagesSummary()
	want := "go, python, rust, ruby, zig, php, typescript, javascript, java, csharp, hcl, json, bash, yaml"
	if got != want {
		t.Fatalf("RenderSupportedLanguagesSummary() = %q, want %q", got, want)
	}
}

func repoRoot(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd: %v", err)
	}
	root := filepath.Clean(filepath.Join(wd, "..", ".."))
	if _, err := os.Stat(filepath.Join(root, "go.mod")); err != nil {
		t.Fatalf("repo root %s missing go.mod: %v", root, err)
	}
	return root
}
