package typescript

import (
	"strings"

	"github.com/pusherofbrooms/codesieve/internal/parser/core"
	tstypescript "github.com/pusherofbrooms/codesieve/internal/tslang/typescript"
	treesitter "github.com/tree-sitter/go-tree-sitter"
)

const Name = "typescript"

var Extensions = []string{".ts", ".tsx"}

func Parse(path string, content []byte) ([]core.Symbol, error) {
	language := treesitter.NewLanguage(tstypescript.LanguageTypescript())
	if strings.EqualFold(pathExt(path), ".tsx") {
		language = treesitter.NewLanguage(tstypescript.LanguageTSX())
	}
	return parseTSJSTreeSitter(content, language)
}

func pathExt(path string) string {
	for i := len(path) - 1; i >= 0; i-- {
		if path[i] == '.' {
			return path[i:]
		}
		if path[i] == '/' {
			break
		}
	}
	return ""
}

func parseTSJSTreeSitter(content []byte, language *treesitter.Language) ([]core.Symbol, error) {
	return core.ParseWithTreeSitter(content, language, func(root *treesitter.Node) []core.Symbol {
		var symbols []core.Symbol
		var walk func(node *treesitter.Node, className string)
		walk = func(node *treesitter.Node, className string) {
			if node == nil {
				return
			}
			switch node.Kind() {
			case "export_statement", "statement_block", "program", "class_body":
				for i := uint(0); i < node.NamedChildCount(); i++ {
					walk(node.NamedChild(i), className)
				}
				return
			case "class_declaration":
				nameNode := node.ChildByFieldName("name")
				if nameNode != nil {
					name := core.NodeText(nameNode, content)
					symbols = append(symbols, core.MakeSymbol(content, node, name, name, "class"))
					walk(node.ChildByFieldName("body"), name)
					return
				}
			case "function_declaration", "generator_function_declaration":
				nameNode := node.ChildByFieldName("name")
				if nameNode != nil {
					name := core.NodeText(nameNode, content)
					sym := core.MakeSymbol(content, node, name, name, "function")
					sym.Signature = core.SignatureFromNode(node, content)
					symbols = append(symbols, sym)
				}
				return
			case "method_definition":
				nameNode := node.ChildByFieldName("name")
				if nameNode != nil {
					name := core.NodeText(nameNode, content)
					qualified := name
					parent := ""
					if className != "" {
						qualified = className + "." + name
						parent = className
					}
					sym := core.MakeSymbol(content, node, name, qualified, "method")
					sym.ParentID = parent
					sym.Signature = core.SignatureFromNode(node, content)
					symbols = append(symbols, sym)
				}
				return
			case "interface_declaration":
				core.AppendNamedNode(&symbols, node, content, "name", "interface")
				return
			case "type_alias_declaration":
				core.AppendNamedNode(&symbols, node, content, "name", "type")
				return
			case "enum_declaration":
				core.AppendNamedNode(&symbols, node, content, "name", "enum")
				return
			case "lexical_declaration", "variable_declaration":
				for i := uint(0); i < node.NamedChildCount(); i++ {
					decl := node.NamedChild(i)
					if decl == nil || decl.Kind() != "variable_declarator" {
						continue
					}
					nameNode := decl.ChildByFieldName("name")
					valueNode := decl.ChildByFieldName("value")
					if nameNode == nil || valueNode == nil {
						continue
					}
					emitAsFunction := false
					switch valueNode.Kind() {
					case "arrow_function", "function_expression", "generator_function":
						emitAsFunction = true
					case "call_expression":
						emitAsFunction = callExpressionHasFunctionArg(valueNode)
					}
					if emitAsFunction {
						name := core.NodeText(nameNode, content)
						sym := core.MakeSymbol(content, decl, name, name, "function")
						sym.Signature = core.SignatureFromNode(decl, content)
						symbols = append(symbols, sym)
					}
				}
				return
			}
			for i := uint(0); i < node.NamedChildCount(); i++ {
				walk(node.NamedChild(i), className)
			}
		}
		walk(root, "")
		core.SortSymbols(symbols)
		return symbols
	})
}

func callExpressionHasFunctionArg(node *treesitter.Node) bool {
	if node == nil || node.Kind() != "call_expression" {
		return false
	}
	args := node.ChildByFieldName("arguments")
	if args == nil {
		return false
	}
	for i := uint(0); i < args.NamedChildCount(); i++ {
		arg := args.NamedChild(i)
		if arg == nil {
			continue
		}
		switch arg.Kind() {
		case "arrow_function", "function_expression", "generator_function":
			return true
		case "parenthesized_expression":
			for j := uint(0); j < arg.NamedChildCount(); j++ {
				inner := arg.NamedChild(j)
				if inner == nil {
					continue
				}
				switch inner.Kind() {
				case "arrow_function", "function_expression", "generator_function":
					return true
				}
			}
		}
	}
	return false
}
