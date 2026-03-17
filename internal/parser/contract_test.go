package parser

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/pusherofbrooms/codesieve/internal/parser/languages/bash"
	"github.com/pusherofbrooms/codesieve/internal/parser/languages/golang"
	"github.com/pusherofbrooms/codesieve/internal/parser/languages/javascript"
	"github.com/pusherofbrooms/codesieve/internal/parser/languages/python"
	"github.com/pusherofbrooms/codesieve/internal/parser/languages/typescript"
	"github.com/pusherofbrooms/codesieve/internal/parser/languages/yaml"
)

var (
	_ ParseFunc = golang.Parse
	_ ParseFunc = python.Parse
	_ ParseFunc = typescript.Parse
	_ ParseFunc = javascript.Parse
	_ ParseFunc = bash.Parse
	_ ParseFunc = yaml.Parse
)

func TestParseContractPopulatesRequiredSymbolFields(t *testing.T) {
	tests := []struct {
		name string
		path string
	}{
		{name: "go", path: "tests/testdata/languages/go/basic.go"},
		{name: "python", path: "tests/testdata/languages/python/basic.py"},
		{name: "typescript", path: "tests/testdata/languages/typescript/basic.ts"},
		{name: "bash", path: "tests/testdata/languages/bash/basic.sh"},
		{name: "yaml", path: "tests/testdata/languages/yaml/basic.yaml"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			content := mustReadFixture(t, tc.path)
			syms, lang, err := ParseSymbols(tc.path, content)
			if err != nil {
				t.Fatalf("ParseSymbols error: %v", err)
			}
			if lang == "" {
				t.Fatalf("expected language for %s", tc.path)
			}
			if len(syms) == 0 {
				t.Fatalf("expected at least one symbol for %s", tc.path)
			}
			for i, sym := range syms {
				if sym.Name == "" {
					t.Fatalf("symbol[%d] missing Name: %+v", i, sym)
				}
				if sym.Kind == "" {
					t.Fatalf("symbol[%d] missing Kind: %+v", i, sym)
				}
				if sym.StartLine <= 0 || sym.EndLine <= 0 {
					t.Fatalf("symbol[%d] has invalid line range: %+v", i, sym)
				}
				if sym.StartByte < 0 || sym.EndByte < 0 {
					t.Fatalf("symbol[%d] has invalid byte range: %+v", i, sym)
				}
			}
		})
	}
}

func mustReadFixture(t *testing.T, rel string) []byte {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd: %v", err)
	}
	repoRoot := filepath.Clean(filepath.Join(wd, "..", ".."))
	full := filepath.Join(repoRoot, rel)
	content, err := os.ReadFile(full)
	if err != nil {
		t.Fatalf("ReadFile(%s): %v", full, err)
	}
	return content
}
