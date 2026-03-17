package parser

import (
	"path/filepath"
	"slices"
	"strings"

	"github.com/pusherofbrooms/codesieve/internal/parser/core"
	"github.com/pusherofbrooms/codesieve/internal/parser/languages/bash"
	"github.com/pusherofbrooms/codesieve/internal/parser/languages/golang"
	"github.com/pusherofbrooms/codesieve/internal/parser/languages/javascript"
	"github.com/pusherofbrooms/codesieve/internal/parser/languages/python"
	"github.com/pusherofbrooms/codesieve/internal/parser/languages/typescript"
)

type Symbol = core.Symbol

type ParseFunc func(path string, content []byte) ([]Symbol, error)

type Spec struct {
	Name       string
	Extensions []string
	Parse      ParseFunc
}

var specs = []Spec{
	{Name: golang.Name, Extensions: slices.Clone(golang.Extensions), Parse: golang.Parse},
	{Name: python.Name, Extensions: slices.Clone(python.Extensions), Parse: python.Parse},
	{Name: typescript.Name, Extensions: slices.Clone(typescript.Extensions), Parse: typescript.Parse},
	{Name: javascript.Name, Extensions: slices.Clone(javascript.Extensions), Parse: javascript.Parse},
	{Name: bash.Name, Extensions: slices.Clone(bash.Extensions), Parse: bash.Parse},
}

func SupportedLanguages() []string {
	out := make([]string, 0, len(specs))
	for _, spec := range specs {
		out = append(out, spec.Name)
	}
	return out
}

func DetectLanguage(path string) string {
	spec := specForPath(path)
	if spec == nil {
		return ""
	}
	return spec.Name
}

func ParseSymbols(path string, content []byte) ([]Symbol, string, error) {
	spec := specForPath(path)
	if spec == nil {
		return nil, "", nil
	}
	syms, err := spec.Parse(path, content)
	return syms, spec.Name, err
}

func specForPath(path string) *Spec {
	ext := strings.ToLower(filepath.Ext(path))
	for i := range specs {
		spec := &specs[i]
		for _, candidate := range spec.Extensions {
			if ext == candidate {
				return spec
			}
		}
	}
	return nil
}
