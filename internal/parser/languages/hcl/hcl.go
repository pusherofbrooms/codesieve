package hcl

import (
	"path/filepath"
	"strings"

	"github.com/pusherofbrooms/codesieve/internal/parser/core"
	tshcl "github.com/pusherofbrooms/codesieve/internal/tslang/hcl"
	treesitter "github.com/tree-sitter/go-tree-sitter"
)

const Name = "hcl"

var Extensions = []string{".tf", ".tfvars", ".hcl"}

func Parse(path string, content []byte) ([]core.Symbol, error) {
	return core.ParseWithTreeSitter(content, treesitter.NewLanguage(tshcl.Language()), func(root *treesitter.Node) []core.Symbol {
		rootName := "file:" + filepath.Base(path)
		file := core.MakeSymbol(content, root, rootName, rootName, "file")
		file.Signature = ""
		symbols := []core.Symbol{file}

		var walk func(node *treesitter.Node, parentID, containerQualified, containerKind string)
		walk = func(node *treesitter.Node, parentID, containerQualified, containerKind string) {
			if node == nil {
				return
			}
			switch node.Kind() {
			case "config_file", "body":
				for i := uint(0); i < node.NamedChildCount(); i++ {
					walk(node.NamedChild(i), parentID, containerQualified, containerKind)
				}
				return
			case "block":
				sym := blockSymbol(node, content)
				if sym.Name == "" || sym.Kind == "" {
					break
				}
				sym.ParentID = parentID
				symbols = append(symbols, sym)
				walk(blockBody(node), sym.QualifiedName, sym.QualifiedName, sym.Kind)
				return
			case "attribute":
				sym := attributeSymbol(node, content, containerQualified, containerKind)
				if sym.Name == "" || sym.Kind == "" {
					return
				}
				sym.ParentID = parentID
				symbols = append(symbols, sym)
				return
			}
			for i := uint(0); i < node.NamedChildCount(); i++ {
				walk(node.NamedChild(i), parentID, containerQualified, containerKind)
			}
		}

		walk(root, rootName, "", "")
		core.SortSymbols(symbols)
		return symbols
	})
}

func blockSymbol(node *treesitter.Node, content []byte) core.Symbol {
	labels := blockLabels(node, content)
	if len(labels) == 0 {
		return core.Symbol{}
	}
	blockType := labels[0]
	rest := labels[1:]
	name := blockType
	qualified := blockType
	kind := "block"

	switch blockType {
	case "resource", "data":
		if len(rest) >= 2 {
			name = rest[1]
			qualified = blockType + "." + rest[0] + "." + rest[1]
			kind = blockType
		}
	case "module", "variable", "output", "provider":
		kind = blockType
		if len(rest) >= 1 {
			name = rest[0]
			qualified = blockType + "." + rest[0]
		}
	case "locals", "terraform":
		kind = blockType
		name = blockType
		qualified = blockType
	default:
		if len(rest) > 0 {
			name = blockType + "." + strings.Join(rest, ".")
			qualified = name
		}
	}

	return core.MakeSymbol(content, node, name, qualified, kind)
}

func attributeSymbol(node *treesitter.Node, content []byte, containerQualified, containerKind string) core.Symbol {
	name := ""
	for i := uint(0); i < node.NamedChildCount(); i++ {
		child := node.NamedChild(i)
		if child != nil && child.Kind() == "identifier" {
			name = cleanLabel(core.NodeText(child, content))
			break
		}
	}
	if name == "" {
		return core.Symbol{}
	}
	qualified := name
	if containerQualified != "" {
		qualified = containerQualified + "." + name
	}
	kind := "argument"
	if containerKind == "locals" {
		kind = "local"
	}
	return core.MakeSymbol(content, node, name, qualified, kind)
}

func blockLabels(node *treesitter.Node, content []byte) []string {
	if node == nil {
		return nil
	}
	labels := make([]string, 0, 3)
	for i := uint(0); i < node.NamedChildCount(); i++ {
		child := node.NamedChild(i)
		if child == nil {
			continue
		}
		switch child.Kind() {
		case "identifier", "string_lit":
			label := cleanLabel(core.NodeText(child, content))
			if label != "" {
				labels = append(labels, label)
			}
		}
	}
	return labels
}

func blockBody(node *treesitter.Node) *treesitter.Node {
	if node == nil {
		return nil
	}
	for i := uint(0); i < node.NamedChildCount(); i++ {
		child := node.NamedChild(i)
		if child != nil && child.Kind() == "body" {
			return child
		}
	}
	return nil
}

func cleanLabel(in string) string {
	in = strings.TrimSpace(in)
	in = strings.TrimPrefix(in, "\"")
	in = strings.TrimSuffix(in, "\"")
	in = strings.TrimPrefix(in, "'")
	in = strings.TrimSuffix(in, "'")
	return strings.TrimSpace(in)
}
