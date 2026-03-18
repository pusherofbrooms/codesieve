package languages

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGeneratedTSLangBindingsAreUpToDate(t *testing.T) {
	repoRoot := repoRoot(t)
	files := RenderTSLangBindings()
	for _, rel := range SortedTSBindingPaths(files) {
		path := filepath.Join(repoRoot, rel)
		got, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("ReadFile(%s): %v", path, err)
		}
		want := files[rel]
		if string(got) != want {
			t.Fatalf("%s is out of date; run: nix develop --command go run ./cmd/gen-tslang-bindings", path)
		}
	}
}
