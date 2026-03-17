package languages

import (
	"slices"

	"github.com/pusherofbrooms/codesieve/internal/parser/languages/bash"
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

func Specs() []spec.Spec {
	return []spec.Spec{
		{Name: golang.Name, Version: "1", Extensions: slices.Clone(golang.Extensions), Parse: golang.Parse},
		{Name: python.Name, Version: "1", Extensions: slices.Clone(python.Extensions), Parse: python.Parse},
		{Name: rust.Name, Version: "1", Extensions: slices.Clone(rust.Extensions), Parse: rust.Parse},
		{Name: typescript.Name, Version: "1", Extensions: slices.Clone(typescript.Extensions), Parse: typescript.Parse},
		{Name: javascript.Name, Version: "1", Extensions: slices.Clone(javascript.Extensions), Parse: javascript.Parse},
		{Name: java.Name, Version: "1", Extensions: slices.Clone(java.Extensions), Parse: java.Parse},
		{Name: hcl.Name, Version: "1", Extensions: slices.Clone(hcl.Extensions), Parse: hcl.Parse},
		{Name: json.Name, Version: "1", Extensions: slices.Clone(json.Extensions), Parse: json.Parse},
		{Name: bash.Name, Version: "1", Extensions: slices.Clone(bash.Extensions), Parse: bash.Parse},
		{Name: yaml.Name, Version: "1", Extensions: slices.Clone(yaml.Extensions), Parse: yaml.Parse},
	}
}
