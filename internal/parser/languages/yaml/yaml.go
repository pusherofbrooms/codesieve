package yaml

import (
	"path/filepath"
	"strings"

	"github.com/pusherofbrooms/codesieve/internal/parser/core"
	"github.com/pusherofbrooms/codesieve/internal/parser/languages/register"
	"github.com/pusherofbrooms/codesieve/internal/parser/structured/cfn"
	tsyaml "github.com/pusherofbrooms/codesieve/internal/tslang/yaml"
	treesitter "github.com/tree-sitter/go-tree-sitter"
)

const Name = "yaml"

var Extensions = []string{".yaml", ".yml"}

func init() {
	register.MustRegister(Name, Parse)
}

func Parse(path string, content []byte) ([]core.Symbol, error) {
	return core.ParseWithTreeSitter(content, treesitter.NewLanguage(tsyaml.Language()), func(root *treesitter.Node) []core.Symbol {
		base := filepath.Base(path)
		rootName := "document:" + base
		rootKind := "document"

		topPairNodes := topLevelPairs(root)
		topPairs := toCFNPairs(topPairNodes, content)
		isCFN := cfn.IsTemplate(topPairs, string(content))
		if isCFN {
			rootName = "template:" + base
			rootKind = "template"
		}

		symbols := []core.Symbol{core.MakeSymbol(content, root, rootName, rootName, rootKind)}
		seen := map[string]struct{}{rootName + "#" + rootKind: {}}

		if isCFN {
			cfn.ExtractSymbols(rootName, topPairs, cfn.Ops[*treesitter.Node]{
				IsZero: func(n *treesitter.Node) bool { return n == nil },
				PairsFromValue: func(n *treesitter.Node) []cfn.Pair[*treesitter.Node] {
					return toCFNPairs(mappingPairsFromValue(n), content)
				},
				ScalarValue:   func(n *treesitter.Node) string { return scalarValue(n, content) },
				NodeText:      func(n *treesitter.Node) string { return core.NodeText(n, content) },
				NamedChildren: namedChildren,
				MakeSymbol: func(n *treesitter.Node, name, qualified, kind string) core.Symbol {
					return core.MakeSymbol(content, n, name, qualified, kind)
				},
				Emit: func(sym core.Symbol, parent string) { appendUniqueSymbol(&symbols, seen, sym, parent) },
			})
		} else {
			walkGenericPairs(&symbols, seen, rootName, topPairNodes, nil, content)
		}

		core.SortSymbols(symbols)
		return symbols
	})
}

func topLevelPairs(root *treesitter.Node) []*treesitter.Node {
	if root == nil {
		return nil
	}
	var pairs []*treesitter.Node
	for i := uint(0); i < root.NamedChildCount(); i++ {
		pairs = append(pairs, mappingPairsFromValue(root.NamedChild(i))...)
	}
	return pairs
}

func toCFNPairs(nodes []*treesitter.Node, content []byte) []cfn.Pair[*treesitter.Node] {
	pairs := make([]cfn.Pair[*treesitter.Node], 0, len(nodes))
	for _, node := range nodes {
		if node == nil {
			continue
		}
		pairs = append(pairs, cfn.Pair[*treesitter.Node]{
			Node:  node,
			Key:   pairKeyName(node, content),
			Value: node.ChildByFieldName("value"),
		})
	}
	return pairs
}

func namedChildren(node *treesitter.Node) []*treesitter.Node {
	if node == nil {
		return nil
	}
	out := make([]*treesitter.Node, 0, node.NamedChildCount())
	for i := uint(0); i < node.NamedChildCount(); i++ {
		out = append(out, node.NamedChild(i))
	}
	return out
}

func walkGenericPairs(symbols *[]core.Symbol, seen map[string]struct{}, parent string, pairs []*treesitter.Node, path []string, content []byte) {
	for _, pair := range pairs {
		if pair == nil {
			continue
		}
		key := pairKeyName(pair, content)
		if key == "" {
			continue
		}
		full := append(append([]string{}, path...), key)
		qualified := strings.Join(full, ".")
		appendUniqueSymbol(symbols, seen, core.MakeSymbol(content, pair, key, qualified, "key"), parent)
		walkValue(symbols, seen, qualified, pair.ChildByFieldName("value"), full, content)
	}
}

func walkValue(symbols *[]core.Symbol, seen map[string]struct{}, parent string, node *treesitter.Node, path []string, content []byte) {
	if node == nil {
		return
	}
	switch node.Kind() {
	case "block_mapping", "flow_mapping":
		pairs := mappingPairs(node)
		walkGenericPairs(symbols, seen, parent, pairs, path, content)
		return
	case "block_sequence", "flow_sequence", "block_sequence_item", "block_node", "flow_node", "document", "stream":
		for i := uint(0); i < node.NamedChildCount(); i++ {
			walkValue(symbols, seen, parent, node.NamedChild(i), path, content)
		}
	}
}

func mappingPairs(node *treesitter.Node) []*treesitter.Node {
	if node == nil {
		return nil
	}
	pairs := make([]*treesitter.Node, 0, node.NamedChildCount())
	for i := uint(0); i < node.NamedChildCount(); i++ {
		child := node.NamedChild(i)
		if child == nil {
			continue
		}
		if child.Kind() == "block_mapping_pair" || child.Kind() == "flow_pair" {
			pairs = append(pairs, child)
		}
	}
	return pairs
}

func pairKeyName(pair *treesitter.Node, content []byte) string {
	if pair == nil {
		return ""
	}
	keyNode := pair.ChildByFieldName("key")
	return scalarValue(keyNode, content)
}

func scalarValue(node *treesitter.Node, content []byte) string {
	if node == nil {
		return ""
	}
	switch node.Kind() {
	case "block_node", "flow_node", "plain_scalar", "block_sequence_item", "document":
		if node.NamedChildCount() > 0 {
			return scalarValue(node.NamedChild(0), content)
		}
	case "single_quote_scalar", "double_quote_scalar", "string_scalar", "integer_scalar", "float_scalar", "boolean_scalar", "null_scalar", "timestamp_scalar", "block_scalar", "anchor_name", "alias_name":
		text := strings.TrimSpace(core.NodeText(node, content))
		return strings.Trim(text, `"'`)
	}
	text := strings.TrimSpace(core.NodeText(node, content))
	text = strings.Trim(text, `"'`)
	if idx := strings.IndexAny(text, "\n\r"); idx >= 0 {
		text = text[:idx]
	}
	return strings.TrimSpace(text)
}

func mappingPairsFromValue(node *treesitter.Node) []*treesitter.Node {
	if node == nil {
		return nil
	}
	switch node.Kind() {
	case "block_mapping", "flow_mapping":
		return mappingPairs(node)
	}
	for i := uint(0); i < node.NamedChildCount(); i++ {
		if pairs := mappingPairsFromValue(node.NamedChild(i)); len(pairs) > 0 {
			return pairs
		}
	}
	return nil
}

func appendUniqueSymbol(symbols *[]core.Symbol, seen map[string]struct{}, sym core.Symbol, parent string) {
	if sym.Name == "" || sym.Kind == "" {
		return
	}
	key := sym.QualifiedName + "#" + sym.Kind
	if _, ok := seen[key]; ok {
		return
	}
	sym.ParentID = parent
	*symbols = append(*symbols, sym)
	seen[key] = struct{}{}
}
