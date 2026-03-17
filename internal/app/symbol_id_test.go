package app

import (
	"testing"

	"github.com/pusherofbrooms/codesieve/internal/parser"
)

func TestSymbolIDUsesByteRangeToAvoidSameLineCollisions(t *testing.T) {
	repoPath := "/tmp/repo"
	relPath := "src/minified.js"

	a := Symbol{
		Name:          "foo",
		QualifiedName: "A.foo",
		Kind:          "method",
		StartLine:     1,
		StartByte:     10,
		EndByte:       25,
	}
	b := Symbol{
		Name:          "foo",
		QualifiedName: "A.foo",
		Kind:          "method",
		StartLine:     1,
		StartByte:     30,
		EndByte:       45,
	}

	idA := symbolID(repoPath, relPath, a)
	idB := symbolID(repoPath, relPath, b)
	if idA == idB {
		t.Fatalf("expected distinct ids for distinct byte ranges, got %q", idA)
	}
}

func TestParseJavaScriptDuplicateMethodsOnOneLineIndexAsDistinctSymbols(t *testing.T) {
	src := []byte(`class A { foo(){} foo(){} }`)
	parsed, lang, err := parser.ParseSymbols("dup.js", src)
	if err != nil {
		t.Fatalf("ParseSymbols error: %v", err)
	}
	if lang != "javascript" {
		t.Fatalf("lang = %q", lang)
	}
	if len(parsed) != 3 {
		t.Fatalf("expected 3 symbols, got %d", len(parsed))
	}

	seen := map[string]struct{}{}
	for _, sym := range parsed {
		id := symbolID("/tmp/repo", "dup.js", sym)
		if _, ok := seen[id]; ok {
			t.Fatalf("duplicate id generated for symbol %+v: %s", sym, id)
		}
		seen[id] = struct{}{}
	}
}
