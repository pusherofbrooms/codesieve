package app

import (
	"path/filepath"
	"strings"
)

type languageSpec struct {
	name       string
	extensions []string
	parse      func(path string, content []byte) ([]Symbol, error)
}

var languageSpecs = []languageSpec{
	{
		name:       "go",
		extensions: []string{".go"},
		parse: func(path string, content []byte) ([]Symbol, error) {
			return parseGo(path, content)
		},
	},
	{
		name:       "python",
		extensions: []string{".py"},
		parse: func(_ string, content []byte) ([]Symbol, error) {
			return parsePythonTreeSitter(content)
		},
	},
	{
		name:       "typescript",
		extensions: []string{".ts", ".tsx"},
		parse:      parseTypeScriptSymbols,
	},
	{
		name:       "javascript",
		extensions: []string{".js", ".jsx", ".mjs", ".cjs"},
		parse:      parseJavaScriptSymbols,
	},
}

func DetectLanguage(path string) string {
	spec := languageSpecForPath(path)
	if spec == nil {
		return ""
	}
	return spec.name
}

func ParseSymbols(path string, content []byte) ([]Symbol, string, error) {
	spec := languageSpecForPath(path)
	if spec == nil {
		return nil, "", nil
	}
	syms, err := spec.parse(path, content)
	return syms, spec.name, err
}

func languageSpecForPath(path string) *languageSpec {
	ext := strings.ToLower(filepath.Ext(path))
	for i := range languageSpecs {
		spec := &languageSpecs[i]
		for _, candidate := range spec.extensions {
			if ext == candidate {
				return spec
			}
		}
	}
	return nil
}
