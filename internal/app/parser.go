package app

import (
	"fmt"
	"go/ast"
	goparser "go/parser"
	"go/token"
	"path/filepath"
	"sort"
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
	tsjavascript "github.com/smacker/go-tree-sitter/javascript"
	tspython "github.com/smacker/go-tree-sitter/python"
	tstypescript "github.com/smacker/go-tree-sitter/typescript/typescript"
)

func DetectLanguage(path string) string {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".go":
		return "go"
	case ".py":
		return "python"
	case ".ts", ".tsx":
		return "typescript"
	case ".js", ".jsx", ".mjs", ".cjs":
		return "javascript"
	default:
		return ""
	}
}

func ParseSymbols(path string, content []byte) ([]Symbol, string, error) {
	lang := DetectLanguage(path)
	switch lang {
	case "go":
		syms, err := parseGo(path, content)
		return syms, lang, err
	case "python":
		syms, err := parsePythonTreeSitter(content)
		return syms, lang, err
	case "typescript":
		syms, err := parseTSJSTreeSitter(content, tstypescript.GetLanguage())
		return syms, lang, err
	case "javascript":
		syms, err := parseTSJSTreeSitter(content, tsjavascript.GetLanguage())
		return syms, lang, err
	default:
		return nil, "", nil
	}
}

func parseGo(path string, content []byte) ([]Symbol, error) {
	fset := token.NewFileSet()
	file, err := goparser.ParseFile(fset, path, content, goparser.ParseComments)
	if err != nil {
		return nil, err
	}
	var symbols []Symbol
	ast.Inspect(file, func(n ast.Node) bool {
		switch node := n.(type) {
		case *ast.FuncDecl:
			start := fset.Position(node.Pos())
			end := fset.Position(node.End())
			kind := "function"
			qualified := node.Name.Name
			sig := renderGoFuncSignature(node)
			if node.Recv != nil && len(node.Recv.List) > 0 {
				kind = "method"
				recv := recvType(node.Recv.List[0].Type)
				qualified = recv + "." + node.Name.Name
			}
			symbols = append(symbols, Symbol{
				Name:          node.Name.Name,
				QualifiedName: qualified,
				Kind:          kind,
				Signature:     sig,
				StartLine:     start.Line,
				EndLine:       end.Line,
				StartByte:     start.Offset,
				EndByte:       end.Offset,
			})
			return false
		case *ast.TypeSpec:
			start := fset.Position(node.Pos())
			end := fset.Position(node.End())
			kind := "type"
			switch node.Type.(type) {
			case *ast.InterfaceType:
				kind = "interface"
			case *ast.StructType:
				kind = "struct"
			}
			symbols = append(symbols, Symbol{
				Name:          node.Name.Name,
				QualifiedName: node.Name.Name,
				Kind:          kind,
				Signature:     "type " + node.Name.Name,
				StartLine:     start.Line,
				EndLine:       end.Line,
				StartByte:     start.Offset,
				EndByte:       end.Offset,
			})
			return false
		case *ast.GenDecl:
			if node.Tok != token.CONST && node.Tok != token.VAR {
				return true
			}
			for _, spec := range node.Specs {
				vs, ok := spec.(*ast.ValueSpec)
				if !ok {
					continue
				}
				for _, name := range vs.Names {
					start := fset.Position(name.Pos())
					end := fset.Position(vs.End())
					symbols = append(symbols, Symbol{
						Name:          name.Name,
						QualifiedName: name.Name,
						Kind:          strings.ToLower(node.Tok.String()),
						Signature:     strings.ToLower(node.Tok.String()) + " " + name.Name,
						StartLine:     start.Line,
						EndLine:       end.Line,
						StartByte:     start.Offset,
						EndByte:       end.Offset,
					})
				}
			}
		}
		return true
	})
	sortSymbols(symbols)
	return symbols, nil
}

func parsePythonTreeSitter(content []byte) ([]Symbol, error) {
	return parseWithTreeSitter(content, tspython.GetLanguage(), func(root *sitter.Node) []Symbol {
		var symbols []Symbol
		var walk func(node *sitter.Node, container string)
		walk = func(node *sitter.Node, container string) {
			if node == nil || node.IsNull() {
				return
			}
			switch node.Type() {
			case "decorated_definition":
				for i := 0; i < int(node.NamedChildCount()); i++ {
					child := node.NamedChild(i)
					if child != nil && child.Type() != "decorator" {
						walk(child, container)
					}
				}
				return
			case "class_definition":
				nameNode := node.ChildByFieldName("name")
				if nameNode != nil {
					name := nodeText(nameNode, content)
					symbols = append(symbols, makeSymbol(content, node, name, name, "class"))
					walk(node.ChildByFieldName("body"), name)
					return
				}
			case "function_definition", "async_function_definition":
				nameNode := node.ChildByFieldName("name")
				if nameNode != nil {
					name := nodeText(nameNode, content)
					kind := "function"
					qualified := name
					if container != "" {
						kind = "method"
						qualified = container + "." + name
					}
					sym := makeSymbol(content, node, name, qualified, kind)
					sym.Signature = pythonSignature(node, content)
					symbols = append(symbols, sym)
					walk(node.ChildByFieldName("body"), "")
					return
				}
			}
			for i := 0; i < int(node.NamedChildCount()); i++ {
				walk(node.NamedChild(i), container)
			}
		}
		walk(root, "")
		sortSymbols(symbols)
		return symbols
	})
}

