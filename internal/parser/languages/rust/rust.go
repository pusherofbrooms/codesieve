package rust

import (
	"strings"

	"github.com/pusherofbrooms/codesieve/internal/parser/core"
	"github.com/pusherofbrooms/codesieve/internal/parser/languages/register"
	tsrust "github.com/pusherofbrooms/codesieve/internal/tslang/rust"
	treesitter "github.com/tree-sitter/go-tree-sitter"
)

const Name = "rust"

var Extensions = []string{".rs"}

func init() {
	register.MustRegister(Name, Parse)
}

func Parse(_ string, content []byte) ([]core.Symbol, error) {
	return core.ParseWithTreeSitter(content, treesitter.NewLanguage(tsrust.Language()), func(root *treesitter.Node) []core.Symbol {
		var symbols []core.Symbol

		var walk func(node *treesitter.Node, moduleQualified, implContainer, traitContainer string)
		walk = func(node *treesitter.Node, moduleQualified, implContainer, traitContainer string) {
			if node == nil {
				return
			}

			switch node.Kind() {
			case "source_file", "declaration_list":
				core.WalkNamedChildren(node, func(child *treesitter.Node) {
					walk(child, moduleQualified, implContainer, traitContainer)
				})
				return
			case "mod_item":
				name := nodeName(node, content)
				if name != "" {
					qualified, parent := core.QualifiedNameFromContainer(moduleQualified, name)
					sym := core.MakeSymbol(content, node, name, qualified, "module")
					sym.ParentID = parent
					symbols = append(symbols, sym)
					walk(node.ChildByFieldName("body"), qualified, "", "")
					return
				}
			case "struct_item":
				appendContainerSymbol(&symbols, node, content, moduleQualified, "struct")
				return
			case "enum_item":
				name := nodeName(node, content)
				if name == "" {
					return
				}
				enumQualified, parent := core.QualifiedNameFromContainer(moduleQualified, name)
				enumSym := core.MakeSymbol(content, node, name, enumQualified, "enum")
				enumSym.ParentID = parent
				symbols = append(symbols, enumSym)

				body := node.ChildByFieldName("body")
				if body != nil {
					core.WalkNamedChildren(body, func(child *treesitter.Node) {
						if child == nil || child.Kind() != "enum_variant" {
							return
						}
						variantName := nodeName(child, content)
						if variantName == "" {
							return
						}
						variant := core.MakeSymbol(content, child, variantName, enumQualified+"."+variantName, "variant")
						variant.ParentID = enumQualified
						symbols = append(symbols, variant)
					})
				}
				return
			case "trait_item":
				name := nodeName(node, content)
				if name == "" {
					return
				}
				qualified, parent := core.QualifiedNameFromContainer(moduleQualified, name)
				sym := core.MakeSymbol(content, node, name, qualified, "trait")
				sym.ParentID = parent
				symbols = append(symbols, sym)
				walk(node.ChildByFieldName("body"), moduleQualified, "", qualified)
				return
			case "impl_item":
				implFor := normalizeTypeName(core.NodeText(node.ChildByFieldName("type"), content))
				if implFor == "" {
					walk(node.ChildByFieldName("body"), moduleQualified, "", "")
					return
				}
				if moduleQualified != "" && !strings.Contains(implFor, ".") {
					implFor = moduleQualified + "." + implFor
				}
				walk(node.ChildByFieldName("body"), moduleQualified, implFor, "")
				return
			case "function_item", "function_signature_item":
				name := nodeName(node, content)
				if name == "" {
					return
				}
				container := moduleQualified
				kind := "function"
				parent := ""
				if implContainer != "" {
					container = implContainer
					kind = "method"
					parent = implContainer
				} else if traitContainer != "" {
					container = traitContainer
					kind = "method"
					parent = traitContainer
				}
				qualified, _ := core.QualifiedNameFromContainer(container, name)
				sym := core.MakeSymbol(content, node, name, qualified, kind)
				sym.ParentID = parent
				symbols = append(symbols, sym)
				return
			case "const_item":
				appendNamedMemberSymbol(&symbols, node, content, currentContainer(moduleQualified, implContainer, traitContainer), "constant")
				return
			case "type_item", "associated_type":
				appendNamedMemberSymbol(&symbols, node, content, currentContainer(moduleQualified, implContainer, traitContainer), "type")
				return
			case "use_declaration":
				arg := strings.Join(strings.Fields(core.NodeText(node.ChildByFieldName("argument"), content)), " ")
				if arg != "" {
					sym := core.MakeSymbol(content, node, arg, arg, "import")
					symbols = append(symbols, sym)
				}
				return
			case "macro_definition":
				appendNamedMemberSymbol(&symbols, node, content, moduleQualified, "macro")
				return
			}

			core.WalkNamedChildren(node, func(child *treesitter.Node) {
				walk(child, moduleQualified, implContainer, traitContainer)
			})
		}

		walk(root, "", "", "")
		core.SortSymbols(symbols)
		return symbols
	})
}

func appendContainerSymbol(symbols *[]core.Symbol, node *treesitter.Node, content []byte, container, kind string) {
	name := nodeName(node, content)
	if name == "" {
		return
	}
	qualified, parent := core.QualifiedNameFromContainer(container, name)
	sym := core.MakeSymbol(content, node, name, qualified, kind)
	sym.ParentID = parent
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

func nodeName(node *treesitter.Node, content []byte) string {
	if node == nil {
		return ""
	}
	if nameNode := node.ChildByFieldName("name"); nameNode != nil {
		return strings.TrimSpace(core.NodeText(nameNode, content))
	}
	return ""
}

func currentContainer(moduleQualified, implContainer, traitContainer string) string {
	if implContainer != "" {
		return implContainer
	}
	if traitContainer != "" {
		return traitContainer
	}
	return moduleQualified
}

func normalizeTypeName(in string) string {
	in = strings.TrimSpace(in)
	if in == "" {
		return ""
	}
	in = strings.Join(strings.Fields(in), "")
	in = strings.TrimPrefix(in, "&")
	in = strings.TrimPrefix(in, "mut")
	if idx := strings.Index(in, "<"); idx >= 0 {
		in = in[:idx]
	}
	in = strings.TrimPrefix(in, "(")
	in = strings.TrimSuffix(in, ")")
	in = strings.TrimPrefix(in, "dyn")
	in = strings.TrimLeft(in, "&*")
	in = strings.TrimSpace(in)
	in = strings.ReplaceAll(in, "::", ".")
	return in
}
