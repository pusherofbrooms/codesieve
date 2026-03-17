package java

import (
	"strings"

	"github.com/pusherofbrooms/codesieve/internal/parser/core"
	tsjava "github.com/pusherofbrooms/codesieve/internal/tslang/java"
	treesitter "github.com/tree-sitter/go-tree-sitter"
)

const Name = "java"

var Extensions = []string{".java"}

func Parse(_ string, content []byte) ([]core.Symbol, error) {
	return core.ParseWithTreeSitter(content, treesitter.NewLanguage(tsjava.Language()), func(root *treesitter.Node) []core.Symbol {
		var symbols []core.Symbol

		var walk func(node *treesitter.Node, containerQualified string)
		walk = func(node *treesitter.Node, containerQualified string) {
			if node == nil {
				return
			}

			switch node.Kind() {
			case "class_declaration", "interface_declaration", "enum_declaration", "record_declaration", "annotation_type_declaration":
				name := nodeName(node, content)
				if name != "" {
					kind := javaContainerKind(node.Kind())
					qualified, parent := core.QualifiedNameFromContainer(containerQualified, name)
					sym := core.MakeSymbol(content, node, name, qualified, kind)
					sym.ParentID = parent
					symbols = append(symbols, sym)
					walk(node.ChildByFieldName("body"), qualified)
					return
				}
			case "method_declaration":
				name := nodeName(node, content)
				if name != "" {
					qualified, parent := core.QualifiedNameFromContainer(containerQualified, name)
					kind := "function"
					if containerQualified != "" {
						kind = "method"
					}
					sym := core.MakeSymbol(content, node, name, qualified, kind)
					sym.ParentID = parent
					symbols = append(symbols, sym)
				}
				return
			case "constructor_declaration", "compact_constructor_declaration":
				name := nodeName(node, content)
				if name != "" {
					qualified, parent := core.QualifiedNameFromContainer(containerQualified, name)
					sym := core.MakeSymbol(content, node, name, qualified, "constructor")
					sym.ParentID = parent
					symbols = append(symbols, sym)
				}
				return
			case "field_declaration", "constant_declaration":
				fieldKind := "field"
				if node.Kind() == "constant_declaration" {
					fieldKind = "constant"
				}
				for i := uint(0); i < node.NamedChildCount(); i++ {
					child := node.NamedChild(i)
					if child == nil || child.Kind() != "variable_declarator" {
						continue
					}
					name := nodeName(child, content)
					if name == "" {
						continue
					}
					qualified, parent := core.QualifiedNameFromContainer(containerQualified, name)
					sym := core.MakeSymbol(content, child, name, qualified, fieldKind)
					sym.ParentID = parent
					symbols = append(symbols, sym)
				}
				return
			case "enum_constant":
				name := nodeName(node, content)
				if name != "" {
					qualified, parent := core.QualifiedNameFromContainer(containerQualified, name)
					sym := core.MakeSymbol(content, node, name, qualified, "constant")
					sym.ParentID = parent
					symbols = append(symbols, sym)
				}
				return
			}

			for i := uint(0); i < node.NamedChildCount(); i++ {
				walk(node.NamedChild(i), containerQualified)
			}
		}

		walk(root, "")
		core.SortSymbols(symbols)
		return symbols
	})
}

func nodeName(node *treesitter.Node, content []byte) string {
	if node == nil {
		return ""
	}
	if nameNode := node.ChildByFieldName("name"); nameNode != nil {
		return strings.TrimSpace(core.NodeText(nameNode, content))
	}
	if node.Kind() == "enum_constant" && node.NamedChildCount() > 0 {
		first := node.NamedChild(0)
		if first != nil {
			return strings.TrimSpace(core.NodeText(first, content))
		}
	}
	return ""
}

func javaContainerKind(kind string) string {
	switch kind {
	case "class_declaration":
		return "class"
	case "interface_declaration":
		return "interface"
	case "enum_declaration":
		return "enum"
	case "record_declaration":
		return "record"
	case "annotation_type_declaration":
		return "annotation"
	default:
		return "class"
	}
}
