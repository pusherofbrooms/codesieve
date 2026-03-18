package php

import (
	"strings"

	"github.com/pusherofbrooms/codesieve/internal/parser/core"
	"github.com/pusherofbrooms/codesieve/internal/parser/languages/register"
	tsphp "github.com/pusherofbrooms/codesieve/internal/tslang/php"
	treesitter "github.com/tree-sitter/go-tree-sitter"
)

const Name = "php"

var Extensions = []string{".php"}

func init() {
	register.MustRegister(Name, Parse)
}

func Parse(_ string, content []byte) ([]core.Symbol, error) {
	return core.ParseWithTreeSitter(content, treesitter.NewLanguage(tsphp.Language()), func(root *treesitter.Node) []core.Symbol {
		var symbols []core.Symbol

		var walk func(node *treesitter.Node, namespaceQualified, containerQualified string)
		walk = func(node *treesitter.Node, namespaceQualified, containerQualified string) {
			if node == nil {
				return
			}

			switch node.Kind() {
			case "program", "declaration_list", "compound_statement":
				currentNamespace := namespaceQualified
				for i := uint(0); i < node.NamedChildCount(); i++ {
					child := node.NamedChild(i)
					if child == nil {
						continue
					}
					if child.Kind() == "namespace_definition" && child.ChildByFieldName("body") == nil {
						qualified := appendNamespaceSymbol(&symbols, child, content)
						if qualified != "" {
							currentNamespace = qualified
						}
						continue
					}
					walk(child, currentNamespace, containerQualified)
				}
				return
			case "namespace_definition":
				namespaceQualified = appendNamespaceSymbol(&symbols, node, content)
				if body := node.ChildByFieldName("body"); body != nil {
					walk(body, namespaceQualified, "")
					return
				}
			case "class_declaration", "interface_declaration", "trait_declaration", "enum_declaration":
				name := nodeName(node, content)
				if name == "" {
					return
				}
				qualified, parent := core.QualifiedNameFromContainer(namespaceQualified, name)
				sym := core.MakeSymbol(content, node, name, qualified, containerKind(node.Kind()))
				sym.ParentID = parent
				symbols = append(symbols, sym)
				if body := node.ChildByFieldName("body"); body != nil {
					walk(body, namespaceQualified, qualified)
				}
				return
			case "function_definition":
				name := nodeName(node, content)
				if name == "" {
					return
				}
				container := namespaceQualified
				kind := "function"
				if containerQualified != "" {
					container = containerQualified
					kind = "method"
				}
				qualified, parent := core.QualifiedNameFromContainer(container, name)
				sym := core.MakeSymbol(content, node, name, qualified, kind)
				sym.ParentID = parent
				symbols = append(symbols, sym)
				return
			case "method_declaration":
				name := nodeName(node, content)
				if name == "" {
					return
				}
				qualified, parent := core.QualifiedNameFromContainer(containerQualified, name)
				kind := "method"
				if strings.EqualFold(name, "__construct") {
					kind = "constructor"
				}
				sym := core.MakeSymbol(content, node, name, qualified, kind)
				sym.ParentID = parent
				symbols = append(symbols, sym)
				return
			case "const_declaration":
				appendConstDeclaration(&symbols, node, content, activeContainer(namespaceQualified, containerQualified))
				return
			case "property_declaration":
				appendPropertyDeclaration(&symbols, node, content, containerQualified)
				return
			case "enum_case":
				name := nodeName(node, content)
				if name == "" {
					return
				}
				qualified, parent := core.QualifiedNameFromContainer(containerQualified, name)
				sym := core.MakeSymbol(content, node, name, qualified, "constant")
				sym.ParentID = parent
				symbols = append(symbols, sym)
				return
			case "namespace_use_declaration", "use_declaration":
				appendUseDeclarations(&symbols, node, content)
				return
			}

			core.WalkNamedChildren(node, func(child *treesitter.Node) {
				walk(child, namespaceQualified, containerQualified)
			})
		}

		walk(root, "", "")
		core.SortSymbols(symbols)
		return symbols
	})
}

func appendNamespaceSymbol(symbols *[]core.Symbol, node *treesitter.Node, content []byte) string {
	qualified := normalizeQualified(core.NodeText(node.ChildByFieldName("name"), content))
	if qualified == "" {
		return ""
	}
	sym := core.MakeSymbol(content, node, qualified, qualified, "namespace")
	sym.ParentID = parentFromQualified(qualified)
	*symbols = append(*symbols, sym)
	return qualified
}

