package nix

import (
	"path/filepath"
	"strings"

	"github.com/pusherofbrooms/codesieve/internal/parser/core"
	"github.com/pusherofbrooms/codesieve/internal/parser/languages/register"
	tsnix "github.com/pusherofbrooms/codesieve/internal/tslang/nix"
	treesitter "github.com/tree-sitter/go-tree-sitter"
)

const Name = "nix"

var Extensions = []string{".nix"}

func init() {
	register.MustRegister(Name, Parse)
}

func Parse(path string, content []byte) ([]core.Symbol, error) {
	return core.ParseWithTreeSitter(content, treesitter.NewLanguage(tsnix.Language()), func(root *treesitter.Node) []core.Symbol {
		base := filepath.Base(path)
		rootName := "file:" + base
		ex := extractor{
			content: content,
			seen:    map[string]struct{}{},
		}

		file := core.MakeSymbol(content, root, rootName, rootName, "file")
		file.Signature = ""
		ex.append(file, "")
		ex.walk(root.ChildByFieldName("expression"), rootName, "")

		core.SortSymbols(ex.symbols)
		return ex.symbols
	})
}

type extractor struct {
	content []byte
	symbols []core.Symbol
	seen    map[string]struct{}
}

func (e *extractor) append(sym core.Symbol, parent string) {
	if sym.Name == "" || sym.Kind == "" {
		return
	}
	key := sym.Kind + ":" + sym.QualifiedName
	if _, ok := e.seen[key]; ok {
		return
	}
	sym.ParentID = parent
	e.symbols = append(e.symbols, sym)
	e.seen[key] = struct{}{}
}

func (e *extractor) walk(node *treesitter.Node, parentID, containerQualified string) {
	if node == nil {
		return
	}

	switch node.Kind() {
	case "source_code", "parenthesized_expression", "assert_expression", "with_expression":
		e.walk(node.ChildByFieldName("expression"), parentID, containerQualified)
		e.walk(node.ChildByFieldName("body"), parentID, containerQualified)
		return
	case "function_expression":
		e.walk(node.ChildByFieldName("body"), parentID, containerQualified)
		return
	case "if_expression":
		e.walk(node.ChildByFieldName("consequence"), parentID, containerQualified)
		e.walk(node.ChildByFieldName("alternative"), parentID, containerQualified)
		return
	case "let_expression":
		e.walkBindingSet(node, parentID, containerQualified)
		e.walk(node.ChildByFieldName("body"), parentID, containerQualified)
		return
	case "attrset_expression", "rec_attrset_expression", "let_attrset_expression":
		e.walkBindingSet(node, parentID, containerQualified)
		return
	case "apply_expression":
		e.walk(node.ChildByFieldName("function"), parentID, containerQualified)
		e.walk(node.ChildByFieldName("argument"), parentID, containerQualified)
		return
	case "select_expression":
		e.walk(node.ChildByFieldName("expression"), parentID, containerQualified)
		return
	}

	for i := uint(0); i < node.NamedChildCount(); i++ {
		e.walk(node.NamedChild(i), parentID, containerQualified)
	}
}

func (e *extractor) walkBindingSet(node *treesitter.Node, parentID, containerQualified string) {
	bindingSet := firstBindingSet(node)
	if bindingSet == nil {
		return
	}
	for i := uint(0); i < bindingSet.NamedChildCount(); i++ {
		child := bindingSet.NamedChild(i)
		if child == nil {
			continue
		}
		switch child.Kind() {
		case "binding":
			parts := attrpathParts(child.ChildByFieldName("attrpath"), e.content)
			if len(parts) == 0 {
				continue
			}
			expr := child.ChildByFieldName("expression")
			containerParts := splitQualified(containerQualified)
			fullParts := withContainer(containerQualified, parts)
			leafQualified := strings.Join(fullParts, ".")
			parent := parentID
			for i := len(containerParts); i < len(fullParts)-1; i++ {
				prefixQualified := strings.Join(fullParts[:i+1], ".")
				prefixName := fullParts[i]
				prefixNode := child.ChildByFieldName("attrpath")
				if prefixNode == nil {
					prefixNode = child
				}
				e.append(core.MakeSymbol(e.content, prefixNode, prefixName, prefixQualified, "attrset"), parent)
				parent = prefixQualified
			}
			leafName := fullParts[len(fullParts)-1]
			kind := bindingKind(expr)
			leaf := core.MakeSymbol(e.content, child, leafName, leafQualified, kind)
			e.append(leaf, parent)
			if kind == "function" {
				e.appendFunctionParams(expr, leafQualified)
			}
			e.walk(expr, leafQualified, leafQualified)
		case "inherit", "inherit_from":
			attrs := inheritedAttrs(child, e.content)
			for _, name := range attrs {
				qualified := qualify(containerQualified, name)
				e.append(core.MakeSymbol(e.content, child, name, qualified, "inherit"), parentID)
			}
			if child.Kind() == "inherit_from" {
				e.walk(child.ChildByFieldName("expression"), parentID, containerQualified)
			}
		}
	}
}

