package cfn

import (
	"testing"

	"github.com/pusherofbrooms/codesieve/internal/parser/core"
)

type testNode struct {
	text        string
	scalar      string
	key         string
	value       *testNode
	objectPairs []*testNode
	children    []*testNode
}

func TestIsTemplate(t *testing.T) {
	top := []Pair[*testNode]{
		{Key: "Resources"},
		{Key: "Parameters"},
	}
	if !IsTemplate(top, "") {
		t.Fatalf("expected Resources+Parameters to classify as CloudFormation")
	}
}

func TestExtractSymbolsIncludesSectionsResourcesAndRefs(t *testing.T) {
	envName := scalar("EnvName")
	appBucket := scalar("AppBucket")

	parametersObj := object(pair("EnvName", object(pair("Type", scalar("String")))))
	resourceObj := object(
		pair("Type", scalar("AWS::S3::Bucket")),
		pair("Properties", objectText("BucketName: !Sub \"${EnvName}-app\"")),
	)
	resourcesObj := object(pair("AppBucket", resourceObj))
	outputsObj := object(pair("BucketName", object(pair("Value", object(pair("Ref", appBucket))))))

	topPairs := []Pair[*testNode]{
		{Node: pair("Parameters", parametersObj), Key: "Parameters", Value: parametersObj},
		{Node: pair("Resources", resourcesObj), Key: "Resources", Value: resourcesObj},
		{Node: pair("Outputs", outputsObj), Key: "Outputs", Value: outputsObj},
	}

	var emitted []core.Symbol
	op := Ops[*testNode]{
		IsZero: func(n *testNode) bool { return n == nil },
		PairsFromValue: func(n *testNode) []Pair[*testNode] {
			if n == nil {
				return nil
			}
			out := make([]Pair[*testNode], 0, len(n.objectPairs))
			for _, p := range n.objectPairs {
				out = append(out, Pair[*testNode]{Node: p, Key: p.key, Value: p.value})
			}
			return out
		},
		ScalarValue: func(n *testNode) string {
			if n == nil {
				return ""
			}
			return n.scalar
		},
		NodeText: func(n *testNode) string {
			if n == nil {
				return ""
			}
			return n.text
		},
		NamedChildren: func(n *testNode) []*testNode {
			if n == nil {
				return nil
			}
			return n.children
		},
		MakeSymbol: func(_ *testNode, name, qualified, kind string) core.Symbol {
			return core.Symbol{Name: name, QualifiedName: qualified, Kind: kind}
		},
		Emit: func(sym core.Symbol, parent string) {
			sym.ParentID = parent
			emitted = append(emitted, sym)
		},
	}

	ExtractSymbols("template:test.yaml", topPairs, op)

	mustHave := map[string]bool{
		"section:Resources":                          false,
		"parameter:Parameters.EnvName":               false,
		"resource:Resources.AppBucket":               false,
		"output:Outputs.BucketName":                  false,
		"reference:Resources.AppBucket.ref.EnvName":  false,
		"reference:Outputs.BucketName.ref.AppBucket": false,
	}
	for _, sym := range emitted {
		key := sym.Kind + ":" + sym.QualifiedName
		if _, ok := mustHave[key]; ok {
			mustHave[key] = true
		}
	}
	for key, found := range mustHave {
		if !found {
			t.Fatalf("missing %s in emitted symbols: %+v", key, emitted)
		}
	}

	_ = envName
}

func scalar(v string) *testNode {
	return &testNode{scalar: v, text: v}
}

func pair(key string, value *testNode) *testNode {
	return &testNode{key: key, value: value, text: key + ":"}
}

func object(pairs ...*testNode) *testNode {
	return &testNode{objectPairs: pairs}
}

func objectText(text string, pairs ...*testNode) *testNode {
	return &testNode{text: text, objectPairs: pairs}
}
