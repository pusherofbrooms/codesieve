package app

import (
	"fmt"
	"go/ast"
	goparser "go/parser"
	"go/token"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

var (
	pyClassRe    = regexp.MustCompile(`^(\s*)class\s+([A-Za-z_][A-Za-z0-9_]*)`)
	pyFuncRe     = regexp.MustCompile(`^(\s*)def\s+([A-Za-z_][A-Za-z0-9_]*)\s*\(([^)]*)\)\s*:`)
	tsDeclRe     = regexp.MustCompile(`^(\s*)(?:export\s+)?(?:async\s+)?function\s+([A-Za-z_$][A-Za-z0-9_$]*)\s*\(([^)]*)\)`)
	tsClassRe    = regexp.MustCompile(`^(\s*)(?:export\s+)?class\s+([A-Za-z_$][A-Za-z0-9_$]*)`)
	tsMethodRe   = regexp.MustCompile(`^(\s*)(?:async\s+)?([A-Za-z_$][A-Za-z0-9_$]*)\s*\(([^)]*)\)\s*\{?\s*$`)
	tsArrowVarRe = regexp.MustCompile(`^(\s*)(?:export\s+)?(?:const|let|var)\s+([A-Za-z_$][A-Za-z0-9_$]*)\s*=\s*(?:async\s*)?\(([^)]*)\)\s*=>`)
)

type indentFrame struct {
	indent int
	index  int
}

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
		return parsePython(path, string(content)), lang, nil
	case "typescript", "javascript":
		return parseTSJS(path, string(content), lang), lang, nil
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
	sort.Slice(symbols, func(i, j int) bool { return symbols[i].StartLine < symbols[j].StartLine })
	return symbols, nil
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

func parsePython(path, src string) []Symbol {
	lines := strings.Split(src, "\n")
	var stack []indentFrame
	var symbols []Symbol
	for i, line := range lines {
		lineNo := i + 1
		if m := pyClassRe.FindStringSubmatch(line); m != nil {
			indent := len(m[1])
			closeIndented(&symbols, &stack, indent, lineNo, 0)
			symbols = append(symbols, Symbol{Name: m[2], QualifiedName: m[2], Kind: "class", StartLine: lineNo, EndLine: lineNo, Language: "python", FilePath: path})
			stack = append(stack, indentFrame{indent: indent, index: len(symbols) - 1})
			continue
		}
		if m := pyFuncRe.FindStringSubmatch(line); m != nil {
			indent := len(m[1])
			closeIndented(&symbols, &stack, indent, lineNo, 0)
			kind := "function"
			qname := m[2]
			if len(stack) > 0 && symbols[stack[len(stack)-1].index].Kind == "class" {
				kind = "method"
				qname = symbols[stack[len(stack)-1].index].Name + "." + m[2]
			}
			symbols = append(symbols, Symbol{Name: m[2], QualifiedName: qname, Kind: kind, Signature: "def " + m[2] + "(" + strings.TrimSpace(m[3]) + ")", StartLine: lineNo, EndLine: lineNo, Language: "python", FilePath: path})
			stack = append(stack, indentFrame{indent: indent, index: len(symbols) - 1})
		}
	}
	closeAll(&symbols, &stack, len(lines))
	return symbols
}

func parseTSJS(path, src, lang string) []Symbol {
	lines := strings.Split(src, "\n")
	var stack []indentFrame
	var symbols []Symbol
	for i, line := range lines {
		lineNo := i + 1
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "//") {
			continue
		}
		if m := tsClassRe.FindStringSubmatch(line); m != nil {
			indent := len(m[1])
			closeIndented(&symbols, &stack, indent, lineNo, 0)
			symbols = append(symbols, Symbol{Name: m[2], QualifiedName: m[2], Kind: "class", StartLine: lineNo, EndLine: lineNo, Language: lang, FilePath: path})
			stack = append(stack, indentFrame{indent: indent, index: len(symbols) - 1})
			continue
		}
		if m := tsDeclRe.FindStringSubmatch(line); m != nil {
			indent := len(m[1])
			closeIndented(&symbols, &stack, indent, lineNo, 0)
			symbols = append(symbols, Symbol{Name: m[2], QualifiedName: m[2], Kind: "function", Signature: "function " + m[2] + "(" + strings.TrimSpace(m[3]) + ")", StartLine: lineNo, EndLine: lineNo, Language: lang, FilePath: path})
			stack = append(stack, indentFrame{indent: indent, index: len(symbols) - 1})
			continue
		}
		if m := tsArrowVarRe.FindStringSubmatch(line); m != nil {
			indent := len(m[1])
			closeIndented(&symbols, &stack, indent, lineNo, 0)
			symbols = append(symbols, Symbol{Name: m[2], QualifiedName: m[2], Kind: "function", Signature: "const " + m[2] + " = (" + strings.TrimSpace(m[3]) + ") =>", StartLine: lineNo, EndLine: lineNo, Language: lang, FilePath: path})
			stack = append(stack, indentFrame{indent: indent, index: len(symbols) - 1})
			continue
		}
		if m := tsMethodRe.FindStringSubmatch(line); m != nil && len(stack) > 0 && symbols[stack[len(stack)-1].index].Kind == "class" {
			indent := len(m[1])
			if indent > stack[len(stack)-1].indent {
				closeIndented(&symbols, &stack, indent, lineNo, 1)
				className := symbols[stack[len(stack)-1].index].Name
				symbols = append(symbols, Symbol{Name: m[2], QualifiedName: className + "." + m[2], Kind: "method", Signature: m[2] + "(" + strings.TrimSpace(m[3]) + ")", StartLine: lineNo, EndLine: lineNo, Language: lang, FilePath: path})
				stack = append(stack, indentFrame{indent: indent, index: len(symbols) - 1})
			}
		}
	}
	closeAll(&symbols, &stack, len(lines))
	return symbols
}

func closeIndented(symbols *[]Symbol, stack *[]indentFrame, indent, lineNo, keep int) {
	for len(*stack) > keep && (*stack)[len(*stack)-1].indent >= indent {
		idx := (*stack)[len(*stack)-1].index
		if (*symbols)[idx].EndLine < lineNo-1 {
			(*symbols)[idx].EndLine = lineNo - 1
		}
		*stack = (*stack)[:len(*stack)-1]
	}
}

func closeAll(symbols *[]Symbol, stack *[]indentFrame, finalLine int) {
	for len(*stack) > 0 {
		idx := (*stack)[len(*stack)-1].index
		if (*symbols)[idx].EndLine < finalLine {
			(*symbols)[idx].EndLine = finalLine
		}
		*stack = (*stack)[:len(*stack)-1]
	}
}
