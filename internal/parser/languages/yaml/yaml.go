package yaml

import (
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/pusherofbrooms/codesieve/internal/parser/core"
	tsyaml "github.com/pusherofbrooms/codesieve/internal/tslang/yaml"
	treesitter "github.com/tree-sitter/go-tree-sitter"
)

const Name = "yaml"

var Extensions = []string{".yaml", ".yml"}

var cfnTopLevelKeys = map[string]struct{}{
	"AWSTemplateFormatVersion": {},
	"Description":              {},
	"Metadata":                 {},
	"Parameters":               {},
	"Mappings":                 {},
	"Conditions":               {},
	"Transform":                {},
	"Resources":                {},
	"Outputs":                  {},
	"Rules":                    {},
}

var cfnIntrinsicKeySet = map[string]struct{}{
	"Ref":              {},
	"Fn::Base64":       {},
	"Fn::Cidr":         {},
	"Fn::FindInMap":    {},
	"Fn::GetAtt":       {},
	"Fn::GetAZs":       {},
	"Fn::If":           {},
	"Fn::ImportValue":  {},
	"Fn::Join":         {},
	"Fn::Length":       {},
	"Fn::Select":       {},
	"Fn::Split":        {},
	"Fn::Sub":          {},
	"Fn::ToJsonString": {},
	"Fn::Transform":    {},
	"Fn::And":          {},
	"Fn::Equals":       {},
	"Fn::Not":          {},
	"Fn::Or":           {},
}

var cfnRefRegex = regexp.MustCompile(`!Ref\s+([A-Za-z0-9._:-]+)`)
var cfnGetAttRegex = regexp.MustCompile(`!GetAtt\s+([A-Za-z0-9._:-]+)`)
var cfnSubRefRegex = regexp.MustCompile(`\$\{([A-Za-z0-9._:-]+)\}`)

