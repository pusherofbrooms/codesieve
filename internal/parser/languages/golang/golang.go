package golang

import (
	"go/ast"
	goparser "go/parser"
	"go/token"
	"strings"

	"github.com/pusherofbrooms/codesieve/internal/parser/core"
	"github.com/pusherofbrooms/codesieve/internal/parser/languages/register"
)

const Name = "go"

var Extensions = []string{".go"}

func init() {
	register.MustRegister(Name, Parse)
}

func Parse(path string, content []byte) ([]core.Symbol, error) {
	fset := token.NewFileSet()
	file, err := goparser.ParseFile(fset, path, content, goparser.ParseComments)
	if err != nil {
		return nil, err
	}
	var symbols []core.Symbol
	ast.Inspect(file, func(n ast.Node) bool {
		switch node := n.(type) {
		case *ast.FuncDecl:
			start := fset.Position(node.Pos())
			end := fset.Position(node.End())
			kind := "function"
			qualified := node.Name.Name
			parent := ""
			sig := core.RenderGoFuncSignature(node)
			if node.Recv != nil && len(node.Recv.List) > 0 {
				kind = "method"
				recv := core.RecvType(node.Recv.List[0].Type)
				qualified = recv + "." + node.Name.Name
				parent = recv
			}
			symbols = append(symbols, core.Symbol{
				Name:          node.Name.Name,
				QualifiedName: qualified,
				Kind:          kind,
				ParentID:      parent,
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
			symbols = append(symbols, core.Symbol{
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
					symbols = append(symbols, core.Symbol{
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
	core.SortSymbols(symbols)
	return symbols, nil
}
