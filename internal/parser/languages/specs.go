package languages

import (
	"fmt"
	"slices"

	catalog "github.com/pusherofbrooms/codesieve/internal/languages"
	"github.com/pusherofbrooms/codesieve/internal/parser/languages/bash"
	"github.com/pusherofbrooms/codesieve/internal/parser/languages/csharp"
	"github.com/pusherofbrooms/codesieve/internal/parser/languages/golang"
	"github.com/pusherofbrooms/codesieve/internal/parser/languages/hcl"
	"github.com/pusherofbrooms/codesieve/internal/parser/languages/java"
	"github.com/pusherofbrooms/codesieve/internal/parser/languages/javascript"
	"github.com/pusherofbrooms/codesieve/internal/parser/languages/json"
	"github.com/pusherofbrooms/codesieve/internal/parser/languages/python"
	"github.com/pusherofbrooms/codesieve/internal/parser/languages/rust"
	"github.com/pusherofbrooms/codesieve/internal/parser/languages/typescript"
	"github.com/pusherofbrooms/codesieve/internal/parser/languages/yaml"
	"github.com/pusherofbrooms/codesieve/internal/parser/spec"
)

func mustByName(name string) catalog.Metadata {
	meta, ok := catalog.ByName(name)
	if !ok {
		panic(fmt.Sprintf("missing language metadata for %s", name))
	}
	return meta
}

func Specs() []spec.Spec {
	goMeta := mustByName("go")
	pythonMeta := mustByName("python")
	rustMeta := mustByName("rust")
	typeScriptMeta := mustByName("typescript")
	javaScriptMeta := mustByName("javascript")
	javaMeta := mustByName("java")
	csharpMeta := mustByName("csharp")
	hclMeta := mustByName("hcl")
	jsonMeta := mustByName("json")
	bashMeta := mustByName("bash")
	yamlMeta := mustByName("yaml")

	return []spec.Spec{
		{Name: goMeta.Name, Version: goMeta.Version, Extensions: slices.Clone(goMeta.Extensions), Parse: golang.Parse},
		{Name: pythonMeta.Name, Version: pythonMeta.Version, Extensions: slices.Clone(pythonMeta.Extensions), Parse: python.Parse},
		{Name: rustMeta.Name, Version: rustMeta.Version, Extensions: slices.Clone(rustMeta.Extensions), Parse: rust.Parse},
		{Name: typeScriptMeta.Name, Version: typeScriptMeta.Version, Extensions: slices.Clone(typeScriptMeta.Extensions), Parse: typescript.Parse},
		{Name: javaScriptMeta.Name, Version: javaScriptMeta.Version, Extensions: slices.Clone(javaScriptMeta.Extensions), Parse: javascript.Parse},
		{Name: javaMeta.Name, Version: javaMeta.Version, Extensions: slices.Clone(javaMeta.Extensions), Parse: java.Parse},
		{Name: csharpMeta.Name, Version: csharpMeta.Version, Extensions: slices.Clone(csharpMeta.Extensions), Parse: csharp.Parse},
		{Name: hclMeta.Name, Version: hclMeta.Version, Extensions: slices.Clone(hclMeta.Extensions), Parse: hcl.Parse},
		{Name: jsonMeta.Name, Version: jsonMeta.Version, Extensions: slices.Clone(jsonMeta.Extensions), Parse: json.Parse},
		{Name: bashMeta.Name, Version: bashMeta.Version, Extensions: slices.Clone(bashMeta.Extensions), Parse: bash.Parse},
		{Name: yamlMeta.Name, Version: yamlMeta.Version, Extensions: slices.Clone(yamlMeta.Extensions), Parse: yaml.Parse},
	}
}
