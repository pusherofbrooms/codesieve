package gdscript

import (
	"strings"

	"github.com/pusherofbrooms/codesieve/internal/parser/core"
	"github.com/pusherofbrooms/codesieve/internal/parser/languages/register"
	tsgdscript "github.com/pusherofbrooms/codesieve/internal/tslang/gdscript"
	treesitter "github.com/tree-sitter/go-tree-sitter"
)

const Name = "gdscript"

var Extensions = []string{".gd"}

func init() {
	register.MustRegister(Name, Parse)
}

func Parse(_ string, content []byte) ([]core.Symbol, error) {
	return core.ParseWithTreeSitter(content, treesitter.NewLanguage(tsgdscript.Language()), func(root *treesitter.Node) []core.Symbol {
		var symbols []core.Symbol
		scriptClass := findScriptClass(root, content)

		var walk func(node *treesitter.Node, container string, inFunction bool)
		walk = func(node *treesitter.Node, container string, inFunction bool) {
			if node == nil {
				return
			}

			switch node.Kind() {
			case "class_name_statement":
				name := nodeName(node, content)
				if name == "" {
					return
				}
				symbols = append(symbols, core.MakeSymbol(content, node, name, name, "class"))
				return
			case "class_definition":
				name := nodeName(node, content)
				if name == "" {
					return
				}
				qualified, parent := core.QualifiedNameFromContainer(container, name)
				sym := core.MakeSymbol(content, node, name, qualified, "class")
				sym.ParentID = parent
				symbols = append(symbols, sym)
				walk(node.ChildByFieldName("body"), qualified, false)
				return
			case "function_definition":
				name := nodeName(node, content)
				if name == "" {
					return
				}
				kind := "function"
				if container != "" {
					kind = "method"
				}
				qualified, parent := core.QualifiedNameFromContainer(container, name)
				sym := core.MakeSymbol(content, node, name, qualified, kind)
				sym.ParentID = parent
				symbols = append(symbols, sym)
				return
			case "constructor_definition":
				name := "_init"
				qualified, parent := core.QualifiedNameFromContainer(container, name)
				sym := core.MakeSymbol(content, node, name, qualified, "constructor")
				sym.ParentID = parent
				symbols = append(symbols, sym)
				return
			case "enum_definition":
				name := nodeName(node, content)
				if name == "" {
					name = "enum"
				}
				qualified, parent := core.QualifiedNameFromContainer(container, name)
				sym := core.MakeSymbol(content, node, name, qualified, "enum")
				sym.ParentID = parent
				symbols = append(symbols, sym)
				return
			case "signal_statement":
				name := nodeName(node, content)
				if name == "" {
					return
				}
				qualified, parent := core.QualifiedNameFromContainer(container, name)
				sym := core.MakeSymbol(content, node, name, qualified, "signal")
				sym.ParentID = parent
				symbols = append(symbols, sym)
				return
			case "const_statement", "variable_statement", "export_variable_statement", "onready_variable_statement":
				if inFunction {
					return
				}
				name := nodeName(node, content)
				if name == "" {
					return
				}
				kind := "field"
				if node.Kind() == "const_statement" {
					kind = "const"
				} else if container == "" {
					kind = "var"
				}
				qualified, parent := core.QualifiedNameFromContainer(container, name)
				sym := core.MakeSymbol(content, node, name, qualified, kind)
				sym.ParentID = parent
				symbols = append(symbols, sym)
				return
			}

			if node.Kind() == "body" && container == "" && scriptClass != "" {
				container = scriptClass
			}
			core.WalkNamedChildren(node, func(child *treesitter.Node) {
				walk(child, container, inFunction || node.Kind() == "function_definition" || node.Kind() == "constructor_definition")
			})
		}

		walk(root, scriptClass, false)
		core.SortSymbols(symbols)
		return symbols
	})
}

func findScriptClass(root *treesitter.Node, content []byte) string {
	if root == nil {
		return ""
	}
	for i := uint(0); i < root.NamedChildCount(); i++ {
		child := root.NamedChild(i)
		if child != nil && child.Kind() == "class_name_statement" {
			return nodeName(child, content)
		}
	}
	return ""
}

func nodeName(node *treesitter.Node, content []byte) string {
	if node == nil {
		return ""
	}
	nameNode := node.ChildByFieldName("name")
	if nameNode == nil {
		return ""
	}
	return strings.TrimSpace(core.NodeText(nameNode, content))
}