func appendConstDeclaration(symbols *[]core.Symbol, node *treesitter.Node, content []byte, container string) {
	core.WalkNamedChildren(node, func(child *treesitter.Node) {
		if child == nil || child.Kind() != "const_element" {
			return
		}
		name := nodeName(child, content)
		if name == "" {
			return
		}
		qualified, parent := core.QualifiedNameFromContainer(container, name)
		sym := core.MakeSymbol(content, child, name, qualified, "constant")
		sym.ParentID = parent
		*symbols = append(*symbols, sym)
	})
}

func appendPropertyDeclaration(symbols *[]core.Symbol, node *treesitter.Node, content []byte, container string) {
	core.WalkNamedChildren(node, func(child *treesitter.Node) {
		if child == nil || child.Kind() != "property_element" {
			return
		}
		name := normalizeVariableName(core.NodeText(child.ChildByFieldName("name"), content))
		if name == "" {
			return
		}
		qualified, parent := core.QualifiedNameFromContainer(container, name)
		sym := core.MakeSymbol(content, child, name, qualified, "property")
		sym.ParentID = parent
		*symbols = append(*symbols, sym)
	})
}

func appendUseDeclarations(symbols *[]core.Symbol, node *treesitter.Node, content []byte) {
	groupPrefix := ""
	if node.Kind() == "namespace_use_declaration" && node.ChildByFieldName("body") != nil {
		core.WalkNamedChildren(node, func(child *treesitter.Node) {
			if child != nil && child.Kind() == "namespace_name" && groupPrefix == "" {
				groupPrefix = normalizeQualified(core.NodeText(child, content))
			}
		})
	}

	var walk func(n *treesitter.Node)
	walk = func(n *treesitter.Node) {
		if n == nil {
			return
		}
		if n.Kind() == "namespace_use_clause" {
			path := normalizeQualified(useClausePath(n, content))
			if path != "" {
				if groupPrefix != "" && !strings.Contains(path, ".") {
					path = groupPrefix + "." + path
				}
				sym := core.MakeSymbol(content, n, path, path, "import")
				*symbols = append(*symbols, sym)
			}
			return
		}
		core.WalkNamedChildren(n, walk)
	}
	walk(node)
}

func useClausePath(node *treesitter.Node, content []byte) string {
	for i := uint(0); i < node.NamedChildCount(); i++ {
		child := node.NamedChild(i)
		if child == nil {
			continue
		}
		if child.Kind() == "name" || child.Kind() == "qualified_name" {
			return core.NodeText(child, content)
		}
	}
	return ""
}

func activeContainer(namespaceQualified, containerQualified string) string {
	if containerQualified != "" {
		return containerQualified
	}
	return namespaceQualified
}

func nodeName(node *treesitter.Node, content []byte) string {
	if node == nil {
		return ""
	}
	if nameNode := node.ChildByFieldName("name"); nameNode != nil {
		return normalizeVariableName(strings.TrimSpace(core.NodeText(nameNode, content)))
	}
	for i := uint(0); i < node.NamedChildCount(); i++ {
		child := node.NamedChild(i)
		if child != nil && child.Kind() == "name" {
			return strings.TrimSpace(core.NodeText(child, content))
		}
	}
	return ""
}

func normalizeVariableName(in string) string {
	in = strings.TrimSpace(in)
	in = strings.TrimPrefix(in, "$")
	return in
}

func normalizeQualified(in string) string {
	in = strings.TrimSpace(in)
	if in == "" {
		return ""
	}
	in = strings.TrimPrefix(in, "\\")
	in = strings.ReplaceAll(in, "\\", ".")
	in = strings.ReplaceAll(in, "::", ".")
	in = strings.Join(strings.Fields(in), "")
	return in
}

func parentFromQualified(qualified string) string {
	idx := strings.LastIndex(qualified, ".")
	if idx <= 0 {
		return ""
	}
	return qualified[:idx]
}

func containerKind(kind string) string {
	switch kind {
	case "class_declaration":
		return "class"
	case "interface_declaration":
		return "interface"
	case "trait_declaration":
		return "trait"
	case "enum_declaration":
		return "enum"
	default:
		return "class"
	}
}
