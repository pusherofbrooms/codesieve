package zig

import (
	"strings"

	"github.com/pusherofbrooms/codesieve/internal/parser/core"
	"github.com/pusherofbrooms/codesieve/internal/parser/languages/register"
	tszig "github.com/pusherofbrooms/codesieve/internal/tslang/zig"
	treesitter "github.com/tree-sitter/go-tree-sitter"
)

const Name = "zig"

var Extensions = []string{".zig"}

func init() {
	register.MustRegister(Name, Parse)
}

func Parse(_ string, content []byte) ([]core.Symbol, error) {
	return core.ParseWithTreeSitter(content, treesitter.NewLanguage(tszig.Language()), func(root *treesitter.Node) []core.Symbol {
		var symbols []core.Symbol

		var walk func(node *treesitter.Node, containerQualified, containerKind string)
		walk = func(node *treesitter.Node, containerQualified, containerKind string) {
			if node == nil {
				return
			}

			switch node.Kind() {
			case "source_file", "struct_declaration", "union_declaration", "enum_declaration", "opaque_declaration", "error_set_declaration":
				core.WalkNamedChildren(node, func(child *treesitter.Node) {
					walk(child, containerQualified, containerKindForNode(node, containerKind))
				})
				return
			case "function_declaration":
				name := nodeName(node, content)
				if name == "" {
					return
				}
				kind := "function"
				if containerQualified != "" {
					kind = "method"
				}
				qualified, parent := core.QualifiedNameFromContainer(containerQualified, name)
				sym := core.MakeSymbol(content, node, name, qualified, kind)
				sym.ParentID = parent
				symbols = append(symbols, sym)
				return
			case "variable_declaration":
				name := variableName(node, content)
				if name == "" || name == "_" {
					return
				}
				if initializer := containerInitializer(node); initializer != nil {
					kind := typeKind(initializer.Kind())
					qualified, parent := core.QualifiedNameFromContainer(containerQualified, name)
					sym := core.MakeSymbol(content, node, name, qualified, kind)
					sym.ParentID = parent
					symbols = append(symbols, sym)
					walk(initializer, qualified, kind)
					return
				}

				kind := "variable"
				if isConstDeclaration(node, content) {
					kind = "constant"
				}
				qualified, parent := core.QualifiedNameFromContainer(containerQualified, name)
				sym := core.MakeSymbol(content, node, name, qualified, kind)
				sym.ParentID = parent
				symbols = append(symbols, sym)
				return
			case "container_field":
				if containerQualified == "" {
					return
				}
				name := nodeName(node, content)
				if name == "" {
					return
				}
				kind := "field"
				if containerKind == "enum" || containerKind == "error" {
					kind = "variant"
				}
				qualified, parent := core.QualifiedNameFromContainer(containerQualified, name)
				sym := core.MakeSymbol(content, node, name, qualified, kind)
				sym.ParentID = parent
				symbols = append(symbols, sym)
				return
			case "using_namespace_declaration":
				target := strings.TrimSpace(core.NodeText(node.NamedChild(0), content))
				if target == "" {
					return
				}
				sym := core.MakeSymbol(content, node, target, target, "import")
				symbols = append(symbols, sym)
				return
			case "test_declaration":
				name := testName(node, content)
				if name == "" {
					return
				}
				qualified, parent := core.QualifiedNameFromContainer(containerQualified, name)
				sym := core.MakeSymbol(content, node, name, qualified, "test")
				sym.ParentID = parent
				symbols = append(symbols, sym)
				return
			}

			core.WalkNamedChildren(node, func(child *treesitter.Node) {
				walk(child, containerQualified, containerKind)
			})
		}

		walk(root, "", "")
		core.SortSymbols(symbols)
		return symbols
	})
}

func nodeName(node *treesitter.Node, content []byte) string {
	if node == nil {
		return ""
	}
	if nameNode := node.ChildByFieldName("name"); nameNode != nil {
		return sanitizeName(core.NodeText(nameNode, content))
	}
	if node.Kind() == "variable_declaration" {
		return variableName(node, content)
	}
	if node.Kind() == "test_declaration" {
		return testName(node, content)
	}
	return ""
}

func variableName(node *treesitter.Node, content []byte) string {
	for i := uint(0); i < node.NamedChildCount(); i++ {
		child := node.NamedChild(i)
		if child != nil && child.Kind() == "identifier" {
			name := sanitizeName(core.NodeText(child, content))
			if name != "" {
				return name
			}
		}
	}
	return ""
}

func testName(node *treesitter.Node, content []byte) string {
	for i := uint(0); i < node.NamedChildCount(); i++ {
		child := node.NamedChild(i)
		if child == nil {
			continue
		}
		switch child.Kind() {
		case "string", "identifier":
			name := sanitizeName(core.NodeText(child, content))
			if name != "" {
				return "test." + name
			}
		}
	}
	return ""
}

func sanitizeName(name string) string {
	name = strings.TrimSpace(name)
	name = strings.Trim(name, `"'`)
	name = strings.Join(strings.Fields(name), " ")
	return name
}

func isConstDeclaration(node *treesitter.Node, content []byte) bool {
	text := strings.TrimSpace(core.NodeText(node, content))
	if text == "" {
		return false
	}
	head := text
	if idx := strings.Index(head, "="); idx >= 0 {
		head = head[:idx]
	}
	head = " " + strings.Join(strings.Fields(head), " ") + " "
	return strings.Contains(head, " const ")
}

func containerInitializer(node *treesitter.Node) *treesitter.Node {
	for i := uint(0); i < node.NamedChildCount(); i++ {
		child := node.NamedChild(i)
		if child == nil {
			continue
		}
		switch child.Kind() {
		case "struct_declaration", "enum_declaration", "union_declaration", "opaque_declaration", "error_set_declaration":
			return child
		}
	}
	return nil
}

func typeKind(kind string) string {
	switch kind {
	case "struct_declaration":
		return "struct"
	case "enum_declaration":
		return "enum"
	case "union_declaration":
		return "union"
	case "opaque_declaration":
		return "opaque"
	case "error_set_declaration":
		return "error"
	default:
		return "type"
	}
}

func containerKindForNode(node *treesitter.Node, current string) string {
	if node == nil {
		return current
	}
	switch node.Kind() {
	case "struct_declaration":
		return "struct"
	case "union_declaration":
		return "union"
	case "enum_declaration":
		return "enum"
	case "opaque_declaration":
		return "opaque"
	case "error_set_declaration":
		return "error"
	default:
		return current
	}
}
