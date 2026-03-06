package app

import (
	tspython "github.com/jorgensen/codesieve/internal/tslang/python"
	treesitter "github.com/tree-sitter/go-tree-sitter"
)

func parsePythonTreeSitter(content []byte) ([]Symbol, error) {
	return parseWithTreeSitter(content, treesitter.NewLanguage(tspython.Language()), func(root *treesitter.Node) []Symbol {
		var symbols []Symbol
		var walk func(node *treesitter.Node, container string)
		walk = func(node *treesitter.Node, container string) {
			if node == nil {
				return
			}
			switch node.Kind() {
			case "decorated_definition":
				for i := uint(0); i < node.NamedChildCount(); i++ {
					child := node.NamedChild(i)
					if child != nil && child.Kind() != "decorator" {
						walk(child, container)
					}
				}
				return
			case "class_definition":
				nameNode := node.ChildByFieldName("name")
				if nameNode != nil {
					name := nodeText(nameNode, content)
					symbols = append(symbols, makeSymbol(content, node, name, name, "class"))
					walk(node.ChildByFieldName("body"), name)
					return
				}
			case "function_definition", "async_function_definition":
				nameNode := node.ChildByFieldName("name")
				if nameNode != nil {
					name := nodeText(nameNode, content)
					kind := "function"
					qualified := name
					if container != "" {
						kind = "method"
						qualified = container + "." + name
					}
					sym := makeSymbol(content, node, name, qualified, kind)
					sym.Signature = pythonSignature(node, content)
					symbols = append(symbols, sym)
					walk(node.ChildByFieldName("body"), "")
					return
				}
			}
			for i := uint(0); i < node.NamedChildCount(); i++ {
				walk(node.NamedChild(i), container)
			}
		}
		walk(root, "")
		sortSymbols(symbols)
		return symbols
	})
}
