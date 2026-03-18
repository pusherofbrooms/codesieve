package languages

import (
	"fmt"
	"slices"

	catalog "github.com/pusherofbrooms/codesieve/internal/languages"
	"github.com/pusherofbrooms/codesieve/internal/parser/languages/register"
	"github.com/pusherofbrooms/codesieve/internal/parser/spec"
)

func Specs() []spec.Spec {
	metas := catalog.All()
	out := make([]spec.Spec, 0, len(metas))
	for _, meta := range metas {
		parse, ok := register.Lookup(meta.Name)
		if !ok {
			panic(fmt.Sprintf("missing parse function registration for %q", meta.Name))
		}
		out = append(out, spec.Spec{
			Name:       meta.Name,
			Version:    meta.Version,
			Extensions: slices.Clone(meta.Extensions),
			Parse:      parse,
		})
	}
	return out
}
