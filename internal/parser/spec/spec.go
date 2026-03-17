package spec

import "github.com/pusherofbrooms/codesieve/internal/parser/core"

type Symbol = core.Symbol

type ParseFunc func(path string, content []byte) ([]Symbol, error)

type Spec struct {
	Name       string
	Version    string
	Extensions []string
	Parse      ParseFunc
}
