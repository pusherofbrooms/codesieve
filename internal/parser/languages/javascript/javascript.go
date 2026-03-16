package javascript

import (
	"github.com/pusherofbrooms/codesieve/internal/parser/core"
	"github.com/pusherofbrooms/codesieve/internal/parser/languages/tsjs"
	tsjavascript "github.com/pusherofbrooms/codesieve/internal/tslang/javascript"
	treesitter "github.com/tree-sitter/go-tree-sitter"
)

const Name = "javascript"

var Extensions = []string{".js", ".jsx", ".mjs", ".cjs"}

func Parse(_ string, content []byte) ([]core.Symbol, error) {
	return tsjs.Parse(content, treesitter.NewLanguage(tsjavascript.Language()))
}
