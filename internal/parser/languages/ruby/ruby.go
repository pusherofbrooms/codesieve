package ruby

import (
	"strings"

	"github.com/pusherofbrooms/codesieve/internal/parser/core"
	"github.com/pusherofbrooms/codesieve/internal/parser/languages/register"
	tsruby "github.com/pusherofbrooms/codesieve/internal/tslang/ruby"
	treesitter "github.com/tree-sitter/go-tree-sitter"
)

const Name = "ruby"

var Extensions = []string{".rb"}

func init() {
	register.MustRegister(Name, Parse)
}

func Parse(_ string, content []byte) ([]core.Symbol, error) {
	return core.ParseWithTreeSitter(content, treesitter.NewLanguage(tsruby.Language()), func(root *treesitter.Node) []core.Symbol {
		var symbols []core.Symbol

		var walk func(node *treesitter.Node, container string)
		walk = func(node *treesitter.Node, container string) {
			if node == nil {
				return
			}

			switch node.Kind() {
			case "class", "module":
				name := normalizeQualified(core.NodeText(node.ChildByFieldName("name"), content))
				if name == "" {
					return
				}
				qualified, parent := qualifyContainer(container, name)
				kind := "class"
				if node.Kind() == "module" {
					kind = "module"
				}
				sym := core.MakeSymbol(content, node, shortName(qualified), qualified, kind)
				sym.ParentID = parent
				symbols = append(symbols, sym)
				walk(node.ChildByFieldName("body"), qualified)
				return
			case "method", "singleton_method":
				name := strings.TrimSpace(core.NodeText(node.ChildByFieldName("name"), content))
				if name == "" {
					return
				}
				qualified, parent := core.QualifiedNameFromContainer(container, name)
				kind := "function"
				if container != "" {
					kind = "method"
				}
				if name == "initialize" && container != "" {
					kind = "constructor"
				}
				sym := core.MakeSymbol(content, node, name, qualified, kind)
				sym.ParentID = parent
				symbols = append(symbols, sym)
				walk(node.ChildByFieldName("body"), container)
				return
			case "assignment":
				name := normalizeQualified(core.NodeText(node.ChildByFieldName("left"), content))
				if isConstantName(name) {
					qualified, parent := qualifyContainer(container, name)
					sym := core.MakeSymbol(content, node, shortName(qualified), qualified, "constant")
					sym.ParentID = parent
					symbols = append(symbols, sym)
				}
			}

			core.WalkNamedChildren(node, func(child *treesitter.Node) {
				walk(child, container)
			})
		}

		walk(root, "")
		core.SortSymbols(symbols)
		return symbols
	})
}

func normalizeQualified(in string) string {
	in = strings.TrimSpace(in)
	if in == "" {
		return ""
	}
	in = strings.ReplaceAll(in, "::", ".")
	in = strings.Join(strings.Fields(in), "")
	in = strings.TrimPrefix(in, ".")
	return in
}

func qualifyContainer(container, name string) (qualified, parent string) {
	if strings.Contains(name, ".") {
		return name, parentFromQualified(name)
	}
	return core.QualifiedNameFromContainer(container, name)
}

func shortName(qualified string) string {
	if idx := strings.LastIndex(qualified, "."); idx >= 0 {
		return qualified[idx+1:]
	}
	return qualified
}

func parentFromQualified(qualified string) string {
	if idx := strings.LastIndex(qualified, "."); idx > 0 {
		return qualified[:idx]
	}
	return ""
}

func isConstantName(name string) bool {
	if name == "" {
		return false
	}
	parts := strings.Split(name, ".")
	for _, p := range parts {
		if p == "" {
			return false
		}
		r := rune(p[0])
		if r < 'A' || r > 'Z' {
			return false
		}
	}
	return true
}
