package hcl

import (
	"path/filepath"
	"strconv"
	"strings"

	"github.com/pusherofbrooms/codesieve/internal/parser/core"
	"github.com/pusherofbrooms/codesieve/internal/parser/filetype"
	tshcl "github.com/pusherofbrooms/codesieve/internal/tslang/hcl"
	treesitter "github.com/tree-sitter/go-tree-sitter"
)

const Name = "hcl"

var Extensions = []string{".tf", ".tfvars", ".hcl"}

func Parse(path string, content []byte) ([]core.Symbol, error) {
	return core.ParseWithTreeSitter(content, treesitter.NewLanguage(tshcl.Language()), func(root *treesitter.Node) []core.Symbol {
		rootName := "file:" + filepath.Base(path)
		ex := &extractor{content: content, uniqueQualified: map[string]int{}}

		file := core.MakeSymbol(content, root, rootName, rootName, "file")
		file.Signature = ""
		ex.append(file, "")

		if filetype.IsTerraformJSONPath(path) {
			extractTerraformJSON(root, rootName, ex)
		} else {
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
					sym := blockSymbol(node, content, containerQualified)
					if sym.Name == "" || sym.Kind == "" {
						break
					}
					sym = ex.append(sym, parentID)
					walk(blockBody(node), sym.QualifiedName, sym.QualifiedName, sym.Kind)
					return
				case "attribute":
					sym := attributeSymbol(node, content, containerQualified, containerKind)
					if sym.Name == "" || sym.Kind == "" {
						return
					}
					ex.append(sym, parentID)
					return
				}
				for i := uint(0); i < node.NamedChildCount(); i++ {
					walk(node.NamedChild(i), parentID, containerQualified, containerKind)
				}
			}
			walk(root, rootName, "", "")
		}

		core.SortSymbols(ex.symbols)
		return ex.symbols
	})
}

type extractor struct {
	content         []byte
	symbols         []core.Symbol
	uniqueQualified map[string]int
}

func (e *extractor) append(sym core.Symbol, parentID string) core.Symbol {
	if sym.Name == "" || sym.Kind == "" {
		return core.Symbol{}
	}
	if sym.QualifiedName == "" {
		sym.QualifiedName = sym.Name
	}
	count := e.uniqueQualified[sym.QualifiedName] + 1
	e.uniqueQualified[sym.QualifiedName] = count
	if count > 1 {
		sym.QualifiedName = sym.QualifiedName + "[" + strconv.Itoa(count) + "]"
	}
	sym.ParentID = parentID
	e.symbols = append(e.symbols, sym)
	return sym
}

func blockSymbol(node *treesitter.Node, content []byte, containerQualified string) core.Symbol {
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
		} else {
			name = blockType
			qualified = blockType
		}
	}

	if containerQualified != "" {
		qualified = containerQualified + "." + qualified
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

func extractTerraformJSON(root *treesitter.Node, rootName string, ex *extractor) {
	obj := firstObjectNode(root)
	if obj == nil {
		return
	}
	for _, elem := range objectElements(obj) {
		section := objectElemKeyName(elem, ex.content)
		val := objectElemValue(elem)
		switch section {
		case "resource", "data":
			extractDoubleLabelBlocksJSON(section, val, rootName, ex)
		case "module", "variable", "output", "provider":
			extractSingleLabelBlocksJSON(section, val, rootName, ex)
		case "locals", "terraform":
			sym := ex.append(core.MakeSymbol(ex.content, elem, section, section, section), rootName)
			extractObjectAttributesJSON(val, sym.QualifiedName, sym.QualifiedName, section, ex)
		}
	}
}

func extractSingleLabelBlocksJSON(kind string, node *treesitter.Node, rootName string, ex *extractor) {
	for _, elem := range objectElements(node) {
		name := objectElemKeyName(elem, ex.content)
		if name == "" {
			continue
		}
		qualified := kind + "." + name
		sym := ex.append(core.MakeSymbol(ex.content, elem, name, qualified, kind), rootName)
		extractObjectAttributesJSON(objectElemValue(elem), sym.QualifiedName, sym.QualifiedName, kind, ex)
	}
}

func extractDoubleLabelBlocksJSON(kind string, node *treesitter.Node, rootName string, ex *extractor) {
	for _, typeElem := range objectElements(node) {
		typeName := objectElemKeyName(typeElem, ex.content)
		typeVal := objectElemValue(typeElem)
		if typeName == "" {
			continue
		}
		for _, nameElem := range objectElements(typeVal) {
			name := objectElemKeyName(nameElem, ex.content)
			if name == "" {
				continue
			}
			qualified := kind + "." + typeName + "." + name
			sym := ex.append(core.MakeSymbol(ex.content, nameElem, name, qualified, kind), rootName)
			extractObjectAttributesJSON(objectElemValue(nameElem), sym.QualifiedName, sym.QualifiedName, kind, ex)
		}
	}
}

func extractObjectAttributesJSON(node *treesitter.Node, parentID, containerQualified, containerKind string, ex *extractor) {
	for _, elem := range objectElements(node) {
		name := objectElemKeyName(elem, ex.content)
		if name == "" {
			continue
		}
		kind := "argument"
		if containerKind == "locals" {
			kind = "local"
		}
		qualified := containerQualified + "." + name
		ex.append(core.MakeSymbol(ex.content, elem, name, qualified, kind), parentID)
	}
}

func objectElements(node *treesitter.Node) []*treesitter.Node {
	obj := toObjectNode(node)
	if obj == nil {
		return nil
	}
	out := make([]*treesitter.Node, 0, obj.NamedChildCount())
	for i := uint(0); i < obj.NamedChildCount(); i++ {
		child := obj.NamedChild(i)
		if child != nil && child.Kind() == "object_elem" {
			out = append(out, child)
		}
	}
	return out
}

func objectElemKeyName(elem *treesitter.Node, content []byte) string {
	key, _ := objectElemParts(elem)
	if key == nil {
		return ""
	}
	return cleanLabel(core.NodeText(key, content))
}

func objectElemValue(elem *treesitter.Node) *treesitter.Node {
	_, value := objectElemParts(elem)
	return value
}

func objectElemParts(elem *treesitter.Node) (key, value *treesitter.Node) {
	if elem == nil || elem.Kind() != "object_elem" {
		return nil, nil
	}
	if elem.NamedChildCount() >= 1 {
		key = elem.NamedChild(0)
	}
	if elem.NamedChildCount() >= 2 {
		value = elem.NamedChild(1)
	}
	return key, value
}

func toObjectNode(node *treesitter.Node) *treesitter.Node {
	if node == nil {
		return nil
	}
	if node.Kind() == "object" {
		return node
	}
	if node.Kind() == "expression" || node.Kind() == "collection_value" {
		for i := uint(0); i < node.NamedChildCount(); i++ {
			if obj := toObjectNode(node.NamedChild(i)); obj != nil {
				return obj
			}
		}
	}
	return nil
}

func firstObjectNode(node *treesitter.Node) *treesitter.Node {
	if node == nil {
		return nil
	}
	if node.Kind() == "object" {
		return node
	}
	for i := uint(0); i < node.NamedChildCount(); i++ {
		if found := firstObjectNode(node.NamedChild(i)); found != nil {
			return found
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
