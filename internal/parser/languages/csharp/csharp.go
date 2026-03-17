package csharp

import (
	"strings"

	"github.com/pusherofbrooms/codesieve/internal/parser/core"
	tscsharp "github.com/pusherofbrooms/codesieve/internal/tslang/csharp"
	treesitter "github.com/tree-sitter/go-tree-sitter"
)

const Name = "csharp"

var Extensions = []string{".cs", ".csx"}

func Parse(_ string, content []byte) ([]core.Symbol, error) {
	return core.ParseWithTreeSitter(content, treesitter.NewLanguage(tscsharp.Language()), func(root *treesitter.Node) []core.Symbol {
		var symbols []core.Symbol
		seenTopLevel := map[string]struct{}{}

		var walk func(node *treesitter.Node, namespaceQualified, typeQualified string)
		walk = func(node *treesitter.Node, namespaceQualified, typeQualified string) {
			if node == nil {
				return
			}

			switch node.Kind() {
			case "namespace_declaration", "file_scoped_namespace_declaration":
				name := normalizeQualifiedName(core.NodeText(node.ChildByFieldName("name"), content))
				if name != "" {
					qualified := namespaceName(namespaceQualified, name)
					sym := core.MakeSymbol(content, node, qualified, qualified, "namespace")
					sym.ParentID = parentFromQualified(qualified)
					appendUnique(&symbols, seenTopLevel, sym)
					namespaceQualified = qualified
				}
				if body := node.ChildByFieldName("body"); body != nil {
					walk(body, namespaceQualified, typeQualified)
					return
				}
			case "using_directive":
				name := usingDirectiveName(node, content)
				if name != "" {
					sym := core.MakeSymbol(content, node, name, name, "import")
					appendUnique(&symbols, seenTopLevel, sym)
				}
				return
			case "class_declaration", "interface_declaration", "struct_declaration", "record_declaration", "enum_declaration", "delegate_declaration":
				name := nodeName(node, content)
				if name == "" {
					return
				}
				qualified, parent := core.QualifiedNameFromContainer(typeQualified, name)
				sym := core.MakeSymbol(content, node, name, qualified, typeKind(node.Kind()))
				sym.ParentID = parent
				symbols = append(symbols, sym)
				if node.Kind() == "enum_declaration" {
					appendEnumMembers(&symbols, node.ChildByFieldName("body"), content, qualified)
				}
				if body := node.ChildByFieldName("body"); body != nil {
					walk(body, namespaceQualified, qualified)
				}
				return
			case "method_declaration", "local_function_statement":
				name := nodeName(node, content)
				if name != "" {
					suffix := methodOverloadSuffix(node, content)
					qualifiedBase, parent := core.QualifiedNameFromContainer(typeQualified, name)
					kind := "function"
					if typeQualified != "" {
						kind = "method"
					}
					sym := core.MakeSymbol(content, node, name, qualifiedBase+suffix, kind)
					sym.ParentID = parent
					symbols = append(symbols, sym)
				}
				return
			case "constructor_declaration":
				name := nodeName(node, content)
				if name != "" {
					suffix := methodOverloadSuffix(node, content)
					qualifiedBase, parent := core.QualifiedNameFromContainer(typeQualified, name)
					sym := core.MakeSymbol(content, node, name, qualifiedBase+suffix, "constructor")
					sym.ParentID = parent
					symbols = append(symbols, sym)
				}
				return
			case "property_declaration":
				appendNamedMemberSymbol(&symbols, node, content, typeQualified, "property")
				return
			case "event_declaration":
				appendNamedMemberSymbol(&symbols, node, content, typeQualified, "event")
				return
			case "field_declaration":
				appendVariableDeclarators(&symbols, node, content, typeQualified, "field")
				return
			case "event_field_declaration":
				appendVariableDeclarators(&symbols, node, content, typeQualified, "event")
				return
			case "operator_declaration":
				appendOperatorSymbol(&symbols, node, content, typeQualified)
				return
			case "conversion_operator_declaration":
				appendConversionOperatorSymbol(&symbols, node, content, typeQualified)
				return
			case "indexer_declaration":
				name := "this[]"
				qualified, parent := core.QualifiedNameFromContainer(typeQualified, name)
				sym := core.MakeSymbol(content, node, name, qualified, "indexer")
				sym.ParentID = parent
				symbols = append(symbols, sym)
				return
			}

			core.WalkNamedChildren(node, func(child *treesitter.Node) {
				walk(child, namespaceQualified, typeQualified)
			})
		}

		walk(root, "", "")
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

func appendNamedMemberSymbol(symbols *[]core.Symbol, node *treesitter.Node, content []byte, container, kind string) {
	name := nodeName(node, content)
	if name == "" {
		return
	}
	qualified, parent := core.QualifiedNameFromContainer(container, name)
	sym := core.MakeSymbol(content, node, name, qualified, kind)
	sym.ParentID = parent
	*symbols = append(*symbols, sym)
}

func appendVariableDeclarators(symbols *[]core.Symbol, node *treesitter.Node, content []byte, container, kind string) {
	for i := uint(0); i < node.NamedChildCount(); i++ {
		child := node.NamedChild(i)
		if child == nil || child.Kind() != "variable_declaration" {
			continue
		}
		for j := uint(0); j < child.NamedChildCount(); j++ {
			decl := child.NamedChild(j)
			if decl == nil || decl.Kind() != "variable_declarator" {
				continue
			}
			name := nodeName(decl, content)
			if name == "" {
				continue
			}
			qualified, parent := core.QualifiedNameFromContainer(container, name)
			sym := core.MakeSymbol(content, decl, name, qualified, kind)
			sym.ParentID = parent
			*symbols = append(*symbols, sym)
		}
	}
}

func appendEnumMembers(symbols *[]core.Symbol, body *treesitter.Node, content []byte, enumQualified string) {
	if body == nil {
		return
	}
	core.WalkNamedChildren(body, func(child *treesitter.Node) {
		if child == nil || child.Kind() != "enum_member_declaration" {
			return
		}
		name := nodeName(child, content)
		if name == "" {
			return
		}
		sym := core.MakeSymbol(content, child, name, enumQualified+"."+name, "constant")
		sym.ParentID = enumQualified
		*symbols = append(*symbols, sym)
	})
}

func appendOperatorSymbol(symbols *[]core.Symbol, node *treesitter.Node, content []byte, container string) {
	op := strings.TrimSpace(core.NodeText(node.ChildByFieldName("operator"), content))
	if op == "" {
		op = "operator"
	}
	name := "operator " + op
	qualified, parent := core.QualifiedNameFromContainer(container, name+methodOverloadSuffix(node, content))
	sym := core.MakeSymbol(content, node, name, qualified, "operator")
	sym.ParentID = parent
	*symbols = append(*symbols, sym)
}

func appendConversionOperatorSymbol(symbols *[]core.Symbol, node *treesitter.Node, content []byte, container string) {
	toType := strings.TrimSpace(core.NodeText(node.ChildByFieldName("type"), content))
	if toType == "" {
		toType = "<unknown>"
	}
	sig := core.SignatureFromNode(node, content)
	prefix := "explicit"
	if strings.Contains(sig, " implicit operator ") {
		prefix = "implicit"
	}
	name := prefix + " operator " + strings.Join(strings.Fields(toType), " ")
	qualified, parent := core.QualifiedNameFromContainer(container, name+methodOverloadSuffix(node, content))
	sym := core.MakeSymbol(content, node, name, qualified, "operator")
	sym.ParentID = parent
	*symbols = append(*symbols, sym)
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
		t = strings.Join(strings.Fields(t), " ")
		types = append(types, t)
	}
	return "(" + strings.Join(types, ",") + ")"
}

func nodeName(node *treesitter.Node, content []byte) string {
	if node == nil {
		return ""
	}
	if nameNode := node.ChildByFieldName("name"); nameNode != nil {
		return strings.TrimSpace(core.NodeText(nameNode, content))
	}
	return ""
}

func usingDirectiveName(node *treesitter.Node, content []byte) string {
	name := normalizeQualifiedName(core.NodeText(node.ChildByFieldName("name"), content))
	if name != "" {
		return name
	}
	text := strings.TrimSpace(core.NodeText(node, content))
	text = strings.TrimSuffix(text, ";")
	text = strings.TrimSpace(strings.TrimPrefix(text, "global"))
	text = strings.TrimSpace(strings.TrimPrefix(text, "using"))
	text = strings.TrimSpace(strings.TrimPrefix(text, "static"))
	if idx := strings.Index(text, "="); idx >= 0 {
		text = strings.TrimSpace(text[idx+1:])
	}
	return normalizeQualifiedName(text)
}

func normalizeQualifiedName(in string) string {
	in = strings.TrimSpace(in)
	if in == "" {
		return ""
	}
	in = strings.ReplaceAll(in, "::", ".")
	in = strings.Join(strings.Fields(in), "")
	return in
}

func namespaceName(container, name string) string {
	if container == "" {
		return name
	}
	if strings.Contains(name, ".") {
		return name
	}
	return container + "." + name
}

func parentFromQualified(qualified string) string {
	idx := strings.LastIndex(qualified, ".")
	if idx <= 0 {
		return ""
	}
	return qualified[:idx]
}

func typeKind(kind string) string {
	switch kind {
	case "class_declaration":
		return "class"
	case "interface_declaration":
		return "interface"
	case "struct_declaration":
		return "struct"
	case "record_declaration":
		return "record"
	case "enum_declaration":
		return "enum"
	case "delegate_declaration":
		return "delegate"
	default:
		return "type"
	}
}
