package tsjs

import (
	"github.com/pusherofbrooms/codesieve/internal/parser/core"
	treesitter "github.com/tree-sitter/go-tree-sitter"
)

func Parse(content []byte, language *treesitter.Language) ([]core.Symbol, error) {
	return core.ParseWithTreeSitter(content, language, func(root *treesitter.Node) []core.Symbol {
		var symbols []core.Symbol
		var walk func(node *treesitter.Node, className string)
		walk = func(node *treesitter.Node, className string) {
			if node == nil {
				return
			}
			switch node.Kind() {
			case "export_statement", "statement_block", "program", "class_body":
				core.WalkNamedChildren(node, func(child *treesitter.Node) {
					walk(child, className)
				})
				return
			case "class_declaration":
				nameNode := node.ChildByFieldName("name")
				if nameNode != nil {
					name := core.NodeText(nameNode, content)
					symbols = append(symbols, core.MakeSymbol(content, node, name, name, "class"))
					walk(node.ChildByFieldName("body"), name)
					return
				}
			case "function_declaration", "generator_function_declaration":
				nameNode := node.ChildByFieldName("name")
				if nameNode != nil {
					name := core.NodeText(nameNode, content)
					sym := core.MakeSymbol(content, node, name, name, "function")
					sym.Signature = core.SignatureFromNode(node, content)
					symbols = append(symbols, sym)
				}
				return
			case "method_definition":
				nameNode := node.ChildByFieldName("name")
				if nameNode != nil {
					name := core.NodeText(nameNode, content)
					qualified, parent := core.QualifiedNameFromContainer(className, name)
					sym := core.MakeSymbol(content, node, name, qualified, "method")
					sym.ParentID = parent
					sym.Signature = core.SignatureFromNode(node, content)
					symbols = append(symbols, sym)
				}
				return
			case "interface_declaration":
				core.AppendNamedNode(&symbols, node, content, "name", "interface")
				return
			case "type_alias_declaration":
				core.AppendNamedNode(&symbols, node, content, "name", "type")
				return
			case "enum_declaration":
				core.AppendNamedNode(&symbols, node, content, "name", "enum")
				return
			case "lexical_declaration", "variable_declaration":
				core.WalkNamedChildren(node, func(child *treesitter.Node) {
					if child == nil || child.Kind() != "variable_declarator" {
						return
					}
					nameNode := child.ChildByFieldName("name")
					valueNode := child.ChildByFieldName("value")
					if nameNode == nil || valueNode == nil || !core.IsFunctionLikeValueNode(valueNode) {
						return
					}
					name := core.NodeText(nameNode, content)
					sym := core.MakeSymbol(content, child, name, name, "function")
					sym.Signature = core.SignatureFromNode(child, content)
					symbols = append(symbols, sym)
				})
				return
			}
			core.WalkNamedChildren(node, func(child *treesitter.Node) {
				walk(child, className)
			})
		}
		walk(root, "")
		core.SortSymbols(symbols)
		return symbols
	})
}
