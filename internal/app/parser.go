package app

import (
	"fmt"
	"go/ast"
	"sort"
	"strings"

	treesitter "github.com/tree-sitter/go-tree-sitter"
)

func parseWithTreeSitter(content []byte, language *treesitter.Language, extract func(root *treesitter.Node) []Symbol) ([]Symbol, error) {
	parser := treesitter.NewParser()
	defer parser.Close()
	if err := parser.SetLanguage(language); err != nil {
		return nil, fmt.Errorf("set tree-sitter language: %w", err)
	}
	tree := parser.Parse(content, nil)
	if tree == nil {
		return nil, fmt.Errorf("failed to parse source")
	}
	defer tree.Close()
	root := tree.RootNode()
	if root == nil {
		return nil, fmt.Errorf("failed to parse source")
	}
	return extract(root), nil
}

func appendNamedNode(symbols *[]Symbol, node *treesitter.Node, content []byte, fieldName, kind string) {
	nameNode := node.ChildByFieldName(fieldName)
	if nameNode == nil {
		return
	}
	name := nodeText(nameNode, content)
	sym := makeSymbol(content, node, name, name, kind)
	sym.Signature = signatureFromNode(node, content)
	*symbols = append(*symbols, sym)
}

func makeSymbol(content []byte, node *treesitter.Node, name, qualifiedName, kind string) Symbol {
	start := node.StartPosition()
	end := node.EndPosition()
	return Symbol{
		Name:          name,
		QualifiedName: qualifiedName,
		Kind:          kind,
		Signature:     signatureFromNode(node, content),
		StartLine:     int(start.Row) + 1,
		EndLine:       int(end.Row) + 1,
		StartByte:     int(node.StartByte()),
		EndByte:       int(node.EndByte()),
	}
}

func signatureFromNode(node *treesitter.Node, content []byte) string {
	if node == nil {
		return ""
	}
	text := nodeText(node, content)
	if idx := strings.IndexByte(text, '\n'); idx >= 0 {
		text = text[:idx]
	}
	text = strings.TrimSpace(text)
	text = strings.TrimSuffix(text, "{")
	return strings.TrimSpace(text)
}

func pythonSignature(node *treesitter.Node, content []byte) string {
	text := signatureFromNode(node, content)
	return strings.TrimSuffix(text, ":")
}

func nodeText(node *treesitter.Node, content []byte) string {
	if node == nil {
		return ""
	}
	return strings.TrimSpace(node.Utf8Text(content))
}

func sortSymbols(symbols []Symbol) {
	sort.Slice(symbols, func(i, j int) bool {
		if symbols[i].StartLine == symbols[j].StartLine {
			return symbols[i].Name < symbols[j].Name
		}
		return symbols[i].StartLine < symbols[j].StartLine
	})
}

func renderGoFuncSignature(fn *ast.FuncDecl) string {
	params := []string{}
	if fn.Type.Params != nil {
		for _, p := range fn.Type.Params.List {
			t := exprString(p.Type)
			if len(p.Names) == 0 {
				params = append(params, t)
				continue
			}
			for _, n := range p.Names {
				params = append(params, n.Name+" "+t)
			}
		}
	}
	returns := []string{}
	if fn.Type.Results != nil {
		for _, r := range fn.Type.Results.List {
			t := exprString(r.Type)
			if len(r.Names) == 0 {
				returns = append(returns, t)
				continue
			}
			for _, n := range r.Names {
				returns = append(returns, n.Name+" "+t)
			}
		}
	}
	sig := fmt.Sprintf("func %s(%s)", fn.Name.Name, strings.Join(params, ", "))
	if len(returns) == 1 {
		sig += " " + returns[0]
	} else if len(returns) > 1 {
		sig += " (" + strings.Join(returns, ", ") + ")"
	}
	return sig
}

func exprString(expr ast.Expr) string {
	switch v := expr.(type) {
	case *ast.Ident:
		return v.Name
	case *ast.StarExpr:
		return "*" + exprString(v.X)
	case *ast.SelectorExpr:
		return exprString(v.X) + "." + v.Sel.Name
	case *ast.ArrayType:
		return "[]" + exprString(v.Elt)
	case *ast.MapType:
		return "map[" + exprString(v.Key) + "]" + exprString(v.Value)
	case *ast.InterfaceType:
		return "interface{}"
	default:
		return "any"
	}
}

func recvType(expr ast.Expr) string {
	if star, ok := expr.(*ast.StarExpr); ok {
		return recvType(star.X)
	}
	if ident, ok := expr.(*ast.Ident); ok {
		return ident.Name
	}
	return exprString(expr)
}