func (e *extractor) appendFunctionParams(node *treesitter.Node, fnQualified string) {
	for node != nil && node.Kind() == "function_expression" {
		if universal := cleanText(core.NodeText(node.ChildByFieldName("universal"), e.content)); universal != "" {
			qualified := fnQualified + "." + universal
			e.append(core.MakeSymbol(e.content, node, universal, qualified, "parameter"), fnQualified)
		}
		formals := node.ChildByFieldName("formals")
		if formals != nil {
			for i := uint(0); i < formals.NamedChildCount(); i++ {
				child := formals.NamedChild(i)
				if child == nil || child.Kind() != "formal" {
					continue
				}
				name := cleanText(core.NodeText(child.ChildByFieldName("name"), e.content))
				if name == "" {
					continue
				}
				qualified := fnQualified + "." + name
				e.append(core.MakeSymbol(e.content, child, name, qualified, "parameter"), fnQualified)
			}
		}
		node = node.ChildByFieldName("body")
	}
}

func bindingKind(expr *treesitter.Node) string {
	if expr == nil {
		return "binding"
	}
	switch expr.Kind() {
	case "function_expression":
		return "function"
	case "attrset_expression", "rec_attrset_expression", "let_attrset_expression":
		return "attrset"
	default:
		return "binding"
	}
}

func firstBindingSet(node *treesitter.Node) *treesitter.Node {
	if node == nil {
		return nil
	}
	for i := uint(0); i < node.NamedChildCount(); i++ {
		child := node.NamedChild(i)
		if child != nil && child.Kind() == "binding_set" {
			return child
		}
	}
	return nil
}

func attrpathParts(node *treesitter.Node, content []byte) []string {
	if node == nil || node.Kind() != "attrpath" {
		return nil
	}
	parts := make([]string, 0, node.NamedChildCount())
	for i := uint(0); i < node.NamedChildCount(); i++ {
		child := node.NamedChild(i)
		if child == nil {
			continue
		}
		part := cleanText(core.NodeText(child, content))
		if part == "" {
			continue
		}
		parts = append(parts, part)
	}
	return parts
}

func inheritedAttrs(node *treesitter.Node, content []byte) []string {
	for i := uint(0); i < node.NamedChildCount(); i++ {
		child := node.NamedChild(i)
		if child == nil || child.Kind() != "inherited_attrs" {
			continue
		}
		attrs := make([]string, 0, child.NamedChildCount())
		for j := uint(0); j < child.NamedChildCount(); j++ {
			name := cleanText(core.NodeText(child.NamedChild(j), content))
			if name != "" {
				attrs = append(attrs, name)
			}
		}
		return attrs
	}
	return nil
}

func withContainer(container string, parts []string) []string {
	if container == "" {
		return append([]string{}, parts...)
	}
	out := strings.Split(container, ".")
	out = append(out, parts...)
	return out
}

func qualify(container, name string) string {
	if container == "" {
		return name
	}
	return container + "." + name
}

func splitQualified(in string) []string {
	if strings.TrimSpace(in) == "" {
		return nil
	}
	return strings.Split(in, ".")
}

func cleanText(in string) string {
	in = strings.TrimSpace(in)
	in = strings.Trim(in, `"'`)
	return strings.TrimSpace(in)
}