func Parse(path string, content []byte) ([]core.Symbol, error) {
	return core.ParseWithTreeSitter(content, treesitter.NewLanguage(tsyaml.Language()), func(root *treesitter.Node) []core.Symbol {
		base := filepath.Base(path)
		rootName := "document:" + base
		rootKind := "document"

		topPairs := topLevelPairs(root)
		isCFN := isCloudFormationTemplate(topPairs, content)
		if isCFN {
			rootName = "template:" + base
			rootKind = "template"
		}

		symbols := []core.Symbol{core.MakeSymbol(content, root, rootName, rootName, rootKind)}
		seen := map[string]struct{}{rootName + "#" + rootKind: {}}

		if isCFN {
			extractCloudFormationSymbols(&symbols, seen, rootName, topPairs, content)
		} else {
			walkGenericPairs(&symbols, seen, rootName, topPairs, nil, content)
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

func isCloudFormationTemplate(pairs []*treesitter.Node, content []byte) bool {
	top := map[string]bool{}
	intrinsicCount := 0
	for _, pair := range pairs {
		key := pairKeyName(pair, content)
		if key == "" {
			continue
		}
		top[key] = true
		if _, ok := cfnIntrinsicKeySet[key]; ok {
			intrinsicCount++
		}
	}

	hasResources := top["Resources"]
	hasSignal := top["AWSTemplateFormatVersion"] || top["Transform"] || top["Parameters"] || top["Outputs"] || top["Mappings"] || top["Conditions"]
	if hasResources && hasSignal {
		return true
	}

	if hasResources {
		text := string(content)
		if strings.Contains(text, "Fn::") || strings.Contains(text, "!Ref") || strings.Contains(text, "!Sub") || strings.Contains(text, "!GetAtt") {
			return true
		}
	}

	count := 0
	for key := range top {
		if _, ok := cfnTopLevelKeys[key]; ok {
			count++
		}
	}
	if count >= 3 {
		return true
	}
	return intrinsicCount > 0 && hasResources
}

func extractCloudFormationSymbols(symbols *[]core.Symbol, seen map[string]struct{}, rootName string, topPairs []*treesitter.Node, content []byte) {
	sections := map[string]*treesitter.Node{}
	for _, pair := range topPairs {
		key := pairKeyName(pair, content)
		if key == "" {
			continue
		}
		qualified := key
		appendUniqueSymbol(symbols, seen, core.MakeSymbol(content, pair, key, qualified, "section"), rootName)
		sections[key] = pair.ChildByFieldName("value")
	}

	extractCFNNamedSection(symbols, seen, "Parameters", "parameter", sections["Parameters"], content)
	extractCFNNamedSection(symbols, seen, "Conditions", "condition", sections["Conditions"], content)
	extractCFNNamedSection(symbols, seen, "Mappings", "mapping", sections["Mappings"], content)
	extractCFNOutputs(symbols, seen, sections["Outputs"], content)
	extractCFNResources(symbols, seen, sections["Resources"], content)
}

func extractCFNNamedSection(symbols *[]core.Symbol, seen map[string]struct{}, section, kind string, node *treesitter.Node, content []byte) {
	for _, pair := range mappingPairsFromValue(node) {
		name := pairKeyName(pair, content)
		if name == "" {
			continue
		}
		qualified := section + "." + name
		appendUniqueSymbol(symbols, seen, core.MakeSymbol(content, pair, name, qualified, kind), section)
		collectCFNRefs(symbols, seen, qualified, pair.ChildByFieldName("value"), content)
	}
}

func extractCFNOutputs(symbols *[]core.Symbol, seen map[string]struct{}, node *treesitter.Node, content []byte) {
	for _, pair := range mappingPairsFromValue(node) {
		name := pairKeyName(pair, content)
		if name == "" {
			continue
		}
		qualified := "Outputs." + name
		appendUniqueSymbol(symbols, seen, core.MakeSymbol(content, pair, name, qualified, "output"), "Outputs")
		collectCFNRefs(symbols, seen, qualified, pair.ChildByFieldName("value"), content)
	}
}

func extractCFNResources(symbols *[]core.Symbol, seen map[string]struct{}, node *treesitter.Node, content []byte) {
	for _, pair := range mappingPairsFromValue(node) {
		name := pairKeyName(pair, content)
		if name == "" {
			continue
		}
		qualified := "Resources." + name
		resourceSym := core.MakeSymbol(content, pair, name, qualified, "resource")
		if typ := cloudFormationResourceType(pair.ChildByFieldName("value"), content); typ != "" {
			resourceSym.Signature = typ
		}
		appendUniqueSymbol(symbols, seen, resourceSym, "Resources")
		collectCFNRefs(symbols, seen, qualified, pair.ChildByFieldName("value"), content)
	}
}

func cloudFormationResourceType(node *treesitter.Node, content []byte) string {
	for _, pair := range mappingPairsFromValue(node) {
		if pairKeyName(pair, content) != "Type" {
			continue
		}
		return scalarValue(pair.ChildByFieldName("value"), content)
	}
	return ""
}

func collectCFNRefs(symbols *[]core.Symbol, seen map[string]struct{}, parent string, node *treesitter.Node, content []byte) {
	if node == nil {
		return
	}
	refs := map[string]struct{}{}
	collectCFNRefsFromNode(node, refs, content)
	ordered := make([]string, 0, len(refs))
	for ref := range refs {
		if ref != "" {
			ordered = append(ordered, ref)
		}
	}
	sort.Strings(ordered)
	for _, ref := range ordered {
		qualified := parent + ".ref." + ref
		appendUniqueSymbol(symbols, seen, core.MakeSymbol(content, node, ref, qualified, "reference"), parent)
	}
}

func collectCFNRefsFromNode(node *treesitter.Node, refs map[string]struct{}, content []byte) {
	if node == nil {
		return
	}
	if node.Kind() == "block_mapping_pair" || node.Kind() == "flow_pair" {
		key := pairKeyName(node, content)
		value := node.ChildByFieldName("value")
		switch key {
		case "Ref":
			if target := scalarValue(value, content); target != "" {
				refs[target] = struct{}{}
			}
		case "Fn::GetAtt":
			target := scalarValue(value, content)
			if idx := strings.Index(target, "."); idx > 0 {
				target = target[:idx]
			}
			if target != "" {
				refs[target] = struct{}{}
			}
		case "Fn::Sub":
			for _, m := range cfnSubRefRegex.FindAllStringSubmatch(core.NodeText(value, content), -1) {
				if len(m) < 2 {
					continue
				}
				target := m[1]
				if idx := strings.Index(target, "."); idx > 0 {
					target = target[:idx]
				}
				if target != "" {
					refs[target] = struct{}{}
				}
			}
		}
	}
	text := core.NodeText(node, content)
	for _, m := range cfnRefRegex.FindAllStringSubmatch(text, -1) {
		if len(m) > 1 && m[1] != "" {
			refs[m[1]] = struct{}{}
		}
	}
	for _, m := range cfnGetAttRegex.FindAllStringSubmatch(text, -1) {
		if len(m) < 2 || m[1] == "" {
			continue
		}
		target := m[1]
		if idx := strings.Index(target, "."); idx > 0 {
			target = target[:idx]
		}
		refs[target] = struct{}{}
	}
	for _, m := range cfnSubRefRegex.FindAllStringSubmatch(text, -1) {
		if len(m) < 2 || m[1] == "" {
			continue
		}
		target := m[1]
		if idx := strings.Index(target, "."); idx > 0 {
			target = target[:idx]
		}
		refs[target] = struct{}{}
	}
	for i := uint(0); i < node.NamedChildCount(); i++ {
		collectCFNRefsFromNode(node.NamedChild(i), refs, content)
	}
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
