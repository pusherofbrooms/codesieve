package cfn

import (
	"regexp"
	"sort"
	"strings"

	"github.com/pusherofbrooms/codesieve/internal/parser/core"
)

type Pair[N any] struct {
	Node  N
	Key   string
	Value N
}

type Ops[N any] struct {
	IsZero         func(N) bool
	PairsFromValue func(N) []Pair[N]
	ScalarValue    func(N) string
	NodeText       func(N) string
	NamedChildren  func(N) []N
	MakeSymbol     func(N, string, string, string) core.Symbol
	Emit           func(core.Symbol, string)
}

var topLevelKeys = map[string]struct{}{
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

var intrinsicKeySet = map[string]struct{}{
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

var refRegex = regexp.MustCompile(`!Ref\s+([A-Za-z0-9._:-]+)`)
var getAttRegex = regexp.MustCompile(`!GetAtt\s+([A-Za-z0-9._:-]+)`)
var subRefRegex = regexp.MustCompile(`\$\{([A-Za-z0-9._:-]+)\}`)

func IsTemplate[N any](topPairs []Pair[N], fullText string) bool {
	top := map[string]bool{}
	intrinsicCount := 0
	for _, pair := range topPairs {
		if pair.Key == "" {
			continue
		}
		top[pair.Key] = true
		if _, ok := intrinsicKeySet[pair.Key]; ok {
			intrinsicCount++
		}
	}

	hasResources := top["Resources"]
	hasSignal := top["AWSTemplateFormatVersion"] || top["Transform"] || top["Parameters"] || top["Outputs"] || top["Mappings"] || top["Conditions"]
	if hasResources && hasSignal {
		return true
	}

	if hasResources {
		if strings.Contains(fullText, "Fn::") || strings.Contains(fullText, "!Ref") || strings.Contains(fullText, "!Sub") || strings.Contains(fullText, "!GetAtt") {
			return true
		}
	}

	count := 0
	for key := range top {
		if _, ok := topLevelKeys[key]; ok {
			count++
		}
	}
	if count >= 3 {
		return true
	}
	return intrinsicCount > 0 && hasResources
}

func ExtractSymbols[N any](rootParent string, topPairs []Pair[N], ops Ops[N]) {
	sections := map[string]N{}
	for _, pair := range topPairs {
		if pair.Key == "" {
			continue
		}
		ops.Emit(ops.MakeSymbol(pair.Node, pair.Key, pair.Key, "section"), rootParent)
		sections[pair.Key] = pair.Value
	}

	extractNamedSection("Parameters", "parameter", sections["Parameters"], ops)
	extractNamedSection("Conditions", "condition", sections["Conditions"], ops)
	extractNamedSection("Mappings", "mapping", sections["Mappings"], ops)
	extractOutputs(sections["Outputs"], ops)
	extractResources(sections["Resources"], ops)
}

func extractNamedSection[N any](section, kind string, node N, ops Ops[N]) {
	if ops.IsZero(node) {
		return
	}
	for _, pair := range ops.PairsFromValue(node) {
		if pair.Key == "" {
			continue
		}
		qualified := section + "." + pair.Key
		ops.Emit(ops.MakeSymbol(pair.Node, pair.Key, qualified, kind), section)
		collectRefs(qualified, pair.Value, ops)
	}
}

func extractOutputs[N any](node N, ops Ops[N]) {
	if ops.IsZero(node) {
		return
	}
	for _, pair := range ops.PairsFromValue(node) {
		if pair.Key == "" {
			continue
		}
		qualified := "Outputs." + pair.Key
		ops.Emit(ops.MakeSymbol(pair.Node, pair.Key, qualified, "output"), "Outputs")
		collectRefs(qualified, pair.Value, ops)
	}
}

func extractResources[N any](node N, ops Ops[N]) {
	if ops.IsZero(node) {
		return
	}
	for _, pair := range ops.PairsFromValue(node) {
		if pair.Key == "" {
			continue
		}
		qualified := "Resources." + pair.Key
		sym := ops.MakeSymbol(pair.Node, pair.Key, qualified, "resource")
		if typ := resourceType(pair.Value, ops); typ != "" {
			sym.Signature = typ
		}
		ops.Emit(sym, "Resources")
		collectRefs(qualified, pair.Value, ops)
	}
}

func resourceType[N any](node N, ops Ops[N]) string {
	if ops.IsZero(node) {
		return ""
	}
	for _, pair := range ops.PairsFromValue(node) {
		if pair.Key != "Type" {
			continue
		}
		return ops.ScalarValue(pair.Value)
	}
	return ""
}

func collectRefs[N any](parent string, node N, ops Ops[N]) {
	if ops.IsZero(node) {
		return
	}
	refs := map[string]struct{}{}
	collectRefsFromNode(node, refs, ops)
	ordered := make([]string, 0, len(refs))
	for ref := range refs {
		if ref != "" {
			ordered = append(ordered, ref)
		}
	}
	sort.Strings(ordered)
	for _, ref := range ordered {
		ops.Emit(ops.MakeSymbol(node, ref, parent+".ref."+ref, "reference"), parent)
	}
}

func collectRefsFromNode[N any](node N, refs map[string]struct{}, ops Ops[N]) {
	if ops.IsZero(node) {
		return
	}
	for _, pair := range ops.PairsFromValue(node) {
		switch pair.Key {
		case "Ref":
			if target := ops.ScalarValue(pair.Value); target != "" {
				refs[target] = struct{}{}
			}
		case "Fn::GetAtt":
			if target := cutRefTarget(ops.ScalarValue(pair.Value)); target != "" {
				refs[target] = struct{}{}
			}
		case "Fn::Sub":
			for _, m := range subRefRegex.FindAllStringSubmatch(ops.NodeText(pair.Value), -1) {
				if len(m) < 2 {
					continue
				}
				if target := cutRefTarget(m[1]); target != "" {
					refs[target] = struct{}{}
				}
			}
		}
		collectRefsFromNode(pair.Value, refs, ops)
	}

	text := ops.NodeText(node)
	for _, m := range refRegex.FindAllStringSubmatch(text, -1) {
		if len(m) > 1 && m[1] != "" {
			refs[m[1]] = struct{}{}
		}
	}
	for _, m := range getAttRegex.FindAllStringSubmatch(text, -1) {
		if len(m) < 2 || m[1] == "" {
			continue
		}
		if target := cutRefTarget(m[1]); target != "" {
			refs[target] = struct{}{}
		}
	}
	for _, m := range subRefRegex.FindAllStringSubmatch(text, -1) {
		if len(m) < 2 || m[1] == "" {
			continue
		}
		if target := cutRefTarget(m[1]); target != "" {
			refs[target] = struct{}{}
		}
	}

	for _, child := range ops.NamedChildren(node) {
		collectRefsFromNode(child, refs, ops)
	}
}

func cutRefTarget(target string) string {
	if idx := strings.Index(target, "."); idx > 0 {
		return target[:idx]
	}
	return target
}
