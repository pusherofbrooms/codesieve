package languages

import (
	"slices"

	"github.com/pusherofbrooms/codesieve/internal/parser/languages/bash"
	"github.com/pusherofbrooms/codesieve/internal/parser/languages/golang"
	"github.com/pusherofbrooms/codesieve/internal/parser/languages/hcl"
	"github.com/pusherofbrooms/codesieve/internal/parser/languages/javascript"
	"github.com/pusherofbrooms/codesieve/internal/parser/languages/json"
	"github.com/pusherofbrooms/codesieve/internal/parser/languages/python"
	"github.com/pusherofbrooms/codesieve/internal/parser/languages/typescript"
	"github.com/pusherofbrooms/codesieve/internal/parser/languages/yaml"
	"github.com/pusherofbrooms/codesieve/internal/parser/spec"
)

func Specs() []spec.Spec {
	return []spec.Spec{
		{Name: golang.Name, Extensions: slices.Clone(golang.Extensions), Parse: golang.Parse},
		{Name: python.Name, Extensions: slices.Clone(python.Extensions), Parse: python.Parse},
		{Name: typescript.Name, Extensions: slices.Clone(typescript.Extensions), Parse: typescript.Parse},
		{Name: javascript.Name, Extensions: slices.Clone(javascript.Extensions), Parse: javascript.Parse},
		{Name: hcl.Name, Extensions: slices.Clone(hcl.Extensions), Parse: hcl.Parse},
		{Name: json.Name, Extensions: slices.Clone(json.Extensions), Parse: json.Parse},
		{Name: bash.Name, Extensions: slices.Clone(bash.Extensions), Parse: bash.Parse},
		{Name: yaml.Name, Extensions: slices.Clone(yaml.Extensions), Parse: yaml.Parse},
	}
}
