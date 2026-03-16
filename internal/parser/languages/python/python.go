package python

import (
	"github.com/pusherofbrooms/codesieve/internal/parser/core"
	tspython "github.com/pusherofbrooms/codesieve/internal/tslang/python"
	treesitter "github.com/tree-sitter/go-tree-sitter"
)

const Name = "python"

var Extensions = []string{".py"}

func Parse(_ string, content []byte) ([]core.Symbol, error) {
	return core.ParseWithTreeSitter(content, treesitter.NewLanguage(tspython.Language()), func(root *treesitter.Node) []core.Symbol {
		var symbols []core.Symbol
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
					name := core.NodeText(nameNode, content)
					symbols = append(symbols, core.MakeSymbol(content, node, name, name, "class"))
					walk(node.ChildByFieldName("body"), name)
					return
				}
			case "function_definition", "async_function_definition":
				nameNode := node.ChildByFieldName("name")
				if nameNode != nil {
					name := core.NodeText(nameNode, content)
					kind := "function"
					qualified := name
					parent := ""
					if container != "" {
						kind = "method"
						qualified = container + "." + name
						parent = container
					}
					sym := core.MakeSymbol(content, node, name, qualified, kind)
					sym.ParentID = parent
					sym.Signature = core.PythonSignature(node, content)
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
		core.SortSymbols(symbols)
		return symbols
	})
}
