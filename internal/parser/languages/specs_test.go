package languages

import (
	"slices"
	"testing"

	catalog "github.com/pusherofbrooms/codesieve/internal/languages"
	"github.com/pusherofbrooms/codesieve/internal/parser/languages/register"
)

func TestRegisteredParsersMatchLanguageCatalog(t *testing.T) {
	got := register.Names()
	slices.Sort(got)

	want := make([]string, 0, len(catalog.All()))
	for _, item := range catalog.All() {
		want = append(want, item.Name)
	}
	slices.Sort(want)

	if !slices.Equal(got, want) {
		t.Fatalf("registered parser set mismatch\n got: %v\nwant: %v", got, want)
	}
}

func TestSpecsMatchLanguageCatalog(t *testing.T) {
	specs := Specs()
	if len(specs) != len(catalog.All()) {
		t.Fatalf("spec count mismatch: got %d want %d", len(specs), len(catalog.All()))
	}

	byName := map[string]struct {
		Version    string
		Extensions []string
	}{}
	for _, item := range catalog.All() {
		byName[item.Name] = struct {
			Version    string
			Extensions []string
		}{Version: item.Version, Extensions: item.Extensions}
	}

	for _, sp := range specs {
		meta, ok := byName[sp.Name]
		if !ok {
			t.Fatalf("spec %q missing from catalog", sp.Name)
		}
		if sp.Version != meta.Version {
			t.Fatalf("version mismatch for %q: spec=%q catalog=%q", sp.Name, sp.Version, meta.Version)
		}
		gotExt := slices.Clone(sp.Extensions)
		wantExt := slices.Clone(meta.Extensions)
		slices.Sort(gotExt)
		slices.Sort(wantExt)
		if !slices.Equal(gotExt, wantExt) {
			t.Fatalf("extension mismatch for %q: spec=%v catalog=%v", sp.Name, gotExt, wantExt)
		}
	}
}
