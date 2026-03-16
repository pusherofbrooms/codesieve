package core

import (
	"testing"

	tsjavascript "github.com/pusherofbrooms/codesieve/internal/tslang/javascript"
	treesitter "github.com/tree-sitter/go-tree-sitter"
)

func TestQualifiedNameFromContainer(t *testing.T) {
	qualified, parent := QualifiedNameFromContainer("Client", "login")
	if qualified != "Client.login" || parent != "Client" {
		t.Fatalf("unexpected qualified/parent with container: %q %q", qualified, parent)
	}

	qualified, parent = QualifiedNameFromContainer("", "login")
	if qualified != "login" || parent != "" {
		t.Fatalf("unexpected qualified/parent without container: %q %q", qualified, parent)
	}
}

func TestIsFunctionLikeValueNode(t *testing.T) {
	gotByName := parseJSFunctionLikeDeclaratorValues(t, []byte(`
const arrow = () => 1
const fnExpr = function() { return 1 }
const wrapped = lazy(() => createRoutes())
const plain = value
`))

	if !gotByName["arrow"] {
		t.Fatalf("expected arrow declarator value to be function-like")
	}
	if !gotByName["fnExpr"] {
		t.Fatalf("expected function-expression declarator value to be function-like")
	}
	if !gotByName["wrapped"] {
		t.Fatalf("expected wrapped call declarator value to be function-like")
	}
	if gotByName["plain"] {
		t.Fatalf("expected plain declarator value not to be function-like")
	}
}

func parseJSFunctionLikeDeclaratorValues(t *testing.T, content []byte) map[string]bool {
	t.Helper()
	parser := treesitter.NewParser()
	defer parser.Close()
	if err := parser.SetLanguage(treesitter.NewLanguage(tsjavascript.Language())); err != nil {
		t.Fatalf("SetLanguage: %v", err)
	}
	tree := parser.Parse(content, nil)
	if tree == nil {
		t.Fatalf("Parse returned nil tree")
	}
	defer tree.Close()
	root := tree.RootNode()
	if root == nil {
		t.Fatalf("Parse returned nil root node")
	}

	out := map[string]bool{}
	var walk func(node *treesitter.Node)
	walk = func(node *treesitter.Node) {
		if node == nil {
			return
		}
		if node.Kind() == "variable_declarator" {
			nameNode := node.ChildByFieldName("name")
			valueNode := node.ChildByFieldName("value")
			if nameNode != nil && valueNode != nil {
				out[NodeText(nameNode, content)] = IsFunctionLikeValueNode(valueNode)
			}
		}
		for i := uint(0); i < node.NamedChildCount(); i++ {
			walk(node.NamedChild(i))
		}
	}
	walk(root)
	return out
}
