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
		seenTopLevel := map[string]struct{}{}

		var walk func(node *treesitter.Node, containerQualified string)
		walk = func(node *treesitter.Node, containerQualified string) {
			if node == nil {
				return
			}

			switch node.Kind() {
			case "package_declaration":
				if containerQualified == "" {
					qualified := declarationPath(node, content, "package")
					if qualified != "" {
						appendUnique(&symbols, seenTopLevel, core.MakeSymbol(content, node, qualified, qualified, "package"))
					}
				}
				return
			case "import_declaration":
				if containerQualified == "" {
					qualified := declarationPath(node, content, "import")
					if qualified != "" {
						appendUnique(&symbols, seenTopLevel, core.MakeSymbol(content, node, qualified, qualified, "import"))
					}
				}
				return
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
					suffix := methodOverloadSuffix(node, content)
					baseQualified, parent := core.QualifiedNameFromContainer(containerQualified, name)
					qualified := baseQualified + suffix
					kind := "function"
					if containerQualified != "" {
						kind = "method"
					}
					sym := core.MakeSymbol(content, node, name, qualified, kind)
					sym.ParentID = parent
					sym.Signature = methodSignature(node, content)
					symbols = append(symbols, sym)
				}
				return
			case "constructor_declaration", "compact_constructor_declaration":
				name := nodeName(node, content)
				if name != "" {
					suffix := methodOverloadSuffix(node, content)
					baseQualified, parent := core.QualifiedNameFromContainer(containerQualified, name)
					qualified := baseQualified + suffix
					sym := core.MakeSymbol(content, node, name, qualified, "constructor")
					sym.ParentID = parent
					sym.Signature = constructorSignature(node, content)
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

func appendUnique(symbols *[]core.Symbol, seen map[string]struct{}, sym core.Symbol) {
	if sym.Name == "" || sym.Kind == "" {
		return
	}
	key := sym.Kind + ":" + sym.QualifiedName
	if _, ok := seen[key]; ok {
		return
	}
	seen[key] = struct{}{}
	*symbols = append(*symbols, sym)
}

func declarationPath(node *treesitter.Node, content []byte, keyword string) string {
	text := strings.TrimSpace(core.NodeText(node, content))
	text = strings.TrimPrefix(text, keyword)
	text = strings.TrimSpace(text)
	text = strings.TrimSuffix(text, ";")
	text = strings.TrimSpace(text)
	return text
}

func methodSignature(node *treesitter.Node, content []byte) string {
	name := nodeName(node, content)
	if name == "" {
		return core.SignatureFromNode(node, content)
	}
	params := parameterListText(node, content)
	ret := strings.TrimSpace(core.NodeText(node.ChildByFieldName("type"), content))
	typeParams := strings.TrimSpace(core.NodeText(node.ChildByFieldName("type_parameters"), content))
	throws := namedChildTextByKind(node, content, "throws")

	parts := make([]string, 0, 4)
	if ret != "" {
		parts = append(parts, ret)
	}
	call := name
	if typeParams != "" {
		call = typeParams + " " + call
	}
	call += params
	parts = append(parts, call)
	if throws != "" {
		parts = append(parts, throws)
	}
	return strings.Join(parts, " ")
}

func constructorSignature(node *treesitter.Node, content []byte) string {
	name := nodeName(node, content)
	if name == "" {
		return core.SignatureFromNode(node, content)
	}
	params := parameterListText(node, content)
	typeParams := strings.TrimSpace(core.NodeText(node.ChildByFieldName("type_parameters"), content))
	throws := namedChildTextByKind(node, content, "throws")
	call := name
	if typeParams != "" {
		call = typeParams + " " + call
	}
	call += params
	if throws != "" {
		call += " " + throws
	}
	return call
}

func methodOverloadSuffix(node *treesitter.Node, content []byte) string {
	params := node.ChildByFieldName("parameters")
	if params == nil {
		return "()"
	}
	types := make([]string, 0, params.NamedChildCount())
	for i := uint(0); i < params.NamedChildCount(); i++ {
		param := params.NamedChild(i)
		if param == nil {
			continue
		}
		t := strings.TrimSpace(core.NodeText(param.ChildByFieldName("type"), content))
		if t == "" {
			t = strings.TrimSpace(core.NodeText(param, content))
		}
		if param.Kind() == "spread_parameter" && !strings.HasSuffix(t, "...") {
			t += "..."
		}
		t = strings.Join(strings.Fields(t), " ")
		types = append(types, t)
	}
	return "(" + strings.Join(types, ",") + ")"
}

func parameterListText(node *treesitter.Node, content []byte) string {
	params := strings.TrimSpace(core.NodeText(node.ChildByFieldName("parameters"), content))
	if params == "" {
		return "()"
	}
	return params
}

func namedChildTextByKind(node *treesitter.Node, content []byte, kind string) string {
	for i := uint(0); i < node.NamedChildCount(); i++ {
		child := node.NamedChild(i)
		if child != nil && child.Kind() == kind {
			return strings.TrimSpace(core.NodeText(child, content))
		}
	}
	return ""
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
