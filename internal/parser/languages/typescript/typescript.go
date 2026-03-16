package typescript

import (
	"strings"

	"github.com/pusherofbrooms/codesieve/internal/parser/core"
	"github.com/pusherofbrooms/codesieve/internal/parser/languages/tsjs"
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
	return tsjs.Parse(content, language)
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