func parseTSJSTreeSitter(content []byte, language *sitter.Language) ([]Symbol, error) {
	return parseWithTreeSitter(content, language, func(root *sitter.Node) []Symbol {
		var symbols []Symbol
		var walk func(node *sitter.Node, className string)
		walk = func(node *sitter.Node, className string) {
			if node == nil || node.IsNull() {
				return
			}
			switch node.Type() {
			case "export_statement", "statement_block", "program", "class_body":
				for i := 0; i < int(node.NamedChildCount()); i++ {
					walk(node.NamedChild(i), className)
				}
				return
			case "class_declaration":
				nameNode := node.ChildByFieldName("name")
				if nameNode != nil {
					name := nodeText(nameNode, content)
					symbols = append(symbols, makeSymbol(content, node, name, name, "class"))
					walk(node.ChildByFieldName("body"), name)
					return
				}
			case "function_declaration", "generator_function_declaration":
				nameNode := node.ChildByFieldName("name")
				if nameNode != nil {
					name := nodeText(nameNode, content)
					sym := makeSymbol(content, node, name, name, "function")
					sym.Signature = signatureFromNode(node, content)
					symbols = append(symbols, sym)
				}
				return
			case "method_definition":
				nameNode := node.ChildByFieldName("name")
				if nameNode != nil {
					name := nodeText(nameNode, content)
					qualified := name
					if className != "" {
						qualified = className + "." + name
					}
					sym := makeSymbol(content, node, name, qualified, "method")
					sym.Signature = signatureFromNode(node, content)
					symbols = append(symbols, sym)
				}
				return
			case "interface_declaration":
				appendNamedNode(&symbols, node, content, "name", "interface")
				return
			case "type_alias_declaration":
				appendNamedNode(&symbols, node, content, "name", "type")
				return
			case "enum_declaration":
				appendNamedNode(&symbols, node, content, "name", "enum")
				return
			case "lexical_declaration", "variable_declaration":
				for i := 0; i < int(node.NamedChildCount()); i++ {
					decl := node.NamedChild(i)
					if decl == nil || decl.Type() != "variable_declarator" {
						continue
					}
					nameNode := decl.ChildByFieldName("name")
					valueNode := decl.ChildByFieldName("value")
					if nameNode == nil || valueNode == nil {
						continue
					}
					switch valueNode.Type() {
					case "arrow_function", "function_expression", "generator_function":
						name := nodeText(nameNode, content)
						sym := makeSymbol(content, decl, name, name, "function")
						sym.Signature = signatureFromNode(decl, content)
						symbols = append(symbols, sym)
					}
				}
				return
			}
			for i := 0; i < int(node.NamedChildCount()); i++ {
				walk(node.NamedChild(i), className)
			}
		}
		walk(root, "")
		sortSymbols(symbols)
		return symbols
	})
}

func parseWithTreeSitter(content []byte, language *sitter.Language, extract func(root *sitter.Node) []Symbol) ([]Symbol, error) {
	parser := sitter.NewParser()
	defer parser.Close()
	parser.SetLanguage(language)
	tree := parser.Parse(nil, content)
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

func appendNamedNode(symbols *[]Symbol, node *sitter.Node, content []byte, fieldName, kind string) {
	nameNode := node.ChildByFieldName(fieldName)
	if nameNode == nil {
		return
	}
	name := nodeText(nameNode, content)
	sym := makeSymbol(content, node, name, name, kind)
	sym.Signature = signatureFromNode(node, content)
	*symbols = append(*symbols, sym)
}

func makeSymbol(content []byte, node *sitter.Node, name, qualifiedName, kind string) Symbol {
	start := node.StartPoint()
	end := node.EndPoint()
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

func signatureFromNode(node *sitter.Node, content []byte) string {
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

func pythonSignature(node *sitter.Node, content []byte) string {
	text := signatureFromNode(node, content)
	return strings.TrimSuffix(text, ":")
}

func nodeText(node *sitter.Node, content []byte) string {
	if node == nil {
		return ""
	}
	return strings.TrimSpace(node.Content(content))
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
