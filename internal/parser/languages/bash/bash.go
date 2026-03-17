package bash

import (
	"github.com/pusherofbrooms/codesieve/internal/parser/core"
	tsbash "github.com/pusherofbrooms/codesieve/internal/tslang/bash"
	treesitter "github.com/tree-sitter/go-tree-sitter"
)

const Name = "bash"

var Extensions = []string{".sh", ".bash"}

func Parse(_ string, content []byte) ([]core.Symbol, error) {
	return core.ParseWithTreeSitter(content, treesitter.NewLanguage(tsbash.Language()), func(root *treesitter.Node) []core.Symbol {
		var symbols []core.Symbol
		var walk func(node *treesitter.Node)
		walk = func(node *treesitter.Node) {
			if node == nil {
				return
			}
			if node.Kind() == "function_definition" {
				nameNode := node.ChildByFieldName("name")
				if nameNode != nil {
					name := core.NodeText(nameNode, content)
					sym := core.MakeSymbol(content, node, name, name, "function")
					symbols = append(symbols, sym)
				}
			}
			core.WalkNamedChildren(node, walk)
		}
		walk(root)
		core.SortSymbols(symbols)
		return symbols
	})
}
