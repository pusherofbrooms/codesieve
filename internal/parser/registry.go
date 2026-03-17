package parser

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/pusherofbrooms/codesieve/internal/parser/core"
	"github.com/pusherofbrooms/codesieve/internal/parser/filetype"
	parselanguages "github.com/pusherofbrooms/codesieve/internal/parser/languages"
	"github.com/pusherofbrooms/codesieve/internal/parser/languages/bash"
	"github.com/pusherofbrooms/codesieve/internal/parser/languages/hcl"
	"github.com/pusherofbrooms/codesieve/internal/parser/spec"
)

type Symbol = core.Symbol

type ParseFunc = spec.ParseFunc

type Spec = spec.Spec

type registryData struct {
	specs  []Spec
	byExt  map[string]*Spec
	byName map[string]*Spec
}

var registry = mustBuildRegistry(parselanguages.Specs())

func SupportedLanguages() []string {
	out := make([]string, 0, len(registry.specs))
	for _, spec := range registry.specs {
		out = append(out, spec.Name)
	}
	return out
}

func DetectLanguage(path string) string {
	if filetype.IsTerraformJSONPath(path) {
		return hcl.Name
	}
	spec := specForPath(path)
	if spec == nil {
		return ""
	}
	return spec.Name
}

func DetectLanguageWithContent(path string, content []byte) string {
	if filetype.IsTerraformJSONPath(path) {
		return hcl.Name
	}
	spec := specForPath(path)
	if spec != nil {
		return spec.Name
	}
	if isBashShebang(content) {
		return bash.Name
	}
	return ""
}

func ParseSymbols(path string, content []byte) ([]Symbol, string, error) {
	lang := DetectLanguageWithContent(path, content)
	if lang == "" {
		return nil, "", nil
	}
	spec := specForName(lang)
	if spec == nil {
		return nil, "", nil
	}
	syms, err := spec.Parse(path, content)
	return syms, spec.Name, err
}

func LanguageVersion(name string) string {
	spec := specForName(name)
	if spec == nil {
		return ""
	}
	return spec.Version
}

func specForPath(path string) *Spec {
	ext := strings.ToLower(filepath.Ext(path))
	if ext == "" {
		return nil
	}
	return registry.byExt[ext]
}

func specForName(name string) *Spec {
	if strings.TrimSpace(name) == "" {
		return nil
	}
	return registry.byName[name]
}

func mustBuildRegistry(specs []Spec) registryData {
	reg, err := buildRegistry(specs)
	if err != nil {
		panic(err)
	}
	return reg
}

func buildRegistry(specs []Spec) (registryData, error) {
	reg := registryData{
		specs:  make([]Spec, 0, len(specs)),
		byExt:  make(map[string]*Spec, len(specs)),
		byName: make(map[string]*Spec, len(specs)),
	}
	for i := range specs {
		s := specs[i]
		name := strings.TrimSpace(s.Name)
		if name == "" {
			return registryData{}, fmt.Errorf("parser spec has empty name")
		}
		if s.Parse == nil {
			return registryData{}, fmt.Errorf("parser spec %q has nil Parse func", name)
		}
		if _, exists := reg.byName[name]; exists {
			return registryData{}, fmt.Errorf("duplicate parser language name %q", name)
		}
		s.Name = name
		s.Version = normalizeVersion(s.Version)
		s.Extensions = normalizeExtensions(s.Extensions)
		reg.specs = append(reg.specs, s)
		stored := &reg.specs[len(reg.specs)-1]
		reg.byName[stored.Name] = stored
		for _, ext := range stored.Extensions {
			if prev, exists := reg.byExt[ext]; exists {
				return registryData{}, fmt.Errorf("duplicate parser extension %q for %q and %q", ext, prev.Name, stored.Name)
			}
			reg.byExt[ext] = stored
		}
	}
	return reg, nil
}

func normalizeExtensions(in []string) []string {
	if len(in) == 0 {
		return nil
	}
	out := make([]string, 0, len(in))
	seen := map[string]struct{}{}
	for _, ext := range in {
		ext = strings.ToLower(strings.TrimSpace(ext))
		if ext == "" {
			continue
		}
		if !strings.HasPrefix(ext, ".") {
			ext = "." + ext
		}
		if _, ok := seen[ext]; ok {
			continue
		}
		seen[ext] = struct{}{}
		out = append(out, ext)
	}
	return out
}

func normalizeVersion(v string) string {
	v = strings.TrimSpace(v)
	if v == "" {
		return "1"
	}
	return v
}

func isBashShebang(content []byte) bool {
	line := string(content)
	if idx := strings.IndexByte(line, '\n'); idx >= 0 {
		line = line[:idx]
	}
	line = strings.TrimSpace(line)
	if !strings.HasPrefix(line, "#!") {
		return false
	}
	line = strings.ToLower(line)
	return strings.Contains(line, "bash")
}
