package bash

import (
	"path/filepath"
	"regexp"
	"strings"

	"github.com/pusherofbrooms/codesieve/internal/parser/core"
	tsbash "github.com/pusherofbrooms/codesieve/internal/tslang/bash"
	treesitter "github.com/tree-sitter/go-tree-sitter"
)

const Name = "bash"

var Extensions = []string{".sh", ".bash"}

var bashConfigVarPattern = regexp.MustCompile(`^[A-Z][A-Z0-9_]*$`)

func Parse(path string, content []byte) ([]core.Symbol, error) {
	return core.ParseWithTreeSitter(content, treesitter.NewLanguage(tsbash.Language()), func(root *treesitter.Node) []core.Symbol {
		scriptName := "script:" + filepath.Base(path)
		script := core.MakeSymbol(content, root, scriptName, scriptName, "script")
		script.Signature = ""

		symbols := []core.Symbol{script}
		seenVars := map[string]struct{}{}

		var walk func(node *treesitter.Node, inFunction bool)
		walk = func(node *treesitter.Node, inFunction bool) {
			if node == nil {
				return
			}

			switch node.Kind() {
			case "function_definition":
				nameNode := node.ChildByFieldName("name")
				if nameNode != nil {
					name := core.NodeText(nameNode, content)
					sym := core.MakeSymbol(content, node, name, name, "function")
					sym.ParentID = scriptName
					symbols = append(symbols, sym)
				}
				core.WalkNamedChildren(node, func(child *treesitter.Node) {
					walk(child, true)
				})
				return
			case "variable_assignment":
				if !inFunction {
					appendBashVariableSymbol(&symbols, seenVars, node, node, content, scriptName)
				}
			case "declaration_command":
				if !inFunction && isBashGlobalDeclaration(node, content) {
					core.WalkNamedChildren(node, func(child *treesitter.Node) {
						switch child.Kind() {
						case "variable_assignment":
							appendBashVariableSymbol(&symbols, seenVars, child, node, content, scriptName)
						case "variable_name":
							appendBashVariableNameSymbol(&symbols, seenVars, core.NodeText(child, content), node, content, scriptName)
						}
					})
				}
			}

			core.WalkNamedChildren(node, func(child *treesitter.Node) {
				walk(child, inFunction)
			})
		}
		walk(root, false)
		core.SortSymbols(symbols)
		return symbols
	})
}

func appendBashVariableSymbol(symbols *[]core.Symbol, seen map[string]struct{}, assignmentNode, signatureNode *treesitter.Node, content []byte, scriptName string) {
	if assignmentNode == nil {
		return
	}
	nameNode := assignmentNode.ChildByFieldName("name")
	if nameNode == nil || nameNode.Kind() != "variable_name" {
		return
	}
	appendBashVariableNameSymbol(symbols, seen, core.NodeText(nameNode, content), signatureNode, content, scriptName)
}

func appendBashVariableNameSymbol(symbols *[]core.Symbol, seen map[string]struct{}, name string, signatureNode *treesitter.Node, content []byte, scriptName string) {
	name = strings.TrimSpace(name)
	if !bashConfigVarPattern.MatchString(name) {
		return
	}
	if _, ok := seen[name]; ok {
		return
	}
	sym := core.MakeSymbol(content, signatureNode, name, name, "variable")
	sym.ParentID = scriptName
	*symbols = append(*symbols, sym)
	seen[name] = struct{}{}
}

func isBashGlobalDeclaration(node *treesitter.Node, content []byte) bool {
	if node == nil {
		return false
	}
	text := strings.ToLower(strings.TrimSpace(core.NodeText(node, content)))
	return strings.HasPrefix(text, "export ") || strings.HasPrefix(text, "declare ") || strings.HasPrefix(text, "readonly ") || strings.HasPrefix(text, "typeset ")
}
